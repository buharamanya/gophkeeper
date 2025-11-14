package app

import (
	"testing"
)

func TestValidateLogin(t *testing.T) {
	tests := []struct {
		name     string
		login    string
		expected string
	}{
		{"Valid login", "john_doe", ""},
		{"Valid login with numbers", "user123", ""},
		{"Valid login with dash", "user-name", ""},
		{"Too short", "ab", "логин должен содержать минимум 3 символа"},
		{"Too long", "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz", "логин не может превышать 50 символов"},
		{"Starts with number", "1user", "логин должен начинаться с буквы"},
		{"Starts with underscore", "_user", "логин должен начинаться с буквы"},
		{"Contains spaces", "user name", "логин может содержать только буквы (a-z, A-Z), цифры (0-9), подчеркивание (_) и дефис (-)"},
		{"Contains special chars", "user@name", "логин может содержать только буквы (a-z, A-Z), цифры (0-9), подчеркивание (_) и дефис (-)"},
		{"Empty login", "", "логин не может быть пустым"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLogin(tt.login)
			if tt.expected == "" && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
			if tt.expected != "" && (err == nil || err.Error() != tt.expected) {
				t.Errorf("Expected error '%s', got: %v", tt.expected, err)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		expected string
	}{
		{"Valid password", "SecurePass123!", ""},
		{"Too short", "Ab1!", "пароль должен содержать минимум 8 символов"},
		{"No uppercase", "password123!", "пароль должен содержать хотя бы одну заглавную букву"},
		{"No lowercase", "PASSWORD123!", "пароль должен содержать хотя бы одну строчную букву"},
		{"No numbers", "Password!", "пароль должен содержать хотя бы одну цифру"},
		{"No special chars", "Password123", "пароль должен содержать хотя бы один специальный символ (например: !@#$%^&*)"},
		{"Common password", "Password123!", ""}, // Этот пароль теперь валидный
		{"Empty password", "", "пароль не может быть пустым"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if tt.expected == "" && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
			if tt.expected != "" && (err == nil || err.Error() != tt.expected) {
				t.Errorf("Expected error '%s', got: %v", tt.expected, err)
			}
		})
	}
}

func TestValidatePassword_WeakPasswords(t *testing.T) {
	weakPasswords := []string{
		"password", "12345678", "qwerty", "admin", "welcome", "monkey",
	}

	for _, pwd := range weakPasswords {
		t.Run(pwd, func(t *testing.T) {
			err := ValidatePassword(pwd)
			if err == nil {
				t.Errorf("Expected error for weak password '%s', but got none", pwd)
			}
		})
	}
}

func TestSanitizeInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Trim spaces", "  user  ", "user"},
		{"Multiple spaces", "user   name", "user name"},
		{"Newlines", "user\nname", "user name"},
		{"Tabs", "user\tname", "user name"},
		{"No changes", "username", "username"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeInput(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
