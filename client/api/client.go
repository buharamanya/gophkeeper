package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/buharamanya/gophkeeper/internal/models"
)

// Client представляет клиент для работы с API GophKeeper
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewClient создает новый экземпляр клиента
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetToken устанавливает токен авторизации
func (c *Client) SetToken(token string) {
	c.token = token
}

// doRequest выполняет HTTP запрос
func (c *Client) doRequest(method, path string, body []byte) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}

// Register регистрирует нового пользователя
func (c *Client) Register(login, password string) (*models.AuthResponse, error) {
	req := models.RegisterRequest{
		Login:    login,
		Password: password,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal register request: %w", err)
	}

	resp, err := c.doRequest(http.MethodPost, "/api/register", body)
	if err != nil {
		return nil, fmt.Errorf("register request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("register failed with status: %d", resp.StatusCode)
	}

	var authResp models.AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, fmt.Errorf("failed to decode register response: %w", err)
	}

	return &authResp, nil
}

// Login выполняет вход пользователя
func (c *Client) Login(login, password string) (*models.AuthResponse, error) {
	req := models.LoginRequest{
		Login:    login,
		Password: password,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal login request: %w", err)
	}

	resp, err := c.doRequest(http.MethodPost, "/api/login", body)
	if err != nil {
		return nil, fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("login failed with status: %d", resp.StatusCode)
	}

	var authResp models.AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, fmt.Errorf("failed to decode login response: %w", err)
	}

	return &authResp, nil
}

// CreateData создает новую запись данных
func (c *Client) CreateData(entry *models.DataEntry) (string, error) {
	body, err := json.Marshal(entry)
	if err != nil {
		return "", fmt.Errorf("failed to marshal data entry: %w", err)
	}

	resp, err := c.doRequest(http.MethodPost, "/api/data", body)
	if err != nil {
		return "", fmt.Errorf("create data request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("create data failed with status: %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result["id"], nil
}

// ListData получает список всех записей данных
func (c *Client) ListData() ([]*models.DataEntry, error) {
	resp, err := c.doRequest(http.MethodGet, "/api/data", nil)
	if err != nil {
		return nil, fmt.Errorf("list data request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list data failed with status: %d", resp.StatusCode)
	}

	var result map[string][]*models.DataEntry
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result["data"], nil
}

// GetData получает запись данных по ID
func (c *Client) GetData(id string) (*models.DataEntry, error) {
	resp, err := c.doRequest(http.MethodGet, "/api/data/"+id, nil)
	if err != nil {
		return nil, fmt.Errorf("get data request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get data failed with status: %d", resp.StatusCode)
	}

	var entry models.DataEntry
	if err := json.NewDecoder(resp.Body).Decode(&entry); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &entry, nil
}

// UpdateData обновляет запись данных (обратная совместимость)
func (c *Client) UpdateData(id string, entry *models.DataEntry) error {
	body, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal data entry: %w", err)
	}

	resp, err := c.doRequest(http.MethodPut, "/api/data/"+id, body)
	if err != nil {
		return fmt.Errorf("update data request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("update data failed with status: %d", resp.StatusCode)
	}

	return nil
}

// UpdateDataWithVersion обновляет запись данных с оптимистической блокировкой
func (c *Client) UpdateDataWithVersion(id string, updateReq *models.UpdateRequest) error {
	body, err := json.Marshal(updateReq)
	if err != nil {
		return fmt.Errorf("failed to marshal update request: %w", err)
	}

	resp, err := c.doRequest(http.MethodPut, "/api/data/"+id, body)
	if err != nil {
		return fmt.Errorf("update data request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusConflict {
			return fmt.Errorf("version conflict: data was modified by another operation")
		}
		return fmt.Errorf("update data failed with status: %d", resp.StatusCode)
	}

	return nil
}

// DeleteData удаляет запись данных
func (c *Client) DeleteData(id string) error {
	resp, err := c.doRequest(http.MethodDelete, "/api/data/"+id, nil)
	if err != nil {
		return fmt.Errorf("delete data request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete data failed with status: %d", resp.StatusCode)
	}

	return nil
}
