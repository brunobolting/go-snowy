// Package snowy provides a simplified HTTP client for making API requests with JSON responses.
//
// # Overview
//
// Snowy is a lightweight, type-safe HTTP client specifically designed for
// interacting with JSON APIs. It offers a clean, generics-based interface for making
// HTTP requests with automatic JSON marshalling/unmarshalling, configurable timeouts,
// and efficient connection pooling.
//
// Key Features:
//   - Type-safe requests with generics
//   - Connection pooling with automatic client caching
//   - Support for JSON and form-encoded request bodies
//   - Comprehensive error handling with custom error types
//   - Convenient helper methods for authentication
//   - Full HTTP method coverage (GET, POST, PUT, PATCH, DELETE)
//   - Custom status code handling for non-standard APIs
//
// # Basic Examples
//
// Making a GET request:
//
//	// Configure the client
//	config := snowy.Config{
//		Timeout: 5 * time.Second,
//	}
//
//	// Define your response type
//	type UserResponse struct {
//		ID    int    `json:"id"`
//		Name  string `json:"name"`
//		Email string `json:"email"`
//	}
//
//	// Make the request
//	response, err := snowy.Get[UserResponse](config, "https://api.example.com/users/1", nil, RequestData{})
//	if err != nil {
//		// Handle error
//		return err
//	}
//
//	// Access the data
//	user := response.Data
//	fmt.Println("User Name:", user.Name)
//
// Making a POST request with JSON body:
//
//	// Create request data
//	type CreateUserRequest struct {
//		Name  string `json:"name"`
//		Email string `json:"email"`
//	}
//
//	userData := CreateUserRequest{
//		Name:  "Jane Doe",
//		Email: "jane@example.com",
//	}
//
//	// Make the request
//	response, err := snowy.Post[UserResponse](
//		config,
//		"https://api.example.com/users",
//		nil,
//		snowy.BodyData{JsonData: userData},
//	)
//
// # Authentication Examples
//
// Using Bearer Token:
//
//	// Create and configure headers with authentication
//	headers := snowy.Headers{}
//	headers.AddBearer("your-token-here")
//
//	// Make authenticated request
//	response, err := snowy.Get[UserResponse](config, "https://api.example.com/users/me", headers, RequestData{})
//
// Using Basic Authentication:
//
//	// Create and configure headers with authentication
//	headers := snowy.Headers{}
//	headers.AddBasicAuth("username", "password")
//
//	// Make authenticated request
//	response, err := snowy.Get[UserResponse](config, "https://api.example.com/users/me", headers)
//
// # Working with Response Headers
//
// Accessing response headers for pagination:
//
//	response, err := snowy.Get[[]UserResponse](config, "https://api.example.com/users", nil, RequestData{})
//	if err != nil {
//		return err
//	}
//
//	// Access response headers
//	nextPageURL := response.Headers.Get("X-Next-Page")
//	totalCount := response.Headers.Get("X-Total-Count")
//
// # Handling Errors
//
// Proper error handling:
//
//	response, err := snowy.Get[UserResponse](config, "https://api.example.com/users/999", nil, RequestData{})
//	if err != nil {
//		// Check for specific error type
//		if reqErr, ok := err.(*snowy.RequestError); ok {
//			if reqErr.StatusCode == 404 {
//				// Handle not found case
//				fmt.Println("User not found")
//			} else {
//				// Handle other API errors
//				fmt.Printf("API Error: %s\n", reqErr.Message)
//				fmt.Printf("Response Data: %v\n", reqErr.Response)
//			}
//		} else {
//			// Handle other types of errors
//			fmt.Printf("Error: %v\n", err)
//		}
//		return err
//	}
//
// # Custom Status Code Handling
//
// Some APIs use non-standard status codes that you might want to treat as successful:
//
//	// Configure the client to accept 201 (Created) and 422 as valid responses
//	config := snowy.Config{
//		Timeout: 5 * time.Second,
//		AcceptableStatusCodes: []int{201, 422},
//	}
//
//	// Make a POST request that might return 201 Created
//	response, err := snowy.Post[UserResponse](
//		config,
//		"https://api.example.com/users",
//		nil,
//		snowy.BodyData{JsonData: userData},
//	)
//	if err != nil {
//		return err
//	}
//
//	// Check the actual status code if needed
//	if response.StatusCode == 422 {
//		// Handle validation issues but still process the response
//		fmt.Println("Warning: Some validation issues occurred")
//	}
//
// # Full Configuration Options
//
// Creating a fully configured client:
//
//	config := snowy.Config{
//		Ctx:                 context.Background(),
//		Timeout:             10 * time.Second,
//		MaxIdleConns:        100,
//		IdleConnTimeout:     90 * time.Second,
//		TLSHandshakeTimeout: 5 * time.Second,
//		AcceptableStatusCodes: []int{202, 207}, // Accept additional status codes that will be treated as successful
//	}
//
// # Thread Safety
//
// The package is thread-safe and can be used concurrently from multiple goroutines.
// HTTP clients are cached based on their configuration to ensure efficient connection pooling.
package snowy

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"
)

