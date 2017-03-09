package fcm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// apiURL is the API URL to use to send messages
const apiURL = "https://fcm.googleapis.com/fcm/send"

type Client struct {
	apiKey string
	apiURL string
	client *http.Client
}

func NewClient(apiKey string, client *http.Client) *Client {
	if apiKey == "" {
		panic("fcm: empty apiKey")
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &Client{apiKey: apiKey, apiURL: apiURL, client: client}
}

// ErrAuthenticationFailure is returned by Client.Send if the FCM server
// responds with a 401 Unauthorized.
var ErrAuthenticationFailure = errors.New("fcm: authentication failure")

func (c *Client) Send(ctx context.Context, msg *Message) (*Response, error) {
	if msg == nil {
		panic("fcm: cannot send nil msg")
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("fcm: cannot marshal msg: %v", msg)
	}
	req, err := http.NewRequest("POST", c.apiURL, bytes.NewReader(data))
	if err != nil {
		panic("fcm: internal error: invalid api URL: " + apiURL)
	}

	req.Header.Set("Authorization", "key="+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Always look for a Retry-After header, since it gets sent
	// for various response status codes.
	retryAfter, _ := time.ParseDuration(resp.Header.Get("Retry-After"))

	// Handle 5xx outside of the switch since it is a large range.
	if 500 <= resp.StatusCode && resp.StatusCode < 600 {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, &ServerError{
			RetryAfter: retryAfter,
			StatusCode: resp.StatusCode,
			Body:       string(body),
		}
	}

	switch resp.StatusCode {
	case http.StatusBadRequest:
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("fcm: invalid request: %s", body)

	case http.StatusUnauthorized:
		return nil, ErrAuthenticationFailure

	case http.StatusOK:
		var response Response
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return nil, fmt.Errorf("fcm: could not decode response: " + err.Error())
		}
		response.RetryAfter = retryAfter
		return &response, nil

	default:
		return nil, fmt.Errorf("fcm: unexpected status code %d", resp.StatusCode)
	}
}

// ServerError represents an internal error on the FCM server's side.
type ServerError struct {
	// RetryAfter, if non-zero, specifies how long to wait before making
	// the same request again.
	RetryAfter time.Duration

	// StatusCode is the HTTP status code returned by the server.
	// It is in the 5xx range.
	StatusCode int

	// Body is full the HTTP response body returned by the server.
	Body string
}

func (err *ServerError) Error() string {
	return fmt.Sprintf("fcm: server returned HTTP %d: %s", err.StatusCode, err.Body)
}
