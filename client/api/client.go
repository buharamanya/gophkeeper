package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/buharamanya/gophkeeper/internal/models"
)

type GophKeeperClient struct {
	baseURL string
	token   string
	client  *http.Client
}

func NewClient(baseURL string) *GophKeeperClient {
	return &GophKeeperClient{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

func (c *GophKeeperClient) SetToken(token string) {
	c.token = token
}

func (c *GophKeeperClient) Register(login, password string) (*models.AuthResponse, error) {
	req := models.RegisterRequest{
		Login:    login,
		Password: password,
	}

	var resp models.AuthResponse
	err := c.doRequest("POST", "/api/register", req, &resp)
	if err != nil {
		return nil, err
	}

	c.token = resp.Token
	return &resp, nil
}

func (c *GophKeeperClient) Login(login, password string) (*models.AuthResponse, error) {
	req := models.LoginRequest{
		Login:    login,
		Password: password,
	}

	var resp models.AuthResponse
	err := c.doRequest("POST", "/api/login", req, &resp)
	if err != nil {
		return nil, err
	}

	c.token = resp.Token
	return &resp, nil
}

func (c *GophKeeperClient) CreateData(entry *models.DataEntry) (string, error) {
	var resp struct {
		ID string `json:"id"`
	}

	err := c.doRequest("POST", "/api/data", entry, &resp)
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

func (c *GophKeeperClient) ListData() ([]*models.DataEntry, error) {
	var resp struct {
		Data []*models.DataEntry `json:"data"`
	}

	err := c.doRequest("GET", "/api/data", nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

func (c *GophKeeperClient) GetData(id string) (*models.DataEntry, error) {
	var entry models.DataEntry
	err := c.doRequest("GET", "/api/data/"+id, nil, &entry)
	if err != nil {
		return nil, err
	}

	return &entry, nil
}

func (c *GophKeeperClient) UpdateData(id string, entry *models.DataEntry) error {
	return c.doRequest("PUT", "/api/data/"+id, entry, nil)
}

func (c *GophKeeperClient) DeleteData(id string) error {
	return c.doRequest("DELETE", "/api/data/"+id, nil, nil)
}

func (c *GophKeeperClient) doRequest(method, path string, body interface{}, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errorResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
			return fmt.Errorf("request failed with status %d", resp.StatusCode)
		}
		return fmt.Errorf("request failed: %s", errorResp.Error)
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}
