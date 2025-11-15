package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/buharamanya/gophkeeper/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRoundTripper реализует http.RoundTripper для мокирования HTTP запросов
type mockRoundTripper struct {
	roundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTripFunc(req)
}

func TestNewClient(t *testing.T) {
	t.Run("should create client with correct baseURL", func(t *testing.T) {
		baseURL := "http://localhost:8080"
		client := NewClient(baseURL)

		assert.Equal(t, baseURL, client.baseURL)
		assert.NotNil(t, client.httpClient)
		assert.Equal(t, 30*time.Second, client.httpClient.Timeout)
	})

	t.Run("should create client with empty token", func(t *testing.T) {
		client := NewClient("http://test.com")
		assert.Empty(t, client.token)
	})
}

func TestSetToken(t *testing.T) {
	t.Run("should set token correctly", func(t *testing.T) {
		client := NewClient("http://test.com")
		token := "test-token-123"

		client.SetToken(token)

		assert.Equal(t, token, client.token)
	})
}

func TestClient_Register(t *testing.T) {
	t.Run("should register user successfully", func(t *testing.T) {
		mockRT := &mockRoundTripper{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, "POST", req.Method)
				assert.Equal(t, "/api/register", req.URL.Path)
				assert.Equal(t, "application/json", req.Header.Get("Content-Type"))

				// Проверяем тело запроса
				var body models.RegisterRequest
				json.NewDecoder(req.Body).Decode(&body)
				assert.Equal(t, "testuser", body.Login)
				assert.Equal(t, "testpass", body.Password)

				// Возвращаем успешный ответ
				authResp := models.AuthResponse{
					Token:  "jwt-token-123",
					UserID: "user-id-456",
				}
				bodyBytes, _ := json.Marshal(authResp)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(bodyBytes)),
					Header:     make(http.Header),
				}, nil
			},
		}

		client := NewClient("http://test.com")
		client.httpClient.Transport = mockRT

		result, err := client.Register("testuser", "testpass")

		require.NoError(t, err)
		assert.Equal(t, "jwt-token-123", result.Token)
		assert.Equal(t, "user-id-456", result.UserID)
	})

	t.Run("should return error on non-200 status", func(t *testing.T) {
		mockRT := &mockRoundTripper{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader("")),
					Header:     make(http.Header),
				}, nil
			},
		}

		client := NewClient("http://test.com")
		client.httpClient.Transport = mockRT

		result, err := client.Register("user", "pass")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "register failed with status: 400")
		assert.Nil(t, result)
	})

	t.Run("should return error on request failure", func(t *testing.T) {
		mockRT := &mockRoundTripper{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("network error")
			},
		}

		client := NewClient("http://test.com")
		client.httpClient.Transport = mockRT

		result, err := client.Register("user", "pass")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "register request failed")
		assert.Nil(t, result)
	})
}

func TestClient_Login(t *testing.T) {
	t.Run("should login user successfully", func(t *testing.T) {
		mockRT := &mockRoundTripper{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, "POST", req.Method)
				assert.Equal(t, "/api/login", req.URL.Path)

				var body models.LoginRequest
				json.NewDecoder(req.Body).Decode(&body)
				assert.Equal(t, "user123", body.Login)
				assert.Equal(t, "pass123", body.Password)

				authResp := models.AuthResponse{
					Token:  "login-token-456",
					UserID: "user-id-789",
				}
				bodyBytes, _ := json.Marshal(authResp)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(bodyBytes)),
					Header:     make(http.Header),
				}, nil
			},
		}

		client := NewClient("http://test.com")
		client.httpClient.Transport = mockRT

		result, err := client.Login("user123", "pass123")

		require.NoError(t, err)
		assert.Equal(t, "login-token-456", result.Token)
		assert.Equal(t, "user-id-789", result.UserID)
	})

	t.Run("should return error on invalid credentials", func(t *testing.T) {
		mockRT := &mockRoundTripper{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader("")),
					Header:     make(http.Header),
				}, nil
			},
		}

		client := NewClient("http://test.com")
		client.httpClient.Transport = mockRT

		result, err := client.Login("wrong", "credentials")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "login failed with status: 401")
		assert.Nil(t, result)
	})
}

