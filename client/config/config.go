package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	ServerURL  string
	Token      string
	SkipVerify bool
	Timeout    int
}

func Load() *Config {
	skipVerify, _ := strconv.ParseBool(getEnv("GOPHKEEPER_SKIP_VERIFY", "false"))
	timeout, _ := strconv.Atoi(getEnv("GOPHKEEPER_TIMEOUT", "30"))

	return &Config{
		ServerURL:  getEnv("GOPHKEEPER_SERVER", "http://localhost:8080"),
		Token:      getEnv("GOPHKEEPER_TOKEN", ""),
		SkipVerify: skipVerify,
		Timeout:    timeout,
	}
}

// Save сохраняет конфигурацию в файл
func Save(cfg *Config) error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	appDir := filepath.Join(configDir, "gophkeeper")
	if err := os.MkdirAll(appDir, 0700); err != nil {
		return err
	}

	configFile := filepath.Join(appDir, "config.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configFile, data, 0600)
}

// LoadFromFile загружает конфигурацию из файла
func LoadFromFile() (*Config, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	configFile := filepath.Join(configDir, "gophkeeper", "config.json")
	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return Load(), nil // Возвращаем конфиг по умолчанию, если файла нет
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