var (
	clientCache = sync.Map{}
)

func (c Config) hash() string {
	return fmt.Sprintf("%d-%d-%d-%d",
		c.Timeout.Milliseconds(),
		c.MaxIdleConns,
		c.IdleConnTimeout.Milliseconds(),
		c.TLSHandshakeTimeout.Milliseconds())
}

func getClient(config Config) *http.Client {
	hash := config.hash()
	if client, ok := clientCache.Load(hash); ok {
		return client.(*http.Client)
	}

	transport := &http.Transport{
		MaxIdleConns:        config.MaxIdleConns,
		IdleConnTimeout:     config.IdleConnTimeout,
		TLSHandshakeTimeout: config.TLSHandshakeTimeout,
	}

	client := &http.Client{
		Timeout:   config.Timeout,
		Transport: transport,
	}

	clientCache.Store(hash, client)
	return client
}

type Response[T any] struct {
	StatusCode int
	Data       *T
	Headers    http.Header
}

type Config struct {
	Ctx                   context.Context
	Timeout               time.Duration
	MaxIdleConns          int
	IdleConnTimeout       time.Duration
	TLSHandshakeTimeout   time.Duration
	AcceptableStatusCodes []int // Accept status codes that will be treated as successful
}

type RequestError struct {
	StatusCode int
	Message    string
	Response   any
}

func (e *RequestError) Error() string {
	return fmt.Sprintf("message: %s", e.Message)
}

type RequestData struct {
	QueryParams map[string]string
	JsonData 	any
	FormData 	map[string]string
}

type Headers map[string]string

func (h Headers) Add(key, value string) {
	h[key] = value
}

func (h Headers) Contains(key string) bool {
	_, ok := h[key]
	return ok
}

func (h Headers) Get(key string) string {
	return h[key]
}

func (h Headers) Remove(key string) {
	delete(h, key)
}

func (h Headers) AddBasicAuth(username, password string) {
	auth := username + ":" + password
	hash := base64.StdEncoding.EncodeToString([]byte(auth))
	h.Add("Authorization", "Basic "+hash)
}

func (h Headers) AddBearer(token string) {
	h.Add("Authorization", "Bearer "+token)
}