func TestClient_CreateData(t *testing.T) {
	t.Run("should create data entry successfully", func(t *testing.T) {
		mockRT := &mockRoundTripper{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, "POST", req.Method)
				assert.Equal(t, "/api/data", req.URL.Path)
				assert.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))

				// Проверяем тело запроса
				var entry models.DataEntry
				json.NewDecoder(req.Body).Decode(&entry)
				assert.Equal(t, models.TypeLoginPassword, entry.Type)
				assert.Equal(t, "Test Password", entry.Name)

				response := map[string]string{"id": "new-data-id-123"}
				bodyBytes, _ := json.Marshal(response)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(bodyBytes)),
					Header:     make(http.Header),
				}, nil
			},
		}

		client := NewClient("http://test.com")
		client.SetToken("test-token")
		client.httpClient.Transport = mockRT

		entry := &models.DataEntry{
			Name:     "Test Password",
			Type:     models.TypeLoginPassword,
			Metadata: "test metadata",
			Data:     []byte("test data"),
			Nonce:    []byte("nonce"),
		}

		id, err := client.CreateData(entry)

		require.NoError(t, err)
		assert.Equal(t, "new-data-id-123", id)
	})

	t.Run("should return error without auth token", func(t *testing.T) {
		mockRT := &mockRoundTripper{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				// Должен отсутствовать Authorization header
				assert.Empty(t, req.Header.Get("Authorization"))
				return &http.Response{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader("")),
					Header:     make(http.Header),
				}, nil
			},
		}

		client := NewClient("http://test.com")
		// Не устанавливаем токен
		client.httpClient.Transport = mockRT

		id, err := client.CreateData(&models.DataEntry{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "create data failed with status: 401")
		assert.Empty(t, id)
	})
}

func TestClient_ListData(t *testing.T) {
	t.Run("should list data entries successfully", func(t *testing.T) {
		expectedData := []*models.DataEntry{
			{
				ID:       "1",
				Name:     "Gmail",
				Type:     models.TypeLoginPassword,
				Metadata: "email password",
				Version:  1,
			},
			{
				ID:       "2",
				Name:     "Credit Card",
				Type:     models.TypeCard,
				Metadata: "main card",
				Version:  1,
			},
		}

		mockRT := &mockRoundTripper{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, "GET", req.Method)
				assert.Equal(t, "/api/data", req.URL.Path)

				response := map[string][]*models.DataEntry{"data": expectedData}
				bodyBytes, _ := json.Marshal(response)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(bodyBytes)),
					Header:     make(http.Header),
				}, nil
			},
		}

		client := NewClient("http://test.com")
		client.httpClient.Transport = mockRT

		result, err := client.ListData()

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, expectedData[0].ID, result[0].ID)
		assert.Equal(t, expectedData[0].Name, result[0].Name)
		assert.Equal(t, expectedData[1].ID, result[1].ID)
		assert.Equal(t, expectedData[1].Name, result[1].Name)
	})

	t.Run("should return empty list when no data", func(t *testing.T) {
		mockRT := &mockRoundTripper{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				response := map[string][]*models.DataEntry{"data": {}}
				bodyBytes, _ := json.Marshal(response)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(bodyBytes)),
					Header:     make(http.Header),
				}, nil
			},
		}

		client := NewClient("http://test.com")
		client.httpClient.Transport = mockRT

		result, err := client.ListData()

		require.NoError(t, err)
		assert.Empty(t, result)
	})
}

