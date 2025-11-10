package config

import "os"

type Config struct {
	ServerURL string
	Token     string
}

func Load() *Config {
	return &Config{
		ServerURL: getEnv("GOPHKEEPER_SERVER", "http://localhost:8080"),
		Token:     getEnv("GOPHKEEPER_TOKEN", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
