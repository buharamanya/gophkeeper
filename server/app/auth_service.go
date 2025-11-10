package app

import (
	"errors"
	"time"

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

func (s *AuthService) Register(login, password string) (*models.AuthResponse, error) {
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
	user, err := s.userRepo.GetUserByLogin(login)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if !crypto.CheckPasswordHash(password, user.PasswordHash) {
		return nil, errors.New("invalid credentials")
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
	claims := &Claims{
		UserID: userID,
		Login:  login,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
