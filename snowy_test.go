package snowy_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/brunobolting/go-snowy"

	"github.com/stretchr/testify/assert"
)

type FakeUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type TestResponse struct {
	User    *FakeUser `json:"user"`
	Message string    `json:"message"`
}

func TestSnowyGet(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(TestResponse{
				User: &FakeUser{
					ID:       "123",
					Username: "test",
					Email:    "mail@test.com",
				},
				Message: "success",
			})
		}))
		defer ts.Close()

		res, err := snowy.Get[TestResponse](snowy.Config{}, ts.URL, nil, snowy.RequestData{})
		assert.Nil(t, err)
		assert.NotNil(t, res)
		assert.IsType(t, &TestResponse{}, res.Data)
		assert.Equal(t, "123", res.Data.User.ID)
		assert.Equal(t, "test", res.Data.User.Username)
		assert.Equal(t, "mail@test.com", res.Data.User.Email)
		assert.Equal(t, "success", res.Data.Message)
	})

	t.Run("success with headers", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer token", r.Header.Get("Authorization"))
			assert.Equal(t, "token", r.Header.Get("X-Token"))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		config := snowy.Config{
			Timeout: 10 * time.Second,
		}
		headers := snowy.Headers{
			"X-Token": "token",
		}
		headers.AddBearer("token")
		res, err := snowy.Get[TestResponse](config, ts.URL, headers, snowy.RequestData{})
		assert.Nil(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Nil(t, res.Data)
	})

	t.Run("success with basic auth", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, password, ok := r.BasicAuth()
			assert.True(t, ok)
			assert.Equal(t, "user", username)
			assert.Equal(t, "pass", password)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(TestResponse{
				User: &FakeUser{
					ID:       "123",
					Username: "test",
					Email:    "mail@test.com",
				},
				Message: "success",
			})
		}))

		config := snowy.Config{
			Timeout: 10 * time.Second,
		}

		headers := snowy.Headers{}
		headers.AddBasicAuth("user", "pass")
		res, err := snowy.Get[TestResponse](config, ts.URL, headers, snowy.RequestData{})
		assert.Nil(t, err)
		assert.NotNil(t, res)
		assert.IsType(t, &TestResponse{}, res.Data)
		assert.Equal(t, "123", res.Data.User.ID)
		assert.Equal(t, "test", res.Data.User.Username)
	})

	t.Run("success with query params", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "search=test", r.URL.RawQuery)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(TestResponse{
				User: &FakeUser{
					ID:       "123",
					Username: "test",
					Email:    "mail@test.com",
				},
				Message: "success",
			})
		}))
		defer ts.Close()

		config := snowy.Config{
			Timeout: 10 * time.Second,
		}

		res, err := snowy.Get[TestResponse](config, ts.URL, snowy.Headers{}, snowy.RequestData{
			QueryParams: map[string]string{
				"search": "test",
			},
		})
		assert.Nil(t, err)
		assert.NotNil(t, res)
		assert.IsType(t, &TestResponse{}, res.Data)
		assert.Equal(t, "123", res.Data.User.ID)
		assert.Equal(t, "test", res.Data.User.Username)
	})

	t.Run("success with acceptable status code", func(t *testing.T) {
		type customErrorRes struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(customErrorRes{
				Error:   "error",
				Message: "err",
			})
		}))

		config := snowy.Config{
			Timeout:               10 * time.Second,
			AcceptableStatusCodes: []int{http.StatusNotFound},
		}

		res, err := snowy.Get[customErrorRes](config, ts.URL, snowy.Headers{}, snowy.RequestData{})
		assert.Nil(t, err)
		assert.NotNil(t, res)
		assert.IsType(t, &customErrorRes{}, res.Data)
		assert.Equal(t, "error", res.Data.Error)
		assert.Equal(t, "err", res.Data.Message)
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
	})

	t.Run("error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()

		config := snowy.Config{
			Timeout: 10 * time.Second,
		}

		res, err := snowy.Get[TestResponse](config, ts.URL, snowy.Headers{}, snowy.RequestData{})
		assert.NotNil(t, err)
		assert.Nil(t, res)
		assert.IsType(t, &snowy.RequestError{}, err)
		assert.Equal(t, http.StatusInternalServerError, err.(*snowy.RequestError).StatusCode)
		assert.Equal(t, "message: unexpected status code: 500", err.(*snowy.RequestError).Error())
	})

	t.Run("timeout", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(50 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		config := snowy.Config{
			Timeout: 10 * time.Millisecond,
		}

		res, err := snowy.Get[TestResponse](config, ts.URL, snowy.Headers{}, snowy.RequestData{})
		assert.NotNil(t, err)
		assert.Nil(t, res)
	})
}

