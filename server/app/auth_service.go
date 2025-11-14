package app

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/buharamanya/gophkeeper/internal/crypto"
	"github.com/buharamanya/gophkeeper/internal/models"
	"github.com/buharamanya/gophkeeper/server/storage/postgres"
	"github.com/golang-jwt/jwt/v4"
)

type AuthService struct {
	userRepo  *postgres.UserRepository
	jwtSecret string
}

func NewAuthService(userRepo *postgres.UserRepository, jwtSecret string) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
	}
}

type Claims struct {
	UserID string `json:"user_id"`
	Login  string `json:"login"`
	jwt.RegisteredClaims
}

// ValidateLogin проверяет логин на соответствие требованиям
func ValidateLogin(login string) error {
	if login == "" {
		return errors.New("логин не может быть пустым")
	}

	if len(login) < 3 {
		return errors.New("логин должен содержать минимум 3 символа")
	}

	if len(login) > 50 {
		return errors.New("логин не может превышать 50 символов")
	}

	// Проверка на допустимые символы (только буквы, цифры, подчеркивание, дефис)
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, login)
	if !matched {
		return errors.New("логин может содержать только буквы (a-z, A-Z), цифры (0-9), подчеркивание (_) и дефис (-)")
	}

	// Проверка, что логин не начинается с цифры или специального символа
	if !unicode.IsLetter(rune(login[0])) {
		return errors.New("логин должен начинаться с буквы")
	}

	return nil
}

// ValidatePassword проверяет пароль на соответствие требованиям безопасности
func ValidatePassword(password string) error {
	if password == "" {
		return errors.New("пароль не может быть пустым")
	}

	if len(password) < 8 {
		return errors.New("пароль должен содержать минимум 8 символов")
	}

	if len(password) > 72 { // bcrypt ограничение
		return errors.New("пароль не может превышать 72 символа")
	}

	var (
		hasUpper   = false
		hasLower   = false
		hasNumber  = false
		hasSpecial = false
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	// Проверка сложности пароля
	if !hasUpper {
		return errors.New("пароль должен содержать хотя бы одну заглавную букву")
	}

	if !hasLower {
		return errors.New("пароль должен содержать хотя бы одну строчную букву")
	}

	if !hasNumber {
		return errors.New("пароль должен содержать хотя бы одну цифру")
	}

	if !hasSpecial {
		return errors.New("пароль должен содержать хотя бы один специальный символ (например: !@#$%^&*)")
	}

	// Проверка на распространенные слабые пароли
	weakPasswords := map[string]bool{
		"password": true, "12345678": true, "qwerty": true,
		"admin": true, "welcome": true, "monkey": true,
		"letmein": true, "master": true, "hello": true,
	}

	// Приводим к нижнему регистру для проверки
	if weakPasswords[strings.ToLower(password)] {
		return errors.New("пароль слишком распространен, выберите другой")
	}

	return nil
}

// SanitizeInput очищает входные данные от потенциально опасных символов
func SanitizeInput(input string) string {
	// Удаляем начальные и конечные пробелы
	input = regexp.MustCompile(`^\s+|\s+$`).ReplaceAllString(input, "")
	// Заменяем множественные пробелы на один
	input = regexp.MustCompile(`\s+`).ReplaceAllString(input, " ")
	return input
}

func (s *AuthService) Register(login, password string) (*models.AuthResponse, error) {
	// Очистка входных данных
	login = SanitizeInput(login)
	password = SanitizeInput(password)

	// Валидация логина
	if err := ValidateLogin(login); err != nil {
		return nil, err
	}

	// Валидация пароля
	if err := ValidatePassword(password); err != nil {
		return nil, err
	}

	user, err := s.userRepo.CreateUser(login, password)
	if err != nil {
		return nil, err
	}

	token, err := s.generateToken(user.ID, user.Login)
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		Token:  token,
		UserID: user.ID,
	}, nil
}

func (s *AuthService) Login(login, password string) (*models.AuthResponse, error) {
	// Очистка входных данных
	login = SanitizeInput(login)
	password = SanitizeInput(password)

	// Базовая валидация при логине
	if login == "" || password == "" {
		return nil, errors.New("логин и пароль не могут быть пустыми")
	}

	user, err := s.userRepo.GetUserByLogin(login)
	if err != nil {
		// Возвращаем одинаковую ошибку для security (не раскрываем, существует ли пользователь)
		return nil, errors.New("неверный логин или пароль")
	}

	if !crypto.CheckPasswordHash(password, user.PasswordHash) {
		return nil, errors.New("неверный логин или пароль")
	}

	token, err := s.generateToken(user.ID, user.Login)
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		Token:  token,
		UserID: user.ID,
	}, nil
}

func (s *AuthService) generateToken(userID, login string) (string, error) {
	// Валидация входных данных
	if userID == "" || login == "" {
		return "", errors.New("userID and login cannot be empty")
	}

	claims := &Claims{
		UserID: userID,
		Login:  login,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "gophkeeper-server",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	expectedSigningMethod := jwt.SigningMethodHS256

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != expectedSigningMethod.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		// Дополнительная валидация claims
		if claims.UserID == "" || claims.Login == "" {
			return nil, errors.New("invalid token claims: missing userID or login")
		}

		// Проверка срока действия
		if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
			return nil, errors.New("token has expired")
		}

		return claims, nil
	}

	return nil, errors.New("invalid token")
}