func TestClient_GetData(t *testing.T) {
	t.Run("should get data entry by id", func(t *testing.T) {
		fixedTime := time.Now().UTC()
		expectedEntry := &models.DataEntry{
			ID:        "test-id",
			Name:      "Test Entry",
			Type:      models.TypeLoginPassword,
			Metadata:  "test metadata",
			Data:      []byte("encrypted data"),
			Nonce:     []byte("nonce"),
			Version:   2,
			CreatedAt: fixedTime,
			UpdatedAt: fixedTime,
		}

		mockRT := &mockRoundTripper{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, "GET", req.Method)
				assert.Equal(t, "/api/data/test-id", req.URL.Path)

				bodyBytes, _ := json.Marshal(expectedEntry)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(bodyBytes)),
					Header:     make(http.Header),
				}, nil
			},
		}

		client := NewClient("http://test.com")
		client.httpClient.Transport = mockRT

		result, err := client.GetData("test-id")

		require.NoError(t, err)
		assert.Equal(t, expectedEntry.ID, result.ID)
		assert.Equal(t, expectedEntry.Name, result.Name)
		assert.Equal(t, expectedEntry.Type, result.Type)
		assert.Equal(t, expectedEntry.Version, result.Version)
		assert.Equal(t, expectedEntry.Data, result.Data)
	})

	t.Run("should return error for non-existent id", func(t *testing.T) {
		mockRT := &mockRoundTripper{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader("")),
					Header:     make(http.Header),
				}, nil
			},
		}

		client := NewClient("http://test.com")
		client.httpClient.Transport = mockRT

		result, err := client.GetData("non-existent")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "get data failed with status: 404")
		assert.Nil(t, result)
	})
}

func TestClient_UpdateDataWithVersion(t *testing.T) {
	t.Run("should update data with version successfully", func(t *testing.T) {
		mockRT := &mockRoundTripper{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, "PUT", req.Method)
				assert.Equal(t, "/api/data/test-id", req.URL.Path)

				var updateReq models.UpdateRequest
				json.NewDecoder(req.Body).Decode(&updateReq)
				assert.Equal(t, int64(2), updateReq.ExpectedVersion)
				assert.Equal(t, "Updated Entry", updateReq.Name)

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("")),
					Header:     make(http.Header),
				}, nil
			},
		}

		client := NewClient("http://test.com")
		client.httpClient.Transport = mockRT

		updateReq := &models.UpdateRequest{
			DataEntry: &models.DataEntry{
				Name:     "Updated Entry",
				Type:     models.TypeLoginPassword,
				Metadata: "updated metadata",
			},
			ExpectedVersion: 2,
		}

		err := client.UpdateDataWithVersion("test-id", updateReq)

		assert.NoError(t, err)
	})

	t.Run("should return version conflict error", func(t *testing.T) {
		mockRT := &mockRoundTripper{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusConflict,
					Body:       io.NopCloser(strings.NewReader("")),
					Header:     make(http.Header),
				}, nil
			},
		}

		client := NewClient("http://test.com")
		client.httpClient.Transport = mockRT

		err := client.UpdateDataWithVersion("test-id", &models.UpdateRequest{
			DataEntry:       &models.DataEntry{},
			ExpectedVersion: 1,
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "version conflict")
	})
}

func TestClient_DeleteData(t *testing.T) {
	t.Run("should delete data successfully", func(t *testing.T) {
		mockRT := &mockRoundTripper{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, "DELETE", req.Method)
				assert.Equal(t, "/api/data/test-id", req.URL.Path)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("")),
					Header:     make(http.Header),
				}, nil
			},
		}

		client := NewClient("http://test.com")
		client.httpClient.Transport = mockRT

		err := client.DeleteData("test-id")

		assert.NoError(t, err)
	})

	t.Run("should return error on delete failure", func(t *testing.T) {
		mockRT := &mockRoundTripper{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader("")),
					Header:     make(http.Header),
				}, nil
			},
		}

		client := NewClient("http://test.com")
		client.httpClient.Transport = mockRT

		err := client.DeleteData("test-id")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "delete data failed with status: 500")
	})
}

func TestClient_UpdateData(t *testing.T) {
	t.Run("should update data successfully", func(t *testing.T) {
		mockRT := &mockRoundTripper{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, "PUT", req.Method)
				assert.Equal(t, "/api/data/test-id", req.URL.Path)

				var entry models.DataEntry
				json.NewDecoder(req.Body).Decode(&entry)
				assert.Equal(t, models.TypeLoginPassword, entry.Type)
				assert.Equal(t, "updated entry", entry.Name)

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("")),
					Header:     make(http.Header),
				}, nil
			},
		}

		client := NewClient("http://test.com")
		client.httpClient.Transport = mockRT

		entry := &models.DataEntry{
			Name:     "updated entry",
			Type:     models.TypeLoginPassword,
			Metadata: "updated metadata",
			Data:     []byte("updated data"),
		}

		err := client.UpdateData("test-id", entry)

		assert.NoError(t, err)
	})
}
