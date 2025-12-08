package provider

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// Error represents a error from the bitbucket api.
type Error struct {
	APIError struct {
		Message string `json:"message,omitempty"`
	} `json:"error,omitempty"`
	Type       string `json:"type,omitempty"`
	StatusCode int
	Endpoint   string
}

func (e Error) Error() string {
	return fmt.Sprintf("API Error: %d %s %s", e.StatusCode, e.Endpoint, e.APIError.Message)
}

const (
	// MapBoxEndpoint is the fqdn used to talk to bitbucket
	MapBoxEndpoint string = "https://api.mapbox.com/"
)

// Client is the base internal Client to talk to bitbuckets API. This should be a username and password
// the password should be a app-password.
type Client struct {
	AccessToken *string
	HTTPClient  *http.Client
}

var errNoResponse = errors.New("no response returned from API")

// Do Will just call the bitbucket api but also add auth to it and some extra headers
func (c *Client) Do(method, endpoint string, payload *bytes.Buffer, contentType string) (*http.Response, error) {
	absoluteendpoint := MapBoxEndpoint + endpoint

	client := c.httpClient()
	req, err := c.buildRequest(method, absoluteendpoint, payload, contentType)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}

	if resp == nil {
		return nil, fmt.Errorf("%w for %s %s", errNoResponse, method, absoluteendpoint)
	}

	if err := c.checkAPIError(resp, endpoint); err != nil {
		return resp, err
	}

	return resp, nil
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}

	return http.DefaultClient
}

func (c *Client) buildRequest(method, absoluteendpoint string, payload *bytes.Buffer, contentType string) (*http.Request, error) {
	var bodyreader io.Reader
	if payload != nil {
		bodyreader = payload
	}

	req, err := http.NewRequest(method, absoluteendpoint, bodyreader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if c.AccessToken != nil {
		q := req.URL.Query()
		q.Add("access_token", *c.AccessToken)
		req.URL.RawQuery = q.Encode()
	}

	if payload != nil && contentType != "" {
		req.Header.Add("Content-Type", contentType)
	}

	req.Close = true
	return req, nil
}

func (c *Client) checkAPIError(resp *http.Response, endpoint string) error {
	if resp.StatusCode < 400 && resp.StatusCode >= 200 {
		return nil
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	apiError := Error{
		StatusCode: resp.StatusCode,
		Endpoint:   endpoint,
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read error response: %w", err)
	}

	err = json.Unmarshal(body, &apiError)
	if err != nil {
		apiError.APIError.Message = string(body)
	}

	return apiError
}

// Get is just a helper method to do but with a GET verb
func (c *Client) Get(endpoint string) (*http.Response, error) {
	return c.Do("GET", endpoint, nil, "application/json")
}

// Post is just a helper method to do but with a POST verb
func (c *Client) Post(endpoint string, jsonpayload *bytes.Buffer) (*http.Response, error) {
	return c.Do("POST", endpoint, jsonpayload, "application/json")
}

// Post is just a helper method to do but with a PATCH verb
func (c *Client) Patch(endpoint string, jsonpayload *bytes.Buffer) (*http.Response, error) {
	return c.Do("PATCH", endpoint, jsonpayload, "application/json")
}

// Put is just a helper method to do but with a PUT verb
func (c *Client) Put(endpoint string, jsonpayload *bytes.Buffer) (*http.Response, error) {
	return c.Do("PUT", endpoint, jsonpayload, "application/json")
}

// PutOnly is just a helper method to do but with a PUT verb and a nil body
func (c *Client) PutOnly(endpoint string) (*http.Response, error) {
	return c.Do("PUT", endpoint, nil, "application/json")
}

// Delete is just a helper to Do but with a DELETE verb
func (c *Client) Delete(endpoint string) (*http.Response, error) {
	return c.Do("DELETE", endpoint, nil, "application/json")
}
