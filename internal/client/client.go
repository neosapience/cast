package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
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
	apiKey      string
	baseURL     string
	httpClient  *http.Client
	attribution string
}

func New(apiKey string) *Client {
	return &Client{
		apiKey:      apiKey,
		baseURL:     DefaultBaseURL,
		attribution: attributionFromEnv(),
		httpClient: &http.Client{
			Timeout: DefaultHTTPTimeout,
		},
	}
}

func NewWithBaseURL(apiKey, baseURL string) *Client {
	return &Client{
		apiKey:      apiKey,
		baseURL:     baseURL,
		attribution: attributionFromEnv(),
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

	return c.do(req)
}

func (c *Client) get(path string) ([]byte, error) {
	req, err := http.NewRequest("GET", c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}

	return c.do(req)
}

func (c *Client) delete(path string) ([]byte, error) {
	req, err := http.NewRequest("DELETE", c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}

	return c.do(req)
}

func (c *Client) do(req *http.Request) ([]byte, error) {
	c.setHeaders(req)
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
		case http.StatusUnprocessableEntity:
			return nil, fmt.Errorf("unprocessable request: %s", extractErrorMessage(data))
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

func (c *Client) setHeaders(req *http.Request) {
	base := "custom"
	if strings.EqualFold(strings.TrimRight(c.baseURL, "/"), DefaultBaseURL) {
		base = "default"
	}
	req.Header.Set("X-API-KEY", c.apiKey)
	req.Header.Set(
		"User-Agent",
		fmt.Sprintf(
			"typecast-cli/%s Go/%s net-http (base=%s; os=%s; arch=%s; platform=cli)",
			Version,
			strings.TrimPrefix(runtime.Version(), "go"),
			base,
			normalizedOS(runtime.GOOS),
			normalizedArch(runtime.GOARCH),
		)+c.attribution,
	)
}

func attributionSuffix(source, generatedBy string) string {
	if source != "llms" && source != "skill" && source != "api-page" && source != "api-docs" || !validGeneratedBy(generatedBy) {
		return ""
	}
	return fmt.Sprintf(" typecast-integration/1 (source=%s; generated_by=%s)", source, generatedBy)
}

func attributionFromEnv() string {
	return attributionSuffix(
		os.Getenv("TYPECAST_INTEGRATION_SOURCE"),
		os.Getenv("TYPECAST_GENERATED_BY"),
	)
}

func validGeneratedBy(value string) bool {
	if len(value) == 0 || len(value) > 32 {
		return false
	}
	for i, char := range value {
		if char >= 'a' && char <= 'z' || char >= '0' && char <= '9' || i > 0 && (char == '.' || char == '_' || char == '-') {
			continue
		}
		return false
	}
	return true
}

func normalizedOS(value string) string {
	if value == "darwin" {
		return "macos"
	}
	if value == "" {
		return "unknown"
	}
	return value
}

func normalizedArch(value string) string {
	switch value {
	case "amd64":
		return "x64"
	case "386":
		return "x86"
	case "":
		return "unknown"
	default:
		return value
	}
}
