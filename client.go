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

type Response struct {
	// MulticastID is the unique ID identifying the multicast message.
	MulticastID int64 `json:"multicast_id"`

	// Success is the number of messages that were processed without an error.
	Success int `json:"success"`

	// Failure is the number of messages that could not be processed.
	Failure int `json:"failure"`

	// CanonicalIDs is the number of results that contain a canonical
	// registration token. A canonical registration ID is the registration
	// token of the last registration requested by the client app.
	// This is the ID that the server should use when sending messages to the device.
	CanonicalIDs int `json:"canonical_ids"`

	// Results is an array of objects representing the status of the messages
	// processed. The objects are listed in the same order as the request
	// (i.e., for each registration ID in the request, its result is listed in
	// the same index in the response).
	Results []MessageResult `json:"results"`

	// RetryAfter indicates when the request should be retried.
	// It is the zero value if no such hint was given.
	RetryAfter time.Duration
}

type MessageResult struct {
	// MessageID is a unique ID for each successfully processed message.
	// It is the empty string if and only if there is an error.
	MessageID string `json:"message_id"`

	// RegistrationID specifies the canonical registration token for the
	// client app that the message was processed and sent to.
	// The sender should use this value as the registration token for
	// future requests. Otherwise, the messages might be rejected.
	//
	// If the sender is already using the canonical registration token,
	// the field is empty.
	RegistrationID string `json:"registration_id"`

	// Error specifies the error that occurred when processing the message
	// for the recipient. The empty string indicates no error.
	//
	// For possible error values, see the documentation at:
	// https://firebase.google.com/docs/cloud-messaging/http-server-ref#table9
	Error string
}

type Client struct {
	apiKey string
	client *http.Client
}

func NewClient(apiKey string, client *http.Client) *Client {
	if apiKey == "" {
		panic("fcm: empty apiKey")
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &Client{apiKey: apiKey, client: client}
}

// apiURL is the API URL to use to send messages
const apiURL = "https://fcm.googleapis.com/fcm/send"

var ErrAuthenticationFailure = errors.New("fcm: authentication failure")

func (c *Client) Send(ctx context.Context, msg *Message) (*Response, error) {
	if msg == nil {
		panic("fcm: cannot send nil msg")
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("fcm: cannot marshal msg: %v", msg)
	}
	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(data))
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

	switch {
	case resp.StatusCode == http.StatusBadRequest:
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("fcm: invalid request: %s", body)

	case resp.StatusCode == http.StatusUnauthorized:
		return nil, ErrAuthenticationFailure

	case 500 <= resp.StatusCode && resp.StatusCode < 600: // 5xx error
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, &InternalError{
			RetryAfter: retryAfter,
			StatusCode: resp.StatusCode,
			Body:       string(body),
		}

	case http.StatusOK:
		var response Response
		if err := json.NewDecoder(&resp.Body).Decode(&response); err != nil {
			return nil, fmt.Errorf("fcm: could not decode response: " + err.Error())
		}
		response.RetryAfter = retryAfter
		return &response, nil

	default:
		return nil, fmt.Errorf("fcm: unexpected status code %d", resp.StatusCode)
	}
}

type InternalError struct {
	RetryAfter time.Duration
	StatusCode int
	Body       string
}
