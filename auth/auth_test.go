package auth

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestVerify(t *testing.T) {
	past := time.Now().Add(-1 * time.Minute).Unix()
	future := time.Now().Add(1 * time.Minute).Unix()
	const projectID = "projectID"

	var tests = []struct {
		Key     key
		Payload map[string]interface{}
		Err     string
		User    *User
	}{
		0: {
			Key: invalidKey,
			Err: "unknown kid",
		},
		1: {
			Key: validKeys[0],
			Err: "expired token",
		},
		2: {
			Key: validKeys[0],
			Payload: map[string]interface{}{
				"exp": past,
			},
			Err: "expired token",
		},
		3: {
			Key: validKeys[1],
			Payload: map[string]interface{}{
				"exp": future,
				"iat": future,
			},
			Err: "token issued in the future",
		},
		4: {
			Key: validKeys[1],
			Payload: map[string]interface{}{
				"exp": future,
				"iat": past,
				"aud": "some-other-project-id",
			},
			Err: "unexpected project ID",
		},
		5: {
			Key: validKeys[1],
			Payload: map[string]interface{}{
				"exp": future,
				"iat": past,
				"aud": projectID,
				"iss": "some-random-value",
			},
			Err: "unexpected issuer",
		},
		6: {
			Key: validKeys[2],
			Payload: map[string]interface{}{
				"exp": future,
				"iat": past,
				"aud": projectID,
				"iss": "https://securetoken.google.com/some-other-project-id",
			},
			Err: "unexpected issuer",
		},
		7: {
			Key: validKeys[0],
			Payload: map[string]interface{}{
				"exp": future,
				"iat": past,
				"aud": projectID,
				"iss": "https://securetoken.google.com/" + projectID,
			},
			Err: "invalid sub or user_id",
		},
		8: {
			Key: validKeys[0],
			Payload: map[string]interface{}{
				"exp":     future,
				"iat":     past,
				"aud":     projectID,
				"iss":     "https://securetoken.google.com/" + projectID,
				"sub":     "sub",
				"user_id": "some-other-value",
			},
			Err: "invalid sub or user_id",
		},
		9: {
			Key: validKeys[0],
			Payload: map[string]interface{}{
				"exp":     future,
				"iat":     past,
				"aud":     projectID,
				"iss":     "https://securetoken.google.com/" + projectID,
				"sub":     "sub",
				"user_id": "sub",
			},
			User: &User{
				ID: "sub",
			},
		},
		10: {
			Key: validKeys[1],
			Payload: map[string]interface{}{
				"exp":              future,
				"iat":              past,
				"aud":              projectID,
				"iss":              "https://securetoken.google.com/" + projectID,
				"sub":              "sub",
				"user_id":          "sub",
				"email":            "foo@example.com",
				"email_verified":   true,
				"sign_in_provider": "some-provider",
			},
			User: &User{
				ID:             "sub",
				Email:          "foo@example.com",
				EmailVerified:  true,
				SignInProvider: "some-provider",
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	verifier := NewVerifier(ctx, projectID, nil)
	for i, test := range tests {
		payload := test.Payload
		if payload == nil {
			payload = map[string]interface{}{}
		}
		token := genToken(payload, test.Key)
		user, err := verifier.Verify(ctx, token)

		if err == nil && test.Err != "" {
			t.Errorf("%d: got err == nil, want %q", i, test.Err)
			continue
		} else if err != nil && test.Err == "" {
			t.Errorf("%d: got err %v, want nil", i, err)
			continue
		} else if err != nil && !strings.Contains(err.Error(), test.Err) {
			t.Errorf("%d: got err %v, doesn't match %q", i, err, test.Err)
			continue
		}
		if err != nil {
			continue // no point checking *User
		}

		if test.User == nil || user == nil {
			t.Errorf("%d: want or got nil user (incorrectly written test?)", i)
			continue
		}
		if !reflect.DeepEqual(*test.User, *user) {
			t.Errorf("%d: got user %+v, want %+v", i, *user, *test.User)
		}
	}
}
