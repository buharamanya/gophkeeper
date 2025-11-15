package postgres

import (
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/buharamanya/gophkeeper/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRepository_CreateUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserRepository(db)

	t.Run("success", func(t *testing.T) {
		login := "testuser"
		password := "testpass"
		expectedUser := &models.User{
			ID:           "123",
			Login:        login,
			PasswordHash: "hashed_password",
			CreatedAt:    time.Now(),
		}

		mock.ExpectQuery("INSERT INTO users").
			WithArgs(login, sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "login", "password_hash", "created_at"}).
				AddRow(expectedUser.ID, expectedUser.Login, expectedUser.PasswordHash, expectedUser.CreatedAt))

		user, err := repo.CreateUser(login, password)

		require.NoError(t, err)
		assert.Equal(t, expectedUser, user)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user already exists", func(t *testing.T) {
		login := "existinguser"
		password := "testpass"

		mock.ExpectQuery("INSERT INTO users").
			WithArgs(login, sqlmock.AnyArg()).
			WillReturnError(sql.ErrConnDone) // В реальности здесь будет ошибка уникальности

		user, err := repo.CreateUser(login, password)

		assert.Error(t, err)
		assert.Nil(t, user)
	})
}

func TestUserRepository_GetUserByLogin(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserRepository(db)

	t.Run("success", func(t *testing.T) {
		login := "testuser"
		expectedUser := &models.User{
			ID:           "123",
			Login:        login,
			PasswordHash: "hashed_password",
			CreatedAt:    time.Now(),
		}

		mock.ExpectQuery("SELECT id, login, password_hash, created_at FROM users WHERE login = ?").
			WithArgs(login).
			WillReturnRows(sqlmock.NewRows([]string{"id", "login", "password_hash", "created_at"}).
				AddRow(expectedUser.ID, expectedUser.Login, expectedUser.PasswordHash, expectedUser.CreatedAt))

		user, err := repo.GetUserByLogin(login)

		require.NoError(t, err)
		assert.Equal(t, expectedUser, user)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user not found", func(t *testing.T) {
		login := "nonexistent"

		mock.ExpectQuery("SELECT id, login, password_hash, created_at FROM users WHERE login = ?").
			WithArgs(login).
			WillReturnError(sql.ErrNoRows)

		user, err := repo.GetUserByLogin(login)

		assert.ErrorIs(t, err, ErrUserNotFound)
		assert.Nil(t, user)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_GetUserByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserRepository(db)

	t.Run("success", func(t *testing.T) {
		id := "123"
		expectedUser := &models.User{
			ID:           id,
			Login:        "testuser",
			PasswordHash: "hashed_password",
			CreatedAt:    time.Now(),
		}

		mock.ExpectQuery("SELECT id, login, password_hash, created_at FROM users WHERE id = ?").
			WithArgs(id).
			WillReturnRows(sqlmock.NewRows([]string{"id", "login", "password_hash", "created_at"}).
				AddRow(expectedUser.ID, expectedUser.Login, expectedUser.PasswordHash, expectedUser.CreatedAt))

		user, err := repo.GetUserByID(id)

		require.NoError(t, err)
		assert.Equal(t, expectedUser, user)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user not found", func(t *testing.T) {
		id := "nonexistent"

		mock.ExpectQuery("SELECT id, login, password_hash, created_at FROM users WHERE id = ?").
			WithArgs(id).
			WillReturnError(sql.ErrNoRows)

		user, err := repo.GetUserByID(id)

		assert.ErrorIs(t, err, ErrUserNotFound)
		assert.Nil(t, user)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
