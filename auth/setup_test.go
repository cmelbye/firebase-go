package auth

import (
	"bytes"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/SermoDigital/jose/crypto"
	"github.com/SermoDigital/jose/jws"
)

type key struct {
	pk   *rsa.PrivateKey
	id   string
	cert string
}

var validKeys []key
var invalidKey key

var testClient *http.Client

func genToken(payload map[string]interface{}, key key) []byte {
	tok := jws.New(payload, crypto.SigningMethodRS256)
	tok.Protected().Set("kid", key.id)
	serialized, err := tok.Compact(key.pk)
	if err != nil {
		panic("could not serialize token: " + err.Error())
	}
	return serialized
}

func keyHandler(w http.ResponseWriter, req *http.Request) {
	keys := make(map[string]string, len(validKeys))
	for _, key := range validKeys {
		keys[key.id] = key.cert
	}
	json.NewEncoder(w).Encode(keys)
}

func generateKeys() {
	// Use regular math/rand instead of crypto/rand to speed up the test.
	// Never do this in regular code!
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < 4; i++ {
		pk, err := rsa.GenerateKey(r, 1024)
		if err != nil {
			panic("could not generate key: " + err.Error())
		}

		// Code taken from https://golang.org/src/crypto/tls/generate_cert.go
		notBefore := time.Now()
		notAfter := notBefore.Add(1 * time.Minute)
		serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
		serialNumber, err := cryptorand.Int(r, serialNumberLimit)
		if err != nil {
			panic("failed to generate serial number: " + err.Error())
		}

		template := x509.Certificate{
			SerialNumber: serialNumber,
			Subject: pkix.Name{
				Organization: []string{"Organization"},
			},
			NotBefore: notBefore,
			NotAfter:  notAfter,

			KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
			ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			BasicConstraintsValid: true,
		}
		derBytes, err := x509.CreateCertificate(r, &template, &template, &pk.PublicKey, pk)
		if err != nil {
			panic("failed to create certificate: " + err.Error())
		}
		var buf bytes.Buffer
		pem.Encode(&buf, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
		validKeys = append(validKeys, key{
			pk:   pk,
			id:   serialNumber.String(),
			cert: buf.String(),
		})
	}

	// Take the last key and make it an invalid key
	invalidKey = validKeys[len(validKeys)-1]
	validKeys = validKeys[:len(validKeys)-1]
}

func TestMain(m *testing.M) {
	generateKeys()
	serv := httptest.NewServer(http.HandlerFunc(keyHandler))
	certificateURL = serv.URL
	code := m.Run()
	serv.Close()
	os.Exit(code)
}