func TestSnowyPost(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "application/json", r.Header.Get("Accept"))
			assert.Equal(t, http.MethodPost, r.Method)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"user": map[string]any{
					"id":       "123",
					"username": "test",
					"email":    "mail",
				},
				"message": "success",
			})
		}))
		defer ts.Close()

		config := snowy.Config{
			Timeout: 10 * time.Second,
		}

		res, err := snowy.Post[TestResponse](config, ts.URL, snowy.Headers{}, snowy.RequestData{})
		assert.Nil(t, err)
		assert.NotNil(t, res)
		assert.IsType(t, &TestResponse{}, res.Data)
		assert.Equal(t, "123", res.Data.User.ID)
		assert.Equal(t, "test", res.Data.User.Username)
	})

	t.Run("success with json body", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var user map[string]any
			err := json.NewDecoder(r.Body).Decode(&user)
			assert.Nil(t, err)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.Equal(t, "123", user["id"])
			assert.Equal(t, "test", user["username"])
			assert.Equal(t, "mail@test.com", user["email"])
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(TestResponse{
				User: &FakeUser{
					ID:       "123",
					Username: "test",
					Email:    "mail@test.com",
				},
				Message: "success",
			})
		}))
		defer ts.Close()

		config := snowy.Config{
			Timeout: 10 * time.Second,
		}

		user := &FakeUser{
			ID:       "123",
			Username: "test",
			Email:    "mail@test.com",
		}

		res, err := snowy.Post[TestResponse](config, ts.URL, snowy.Headers{},
			snowy.RequestData{
				JsonData: user,
			},
		)

		assert.Nil(t, err)
		assert.NotNil(t, res)
		assert.IsType(t, &TestResponse{}, res.Data)
		assert.Equal(t, "123", res.Data.User.ID)
		assert.Equal(t, "test", res.Data.User.Username)
	})

	t.Run("success with form data", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := r.ParseForm()
			assert.Nil(t, err)
			assert.Equal(t, "test", r.Form.Get("username"))
			assert.Equal(t, "id", r.Form.Get("id"))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(TestResponse{
				User: &FakeUser{
					ID:       "123",
					Username: "test",
					Email:    "mail@test.com",
				},
				Message: "success",
			})
		}))

		config := snowy.Config{
			Timeout: 10 * time.Second,
		}

		res, err := snowy.Post[TestResponse](config, ts.URL, snowy.Headers{},
			snowy.RequestData{
				FormData: map[string]string{
					"username": "test",
					"id":       "id",
				},
			},
		)

		assert.Nil(t, err)
		assert.NotNil(t, res)
		assert.IsType(t, &TestResponse{}, res.Data)
		assert.Equal(t, "123", res.Data.User.ID)
		assert.Equal(t, "test", res.Data.User.Username)
	})

	t.Run("success with headers", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer token", r.Header.Get("Authorization"))
			assert.Equal(t, "token", r.Header.Get("X-Token"))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(TestResponse{
				User: &FakeUser{
					ID:       "123",
					Username: "test",
					Email:    "mail@test.com",
				},
				Message: "success",
			})
		}))
		defer ts.Close()

		config := snowy.Config{
			Timeout: 10 * time.Second,
		}

		headers := snowy.Headers{
			"X-Token": "token",
		}

		headers.AddBearer("token")
		res, err := snowy.Post[TestResponse](config, ts.URL, headers, snowy.RequestData{})
		assert.Nil(t, err)
		assert.NotNil(t, res)
		assert.IsType(t, &TestResponse{}, res.Data)
		assert.Equal(t, "123", res.Data.User.ID)
		assert.Equal(t, "test", res.Data.User.Username)
		assert.Equal(t, "Bearer token", headers["Authorization"])
	})
}

