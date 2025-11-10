package postgres

import (
	"database/sql"
	"errors"

	"github.com/buharamanya/gophkeeper/internal/crypto"
	"github.com/buharamanya/gophkeeper/internal/models"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateUser(login, password string) (*models.User, error) {
	passwordHash, err := crypto.HashPassword(password)
	if err != nil {
		return nil, err
	}

	var user models.User
	err = r.db.QueryRow(
		"INSERT INTO users (login, password_hash) VALUES ($1, $2) RETURNING id, login, password_hash, created_at",
		login, passwordHash,
	).Scan(&user.ID, &user.Login, &user.PasswordHash, &user.CreatedAt)

	if err != nil {
		if err.Error() == "pq: duplicate key value violates unique constraint \"users_login_key\"" {
			return nil, ErrUserAlreadyExists
		}
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) GetUserByLogin(login string) (*models.User, error) {
	var user models.User
	err := r.db.QueryRow(
		"SELECT id, login, password_hash, created_at FROM users WHERE login = $1",
		login,
	).Scan(&user.ID, &user.Login, &user.PasswordHash, &user.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) GetUserByID(id string) (*models.User, error) {
	var user models.User
	err := r.db.QueryRow(
		"SELECT id, login, password_hash, created_at FROM users WHERE id = $1",
		id,
	).Scan(&user.ID, &user.Login, &user.PasswordHash, &user.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}
