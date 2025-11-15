package config

import (
	"crypto/tls"
	"os"
	"strconv"
)

type Config struct {
	ServerAddress string
	JWTSecret     string
	DatabaseURL   string
	TLSCertFile   string
	TLSKeyFile    string
	EnableTLS     bool
}

func Load() *Config {
	enableTLS, _ := strconv.ParseBool(getEnv("ENABLE_TLS", "false"))

	return &Config{
		ServerAddress: getEnv("SERVER_ADDRESS", ":8080"),
		JWTSecret:     getEnv("JWT_SECRET", "default-secret-change-in-production"),
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://gopher:gopherpass@localhost:5432/gophkeeper?sslmode=disable"),
		TLSCertFile:   getEnv("TLS_CERT_FILE", ""),
		TLSKeyFile:    getEnv("TLS_KEY_FILE", ""),
		EnableTLS:     enableTLS,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// LoadTLSConfig загружает TLS конфигурацию
func (c *Config) LoadTLSConfig() (*tls.Config, error) {
	if !c.EnableTLS || c.TLSCertFile == "" || c.TLSKeyFile == "" {
		return nil, nil
	}

	cert, err := tls.LoadX509KeyPair(c.TLSCertFile, c.TLSKeyFile)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
		},
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}, nil
}