func TestSnowyPut(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPut, r.Method)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(TestResponse{
				User: &FakeUser{
					ID:       "123",
					Username: "test",
					Email:    "mail@test.com",
				},
				Message: "success",
			})
		}))

		res, err := snowy.Put[TestResponse](snowy.Config{}, ts.URL, snowy.Headers{}, snowy.RequestData{})
		assert.Nil(t, err)
		assert.NotNil(t, res)
		assert.IsType(t, &TestResponse{}, res.Data)
		assert.Equal(t, "123", res.Data.User.ID)
		assert.Equal(t, "test", res.Data.User.Username)
	})
}

func TestSnowyPatch(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPatch, r.Method)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(TestResponse{
				User: &FakeUser{
					ID:       "123",
					Username: "test",
					Email:    "mail@test.com",
				},
				Message: "success",
			})
		}))

		res, err := snowy.Patch[TestResponse](snowy.Config{}, ts.URL, snowy.Headers{}, snowy.RequestData{})
		assert.Nil(t, err)
		assert.NotNil(t, res)
		assert.IsType(t, &TestResponse{}, res.Data)
		assert.Equal(t, "123", res.Data.User.ID)
		assert.Equal(t, "test", res.Data.User.Username)
	})
}

func TestSnowyDelete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodDelete, r.Method)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(TestResponse{
				User: &FakeUser{
					ID:       "123",
					Username: "test",
					Email:    "mail@test.com",
				},
				Message: "success",
			})
		}))

		res, err := snowy.Delete[TestResponse](snowy.Config{}, ts.URL, snowy.Headers{}, snowy.RequestData{})
		assert.Nil(t, err)
		assert.NotNil(t, res)
		assert.IsType(t, &TestResponse{}, res.Data)
		assert.Equal(t, "123", res.Data.User.ID)
		assert.Equal(t, "test", res.Data.User.Username)
	})
}

func TestSnowyRequestError(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		err := snowy.RequestError{
			StatusCode: 500,
			Message:    "unexpected status code: 500",
		}
		assert.Equal(t, "message: unexpected status code: 500", err.Error())
		assert.IsType(t, snowy.RequestError{}, err)
	})
}

func TestSnowyHeaders(t *testing.T) {
	t.Run("header and add bearer", func(t *testing.T) {
		headers := snowy.Headers{
			"X-Token": "token",
		}
		headers.AddBearer("token")
		headers.Add("X-Test", "test")
		assert.Equal(t, "Bearer token", headers["Authorization"])
		assert.Equal(t, "test", headers["X-Test"])
	})

	t.Run("header and add basic auth", func(t *testing.T) {
		headers := snowy.Headers{}
		headers.AddBasicAuth("user", "pass")
		assert.Contains(t, headers, "Authorization")
	})

	t.Run("test contains", func(t *testing.T) {
		headers := snowy.Headers{
			"X-Token": "token",
		}
		assert.True(t, headers.Contains("X-Token"))
		assert.False(t, headers.Contains("Authorization"))
		assert.Contains(t, headers, "X-Token")
		assert.NotContains(t, headers, "Authorization")
	})

	t.Run("test remove", func(t *testing.T) {
		headers := snowy.Headers{
			"X-Token": "token",
		}
		headers.Remove("X-Token")
		assert.NotContains(t, headers, "X-Token")
	})

	t.Run("test get", func(t *testing.T) {
		headers := snowy.Headers{
			"X-Token": "token",
		}
		assert.Equal(t, "token", headers.Get("X-Token"))
		assert.Equal(t, "", headers.Get("Authorization"))
	})
}
