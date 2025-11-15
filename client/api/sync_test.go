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

func TestClient_Sync(t *testing.T) {
	t.Run("should sync data successfully", func(t *testing.T) {
		lastSyncTime := time.Now().UTC().Add(-time.Hour)
		localEntries := []*models.DataEntry{
			{
				ID:   "local-1",
				Name: "Local Entry",
				Type: models.TypeLoginPassword,
			},
		}

		serverTime := time.Now().UTC()
		newEntry := &models.DataEntry{
			ID:   "server-1",
			Name: "Server Entry",
			Type: models.TypeCard,
		}

		mockRT := &mockRoundTripper{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, "POST", req.Method)
				assert.Equal(t, "/api/sync", req.URL.Path)

				var syncReq SyncRequest
				json.NewDecoder(req.Body).Decode(&syncReq)
				assert.WithinDuration(t, lastSyncTime, syncReq.LastSyncTime, time.Second)
				assert.Len(t, syncReq.Entries, 1)

				response := SyncResponse{
					ServerTime: serverTime,
					NewEntries: []*models.DataEntry{newEntry},
					UpdatedIDs: []string{"updated-1"},
				}
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

		result, err := client.Sync(lastSyncTime, localEntries)

		require.NoError(t, err)
		assert.WithinDuration(t, serverTime, result.ServerTime, time.Second)
		assert.Len(t, result.NewEntries, 1)
		assert.Equal(t, "server-1", result.NewEntries[0].ID)
		assert.Len(t, result.UpdatedIDs, 1)
	})

	t.Run("should handle sync conflicts", func(t *testing.T) {
		clientEntry := &models.DataEntry{ID: "1", Name: "Client"}
		serverEntry := &models.DataEntry{ID: "1", Name: "Server"}

		mockRT := &mockRoundTripper{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				response := SyncResponse{
					ServerTime: time.Now().UTC(),
					Conflicts: []*Conflict{
						{
							ClientEntry: clientEntry,
							ServerEntry: serverEntry,
						},
					},
				}
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

		result, err := client.Sync(time.Now().UTC(), []*models.DataEntry{})

		require.NoError(t, err)
		assert.Len(t, result.Conflicts, 1)
		assert.Equal(t, "Client", result.Conflicts[0].ClientEntry.Name)
		assert.Equal(t, "Server", result.Conflicts[0].ServerEntry.Name)
	})

	t.Run("should return error on sync failure", func(t *testing.T) {
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

		result, err := client.Sync(time.Now().UTC(), []*models.DataEntry{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "sync failed with status: 500")
		assert.Nil(t, result)
	})
}

func TestClient_GetSyncStatus(t *testing.T) {
	t.Run("should get sync status successfully", func(t *testing.T) {
		lastSync := time.Now().UTC().Add(-10 * time.Minute)
		expectedStatus := &SyncStatus{
			LastSyncTime:   lastSync,
			PendingChanges: 2,
			HasConflicts:   false,
		}

		mockRT := &mockRoundTripper{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, "GET", req.Method)
				assert.Equal(t, "/api/sync/status", req.URL.Path)

				bodyBytes, _ := json.Marshal(expectedStatus)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(bodyBytes)),
					Header:     make(http.Header),
				}, nil
			},
		}

		client := NewClient("http://test.com")
		client.httpClient.Transport = mockRT

		result, err := client.GetSyncStatus()

		require.NoError(t, err)
		assert.WithinDuration(t, lastSync, result.LastSyncTime, time.Second)
		assert.Equal(t, 2, result.PendingChanges)
		assert.False(t, result.HasConflicts)
	})

	t.Run("should handle status with conflicts", func(t *testing.T) {
		mockRT := &mockRoundTripper{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				status := SyncStatus{
					LastSyncTime:   time.Now().UTC(),
					PendingChanges: 0,
					HasConflicts:   true,
				}
				bodyBytes, _ := json.Marshal(status)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(bodyBytes)),
					Header:     make(http.Header),
				}, nil
			},
		}

		client := NewClient("http://test.com")
		client.httpClient.Transport = mockRT

		result, err := client.GetSyncStatus()

		require.NoError(t, err)
		assert.True(t, result.HasConflicts)
	})

	t.Run("should return error on status failure", func(t *testing.T) {
		mockRT := &mockRoundTripper{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("network error")
			},
		}

		client := NewClient("http://test.com")
		client.httpClient.Transport = mockRT

		result, err := client.GetSyncStatus()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "sync status request failed")
		assert.Nil(t, result)
	})
}

func TestClient_ResolveConflict(t *testing.T) {
	t.Run("should resolve conflict successfully", func(t *testing.T) {
		entry := &models.DataEntry{
			ID:   "conflict-1",
			Name: "Resolved Entry",
			Type: models.TypeLoginPassword,
		}

		mockRT := &mockRoundTripper{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, "POST", req.Method)
				assert.Equal(t, "/api/sync/resolve", req.URL.Path)

				var resolutionReq map[string]interface{}
				json.NewDecoder(req.Body).Decode(&resolutionReq)
				assert.Equal(t, "conflict-123", resolutionReq["conflict_id"])
				assert.Equal(t, "client", resolutionReq["resolution"])

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("")),
					Header:     make(http.Header),
				}, nil
			},
		}

		client := NewClient("http://test.com")
		client.httpClient.Transport = mockRT

		err := client.ResolveConflict("conflict-123", "client", entry)

		assert.NoError(t, err)
	})

	t.Run("should resolve with server version", func(t *testing.T) {
		mockRT := &mockRoundTripper{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				var resolutionReq map[string]interface{}
				json.NewDecoder(req.Body).Decode(&resolutionReq)
				assert.Equal(t, "server", resolutionReq["resolution"])
				assert.Nil(t, resolutionReq["entry"])

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("")),
					Header:     make(http.Header),
				}, nil
			},
		}

		client := NewClient("http://test.com")
		client.httpClient.Transport = mockRT

		err := client.ResolveConflict("conflict-456", "server", nil)

		assert.NoError(t, err)
	})

	t.Run("should return error on resolution failure", func(t *testing.T) {
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

		err := client.ResolveConflict("conflict-789", "client", &models.DataEntry{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "conflict resolution failed with status: 409")
	})
}
