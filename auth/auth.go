package auth

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/SermoDigital/jose/crypto"
	"github.com/SermoDigital/jose/jws"
)

func NewVerifier(ctx context.Context, projectID string, client *http.Client) *Verifier {
	if client == nil {
		client = http.DefaultClient
	}
	v := &Verifier{
		projectID: projectID,
		issuer:    "https://securetoken.google.com/" + projectID,
		client:    client,
		haveCerts: make(chan struct{}),
	}
	go v.fetchCertLoop(ctx)
	return v
}

type Verifier struct {
	projectID string
	issuer    string
	client    *http.Client

	mu        sync.RWMutex
	certs     map[string]*rsa.PublicKey
	haveCerts chan struct{} // closed when we have certs
}

type User struct {
	ID             string
	SignInProvider string
	EmailVerified  bool
	Email          string
}

func (v *Verifier) Verify(ctx context.Context, token []byte) (*User, error) {
	// Verify a signed JWT token according to spec:
	// https://firebase.google.com/docs/auth/admin/verify-id-tokens
	tok, err := jws.ParseCompact(token)
	if err != nil {
		return nil, fmt.Errorf("auth: parse error: %v", err)
	}

	// Step 1: Check alg and kid
	if alg, ok := tok.Protected().Get("alg").(string); !ok || alg != "RS256" {
		return nil, fmt.Errorf("auth: alg is %s, not RS256", alg)
	}
	kid, ok := tok.Protected().Get("kid").(string)
	if !ok {
		return nil, errors.New("auth: invalid kid")
	}

	v.mu.RLock()
	if len(v.certs) == 0 {
		// We don't have any certs; release the lock
		// and wait for certs
		v.mu.RUnlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-v.haveCerts:
			// We have certs; grab the lock again and continue
			v.mu.RLock()
		}
	}
	key, ok := v.certs[kid]
	v.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("auth: unknown kid: %s", kid)
	} else if err := tok.Verify(key, crypto.SigningMethodRS256); err != nil {
		return nil, fmt.Errorf("auth: verification failure: %v", err)
	}

	// Step 2: check exp, iat, aud, iss, sub, per spec.
	claimsMap, ok := tok.Payload().(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("auth: unexpected payload type: %T", tok.Payload())
	}
	claims := jws.Claims(claimsMap)
	now := time.Now()

	// exp: "Must be in the future. The time is measured in seconds since the UNIX epoch."
	if exp, ok := claims.Expiration(); !ok || exp.Before(now) {
		return nil, errors.New("auth: expired token")
	}

	// iat: "Must be in the past. The time is measured in seconds since the UNIX epoch."
	if iat, ok := claims.IssuedAt(); !ok || iat.After(now) {
		return nil, errors.New("auth: token issued in the future")
	}

	// aud: "Must be your Firebase project ID, the unique identifier for your
	// Firebase project, which can be found in the URL of that project's console."
	if aud, ok := claims.Get("aud").(string); !ok || aud != v.projectID {
		return nil, fmt.Errorf("auth: unexpected project ID (%q)", aud)
	}

	// iss: "Must be "https://securetoken.google.com/<projectId>", where <projectId>
	// is the same project ID used for aud above."
	if iss, ok := claims.Get("iss").(string); !ok || iss != v.issuer {
		return nil, fmt.Errorf("auth: unexpected issuer (%q)", iss)
	}

	// sub: "Must be a non-empty string and must be the uid of the user or device."
	sub, _ := claims.Get("sub").(string)
	userID, _ := claims.Get("user_id").(string)
	if sub == "" || userID == "" || sub != userID {
		return nil, fmt.Errorf("auth: invalid sub or user_id (%q / %q)", sub, userID)
	}

	u := new(User)
	u.ID = userID
	u.Email, _ = claims.Get("email").(string)
	u.EmailVerified, _ = claims.Get("email_verified").(bool)
	u.SignInProvider, _ = claims.Get("sign_in_provider").(string)
	return u, nil
}

func (v *Verifier) fetchCertLoop(ctx context.Context) {
	var nextFetch <-chan time.Time
	failExpiry := 1 * time.Second
	firstFetch := true
	for {
		certs, expiry, err := v.fetchCerts(ctx)
		if err == nil {
			v.mu.Lock()
			v.certs = certs
			v.mu.Unlock()

			if firstFetch && len(certs) > 0 {
				close(v.haveCerts)
				firstFetch = false
			}
		} else {
			// We got an error; check if the certs are empty; in that case,
			// set a low expiry so we try often and complain loudly.
			v.mu.RLock()
			gotCerts := len(v.certs) > 0
			v.mu.RUnlock()

			if !gotCerts {
				log.Println("auth: failed to fetch certs and no certs present -- cannot authenticate users! err:", err)
				expiry = failExpiry
				failExpiry *= 2 // exponential back-off
			}
		}

		nextFetch = time.After(expiry)

		// Return if we're done
		select {
		case <-ctx.Done():
			return
		case <-nextFetch:
		}
	}
}

var certificateURL = "https://www.googleapis.com/robot/v1/metadata/x509/securetoken@system.gserviceaccount.com"

func (v *Verifier) fetchCerts(ctx context.Context) (map[string]*rsa.PublicKey, time.Duration, error) {
	req, err := http.NewRequest("GET", certificateURL, nil)
	if err != nil {
		// Should never happen for a well-formed URL
		panic("auth: internal error: could not create request (invalid URL?): " + err.Error())
	}

	resp, err := v.client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	var certs map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&certs); err != nil {
		return nil, 0, err
	}

	parsedCerts := make(map[string]*rsa.PublicKey, len(certs))
	for key, cert := range certs {
		pk, err := crypto.ParseRSAPublicKeyFromPEM([]byte(cert))
		if err != nil {
			return nil, 0, err
		}
		parsedCerts[key] = pk
	}

	// Parse max-age into a duration, per the spec:
	// https://firebase.google.com/docs/auth/admin/verify-id-tokens
	cacheControl := resp.Header.Get("Cache-Control")
	parts := strings.Split(cacheControl, ",")
	expiry := 10 * time.Minute // arbitrary default
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "max-age=") {
			seconds, err := strconv.Atoi(part[len("max-age="):])
			if err != nil {
				log.Println("auth: could not parse max-age for JWT certificate expiry, defaulting to 10m")
				break
			}
			expiry = time.Duration(seconds) * time.Second
		}
	}

	// Sanity check against weird max-age values
	if expiry < 0 {
		expiry = 10 * time.Minute
	}
	return parsedCerts, expiry, nil
}
