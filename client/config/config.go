package config

import (
	"os"
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

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
