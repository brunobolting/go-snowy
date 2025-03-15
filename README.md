# Snowy

Snowy is a lightweight, type-safe HTTP client specifically designed for interacting with JSON APIs. It offers a clean, generics-based interface for making HTTP requests with automatic JSON marshalling/unmarshalling, configurable timeouts, and efficient connection pooling.

<img alt="Coverage Badge" src="https://img.shields.io/badge/coverage-92.2%25-blue">
<a href="https://pkg.go.dev/github.com/brunobolting/go-snowy"><img src="https://pkg.go.dev/badge/github.com/brunobolting/go-snowy.svg" alt="Go Reference"></a>

## Key Features

- Type-safe requests with generics
- Connection pooling with automatic client caching
- Support for JSON and form-encoded request bodies
- Comprehensive error handling with custom error types
- Convenient helper methods for authentication
- Full HTTP method coverage (GET, POST, PUT, PATCH, DELETE)
- Custom status code handling for non-standard APIs

## Installation

```bash
go get github.com/brunobolting/go-snowy
```

## Basic Examples
```go
// Configure the client
config := snowy.Config{
    Timeout: 5 * time.Second,
}

// Define your response type
type UserResponse struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

// Make the request
response, err := snowy.Get[UserResponse](
    config,
    "https://api.example.com/users/1",
    nil,
    snowy.RequestData{}
)
if err != nil {
    // Handle error
    return err
}

// Access the data
user := response.Data
fmt.Println("User Name:", user.Name)
```
See [documentation](https://pkg.go.dev/github.com/brunobolting/go-snowy) for more examples and details.
