package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type apiErrorResponse struct {
	Message string `json:"message"`
}

func extractErrorMessage(data []byte) string {
	var e apiErrorResponse
	if err := json.Unmarshal(data, &e); err == nil && e.Message != "" {
		return e.Message
	}
	return string(data)
}

type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func New(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: DefaultBaseURL,
		httpClient: &http.Client{
			Timeout: DefaultHTTPTimeout,
		},
	}
}

func NewWithBaseURL(apiKey, baseURL string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: DefaultHTTPTimeout,
		},
	}
}

func (c *Client) post(path string, body any) ([]byte, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.baseURL+path, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", c.apiKey)

	return c.do(req)
}

func (c *Client) get(path string) ([]byte, error) {
	req, err := http.NewRequest("GET", c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-API-KEY", c.apiKey)

	return c.do(req)
}

func (c *Client) do(req *http.Request) ([]byte, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return nil, fmt.Errorf("authentication failed: check your API key with 'cast login'")
		case http.StatusForbidden:
			return nil, fmt.Errorf("access forbidden: your API key does not have permission")
		case http.StatusNotFound:
			return nil, fmt.Errorf("not found: %s", extractErrorMessage(data))
		case http.StatusBadRequest:
			return nil, fmt.Errorf("invalid request: %s", extractErrorMessage(data))
		case http.StatusTooManyRequests:
			return nil, fmt.Errorf("rate limit exceeded: please try again later")
		default:
			if resp.StatusCode >= 500 {
				return nil, fmt.Errorf("server error (%d): please try again later", resp.StatusCode)
			}
			return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, extractErrorMessage(data))
		}
	}

	return data, nil
}
