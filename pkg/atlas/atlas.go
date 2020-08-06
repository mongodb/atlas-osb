package atlas

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// Client is an interface for interacting with the Atlas API.
type Client interface {
	GetProvider(name string) (*Provider, error)
}

// HTTPClient is the main implementation of the Client interface which
// communicates with the Atlas API.
type HTTPClient struct {
	BaseURL string
	HTTP    *http.Client
}

// Different errors the api may return.
var (
	ErrUnauthorized = errors.New("invalid API key")
)

const (
	privateAPIPath = "/api/private/unauth"
)

// NewClient will create a new HTTPClient with the specified connection details.
func NewClient(baseURL string, groupID string, publicKey string, privateKey string) *HTTPClient {
	return &HTTPClient{
		BaseURL: baseURL,
		HTTP:    &http.Client{},
	}
}

// requestPrivate will make a request to an endpoint in the private API.
func (c *HTTPClient) requestPrivate(method string, endpoint string, body interface{}, response interface{}) error {
	url := fmt.Sprintf("%s%s/%s", c.BaseURL, privateAPIPath, endpoint)
	return c.request(method, url, body, response)
}

// request makes an HTTP request using the specified method.
// If body is passed it will be JSON encoded and included with the request.
// If the request was successful the response will be decoded into response.
func (c *HTTPClient) request(method string, url string, body interface{}, response interface{}) error {
	var data io.Reader

	// Construct the JSON payload if a body has been passed
	if body != nil {
		json, err := json.Marshal(body)
		if err != nil {
			return err
		}

		data = bytes.NewBuffer(json)
	}

	// Prepare API request.
	req, err := http.NewRequest(method, url, data)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	// Perform HTTP request.
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Decode response if request was successful.
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if response != nil {
			err = json.NewDecoder(resp.Body).Decode(response)

			// EOF error means the response body was empty.
			if err != io.EOF {
				return err
			}
		}

		return nil
	}

	// Invalid credentials will cause a 401 Unauthorized response.
	if resp.StatusCode == http.StatusUnauthorized {
		return ErrUnauthorized
	}

	// Decode error if request was unsuccessful.
	var errorResponse struct {
		Code        string `json:"errorCode"`
		Description string `json:"detail"`
	}
	err = json.NewDecoder(resp.Body).Decode(&errorResponse)
	if err != nil {
		return err
	}

	return fmt.Errorf("atlas error: [%s] %s", errorResponse.Code, errorResponse.Description)
}
