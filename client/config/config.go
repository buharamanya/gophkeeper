package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	ServerURL  string `json:"server_url"`
	Token      string `json:"token,omitempty"`
	SkipVerify bool   `json:"skip_verify"`
	Timeout    int    `json:"timeout"`
}

// Load загружает конфигурацию из файла и переменных окружения
func Load() *Config {
	skipVerify, _ := strconv.ParseBool(getEnv("GOPHKEEPER_SKIP_VERIFY", "false"))
	timeout, _ := strconv.Atoi(getEnv("GOPHKEEPER_TIMEOUT", "30"))

	// Пробуем загрузить из файла
	fileCfg, err := LoadFromFile()
	if err != nil {
		// Если файла нет, создаем конфиг по умолчанию
		return &Config{
			ServerURL:  getEnv("GOPHKEEPER_SERVER", "http://localhost:8080"),
			Token:      getEnv("GOPHKEEPER_TOKEN", ""),
			SkipVerify: skipVerify,
			Timeout:    timeout,
		}
	}

	// Переменные окружения имеют приоритет над файлом
	if serverURL := getEnv("GOPHKEEPER_SERVER", ""); serverURL != "" {
		fileCfg.ServerURL = serverURL
	}
	if token := getEnv("GOPHKEEPER_TOKEN", ""); token != "" {
		fileCfg.Token = token
	}
	fileCfg.SkipVerify = skipVerify
	fileCfg.Timeout = timeout

	return fileCfg
}

// Save сохраняет конфигурацию в файл (безопасно, без вывода в консоль)
func Save(cfg *Config) error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	appDir := filepath.Join(configDir, "gophkeeper")
	if err := os.MkdirAll(appDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configFile := filepath.Join(appDir, "config.json")

	// Создаем временный файл для атомарной записи
	tempFile := configFile + ".tmp"

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Записываем во временный файл
	if err := os.WriteFile(tempFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write temp config file: %w", err)
	}

	// Атомарно заменяем старый файл новым
	if err := os.Rename(tempFile, configFile); err != nil {
		return fmt.Errorf("failed to replace config file: %w", err)
	}

	return nil
}

// SaveToken безопасно сохраняет токен в конфигурационный файл
func SaveToken(token string) error {
	cfg := Load()
	cfg.Token = token
	return Save(cfg)
}

// ClearToken удаляет токен из конфигурации
func ClearToken() error {
	cfg := Load()
	cfg.Token = ""
	return Save(cfg)
}

// LoadFromFile загружает конфигурацию из файла
func LoadFromFile() (*Config, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}

	configFile := filepath.Join(configDir, "gophkeeper", "config.json")
	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil // Возвращаем пустой конфиг, если файла нет
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// GetConfigPath возвращает путь к файлу конфигурации
func GetConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "gophkeeper", "config.json"), nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