func doRequest[T any](config Config, method, url string, headers map[string]string, body io.Reader) (*Response[T], error) {
	if config.Ctx == nil {
		config.Ctx = context.Background()
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxIdleConns == 0 {
		config.MaxIdleConns = 100
	}
	if config.IdleConnTimeout == 0 {
		config.IdleConnTimeout = 90 * time.Second
	}
	if config.TLSHandshakeTimeout == 0 {
		config.TLSHandshakeTimeout = 10 * time.Second
	}
	req, err := http.NewRequestWithContext(config.Ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	if headers == nil {
		headers = make(map[string]string)
	}
	headers["Accept"] = "application/json"
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := getClient(config)
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer res.Body.Close()

	isAcceptable := res.StatusCode >= 200 && res.StatusCode < 300
	if slices.Contains(config.AcceptableStatusCodes, res.StatusCode) {
		isAcceptable = true
	}

	if !isAcceptable {
		bodyBytes, readErr := io.ReadAll(res.Body)
		if readErr != nil {
			return nil, fmt.Errorf("reading error response body: %w", readErr)
		}
		var parsedBody map[string]any
		if json.Unmarshal(bodyBytes, &parsedBody) == nil {
			return nil, &RequestError{
				StatusCode: res.StatusCode,
				Message:    fmt.Sprintf("unexpected status code: %d", res.StatusCode),
				Response:   parsedBody,
			}
		}

		return nil, &RequestError{
			StatusCode: res.StatusCode,
			Message:    fmt.Sprintf("unexpected status code: %d", res.StatusCode),
			Response:   string(bodyBytes), // Convert to string for better display
		}
	}

	var v T
	if err := json.NewDecoder(res.Body).Decode(&v); err != nil {
		if err == io.EOF {
			return &Response[T]{
				StatusCode: res.StatusCode,
				Data:    nil,
				Headers: res.Header,
			}, nil
		}
		return nil, fmt.Errorf("decoding response body: %w", err)
	}
	return &Response[T]{
		StatusCode: res.StatusCode,
		Data:    &v,
		Headers: res.Header,
	}, nil
}

func parseBody(body RequestData) (io.Reader, error) {
	if body.JsonData != nil {
		data, err := json.Marshal(body.JsonData)
		if err != nil {
			return nil, fmt.Errorf("marshalling JSON data: %w", err)
		}
		return bytes.NewReader(data), nil
	}
	if len(body.FormData) > 0 {
		data := url.Values{}
		for k, v := range body.FormData {
			data.Set(k, v)
		}
		return strings.NewReader(data.Encode()), nil
	}
	return nil, nil
}

func parseHeaders(headers map[string]string, body RequestData) map[string]string {
	if body.JsonData != nil {
		headers["Content-Type"] = "application/json"
	}
	if len(body.FormData) > 0 {
		headers["Content-Type"] = "application/x-www-form-urlencoded"
	}
	return headers
}

func parseQueryParams(url string, query RequestData) string {
	if len(query.QueryParams) == 0 {
		return url
	}
	params := url + "?"
	for k, v := range query.QueryParams {
		params += fmt.Sprintf("%s=%s&", k, v)
	}
	return strings.TrimSuffix(params, "&")
}

func Get[T any](config Config, url string, headers map[string]string, query RequestData) (*Response[T], error) {
	url = parseQueryParams(url, query)
	return doRequest[T](config, http.MethodGet, url, headers, nil)
}

func Post[T any](config Config, url string, headers map[string]string, body RequestData) (*Response[T], error) {
	url = parseQueryParams(url, body)
	headers = parseHeaders(headers, body)
	data, err := parseBody(body)
	if err != nil {
		return nil, err
	}
	return doRequest[T](config, http.MethodPost, url, headers, data)
}

func Put[T any](config Config, url string, headers map[string]string, body RequestData) (*Response[T], error) {
	url = parseQueryParams(url, body)

	headers = parseHeaders(headers, body)
	data, err := parseBody(body)
	if err != nil {
		return nil, err
	}
	return doRequest[T](config, http.MethodPut, url, headers, data)
}

func Patch[T any](config Config, url string, headers map[string]string, body RequestData) (*Response[T], error) {
	url = parseQueryParams(url, body)
	headers = parseHeaders(headers, body)
	data, err := parseBody(body)
	if err != nil {
		return nil, err
	}
	return doRequest[T](config, http.MethodPatch, url, headers, data)
}

func Delete[T any](config Config, url string, headers map[string]string, query RequestData) (*Response[T], error) {
	url = parseQueryParams(url, query)
	return doRequest[T](config, http.MethodDelete, url, headers, nil)
}
