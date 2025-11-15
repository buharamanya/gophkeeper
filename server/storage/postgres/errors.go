package postgres

import (
	"errors"
	"fmt"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
)

// HandlePgError обрабатывает ошибки PostgreSQL и возвращает соответствующие ошибки приложения
func HandlePgError(err error, context string) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// Обрабатываем специфичные коды ошибок
		switch pgErr.Code {
		case pgerrcode.UniqueViolation:
			if context == "users" {
				return ErrUserAlreadyExists
			}
			return fmt.Errorf("%s: data already exists", context)

		case pgerrcode.ForeignKeyViolation:
			return fmt.Errorf("%s: referenced data not found", context)

		case pgerrcode.NotNullViolation:
			return fmt.Errorf("%s: required field is missing", context)

		case pgerrcode.CheckViolation:
			return fmt.Errorf("%s: data validation failed", context)

		case "22001": // String data right truncation
			return fmt.Errorf("%s: data too long for field", context)

		case pgerrcode.NumericValueOutOfRange:
			return fmt.Errorf("%s: numeric value out of range", context)

		case pgerrcode.InvalidTextRepresentation:
			return fmt.Errorf("%s: invalid data format", context)

		case pgerrcode.DuplicateTable, pgerrcode.DuplicateObject:
			return fmt.Errorf("%s: object already exists", context)

		case pgerrcode.UndefinedTable:
			return fmt.Errorf("%s: database table does not exist", context)

		case pgerrcode.UndefinedColumn:
			return fmt.Errorf("%s: database column does not exist", context)

		case pgerrcode.InsufficientPrivilege:
			return fmt.Errorf("%s: insufficient database privileges", context)

		case pgerrcode.ConnectionException, pgerrcode.ConnectionDoesNotExist, pgerrcode.ConnectionFailure:
			return fmt.Errorf("%s: database connection error", context)

		case pgerrcode.AdminShutdown, pgerrcode.CrashShutdown, pgerrcode.CannotConnectNow:
			return fmt.Errorf("%s: database is unavailable", context)

		case pgerrcode.OutOfMemory:
			return fmt.Errorf("%s: database out of memory", context)

		case pgerrcode.DiskFull:
			return fmt.Errorf("%s: database disk full", context)

		case pgerrcode.ProgramLimitExceeded:
			return fmt.Errorf("%s: database program limit exceeded", context)

		case pgerrcode.InternalError:
			return fmt.Errorf("%s: internal database error", context)

		case pgerrcode.DataCorrupted:
			return fmt.Errorf("%s: database data corrupted", context)

		default:
			// Для неизвестных ошибок возвращаем общее сообщение с кодом
			return fmt.Errorf("%s: database error [%s]", context, pgErr.Code)
		}
	}

	// Если это не PostgreSQL ошибка, возвращаем как есть
	return fmt.Errorf("%s: %w", context, err)
}

// IsConnectionError проверяет, является ли ошибка ошибкой соединения с БД
func IsConnectionError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgerrcode.ConnectionException, pgerrcode.ConnectionDoesNotExist,
			pgerrcode.ConnectionFailure, pgerrcode.CannotConnectNow:
			return true
		}
	}
	return false
}

// IsConstraintError проверяет, является ли ошибка ошибкой ограничений
func IsConstraintError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgerrcode.UniqueViolation, pgerrcode.ForeignKeyViolation,
			pgerrcode.NotNullViolation, pgerrcode.CheckViolation:
			return true
		}
	}
	return false
}

// GetPgErrorCode возвращает код ошибки PostgreSQL если ошибка является PgError
func GetPgErrorCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

// GetPgErrorDetails возвращает детали ошибки PostgreSQL для логирования
func GetPgErrorDetails(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return fmt.Sprintf("PostgreSQL error [%s]: %s (Detail: %s, Hint: %s, Where: %s)",
			pgErr.Code, pgErr.Message, pgErr.Detail, pgErr.Hint, pgErr.Where)
	}
	return ""
}
