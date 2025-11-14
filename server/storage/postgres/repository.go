package postgres

import (
	"database/sql"
	"errors"
	"time"

	"github.com/buharamanya/gophkeeper/internal/models"
)

var (
	ErrDataNotFound    = errors.New("data not found")
	ErrVersionConflict = errors.New("version conflict")
)

type DataRepository struct {
	db *sql.DB
}

func NewDataRepository(db *sql.DB) *DataRepository {
	return &DataRepository{db: db}
}

func (r *DataRepository) CreateEntry(userID string, entry *models.DataEntry) error {
	err := r.db.QueryRow(
		`INSERT INTO data_entries (user_id, name, type, metadata, data, nonce, version, last_sync_time, created_at, updated_at) 
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) 
         RETURNING id`,
		userID, entry.Name, entry.Type, entry.Metadata, entry.Data, entry.Nonce,
		entry.Version, entry.LastSyncTime, entry.CreatedAt, entry.UpdatedAt,
	).Scan(&entry.ID)

	return err
}

func (r *DataRepository) GetUserEntries(userID string) ([]*models.DataEntry, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, name, type, metadata, data, nonce, version, created_at, updated_at, last_sync_time 
         FROM data_entries WHERE user_id = $1 AND is_deleted = false ORDER BY updated_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*models.DataEntry
	for rows.Next() {
		var entry models.DataEntry
		err := rows.Scan(
			&entry.ID, &entry.UserID, &entry.Name, &entry.Type, &entry.Metadata,
			&entry.Data, &entry.Nonce, &entry.Version, &entry.CreatedAt, &entry.UpdatedAt, &entry.LastSyncTime,
		)
		if err != nil {
			return nil, err
		}
		entries = append(entries, &entry)
	}

	return entries, nil
}

func (r *DataRepository) GetEntryByID(userID, entryID string) (*models.DataEntry, error) {
	var entry models.DataEntry
	err := r.db.QueryRow(
		`SELECT id, user_id, name, type, metadata, data, nonce, version, created_at, updated_at, last_sync_time 
         FROM data_entries WHERE id = $1 AND user_id = $2 AND is_deleted = false`,
		entryID, userID,
	).Scan(
		&entry.ID, &entry.UserID, &entry.Name, &entry.Type, &entry.Metadata,
		&entry.Data, &entry.Nonce, &entry.Version, &entry.CreatedAt, &entry.UpdatedAt, &entry.LastSyncTime,
	)

	if err == sql.ErrNoRows {
		return nil, ErrDataNotFound
	}
	if err != nil {
		return nil, err
	}

	return &entry, nil
}

func (r *DataRepository) UpdateEntry(userID string, entry *models.DataEntry) error {
	result, err := r.db.Exec(
		`UPDATE data_entries 
         SET name = $1, type = $2, metadata = $3, data = $4, nonce = $5, version = $6, 
             updated_at = $7, last_sync_time = $8
         WHERE id = $9 AND user_id = $10`,
		entry.Name, entry.Type, entry.Metadata, entry.Data, entry.Nonce, entry.Version,
		entry.UpdatedAt, entry.LastSyncTime, entry.ID, userID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrDataNotFound
	}

	return nil
}

// UpdateEntryWithVersion обновляет запись с проверкой версии
func (r *DataRepository) UpdateEntryWithVersion(userID string, entry *models.DataEntry, expectedVersion int64) error {
	result, err := r.db.Exec(
		`UPDATE data_entries 
         SET name = $1, type = $2, metadata = $3, data = $4, nonce = $5, version = $6, 
             updated_at = $7, last_sync_time = $8
         WHERE id = $9 AND user_id = $10 AND version = $11`,
		entry.Name, entry.Type, entry.Metadata, entry.Data, entry.Nonce, entry.Version,
		entry.UpdatedAt, entry.LastSyncTime, entry.ID, userID, expectedVersion,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		// Проверяем, существует ли запись
		var exists bool
		err := r.db.QueryRow(
			"SELECT EXISTS(SELECT 1 FROM data_entries WHERE id = $1 AND user_id = $2)",
			entry.ID, userID,
		).Scan(&exists)

		if err != nil {
			return err
		}

		if !exists {
			return ErrDataNotFound
		}

		return ErrVersionConflict
	}

	return nil
}

func (r *DataRepository) DeleteEntry(userID, entryID string) error {
	// Вместо физического удаления помечаем запись как удаленную
	result, err := r.db.Exec(
		`UPDATE data_entries SET is_deleted = true, updated_at = CURRENT_TIMESTAMP 
         WHERE id = $1 AND user_id = $2`,
		entryID, userID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrDataNotFound
	}

	return nil
}

// GetChangesSince возвращает изменения после указанного времени
func (r *DataRepository) GetChangesSince(userID string, since time.Time) ([]*models.DataEntry, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, name, type, metadata, data, nonce, version, created_at, updated_at, last_sync_time 
         FROM data_entries 
         WHERE user_id = $1 AND updated_at > $2 AND is_deleted = false 
         ORDER BY updated_at DESC`,
		userID, since,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*models.DataEntry
	for rows.Next() {
		var entry models.DataEntry
		err := rows.Scan(
			&entry.ID, &entry.UserID, &entry.Name, &entry.Type, &entry.Metadata,
			&entry.Data, &entry.Nonce, &entry.Version, &entry.CreatedAt, &entry.UpdatedAt, &entry.LastSyncTime,
		)
		if err != nil {
			return nil, err
		}
		entries = append(entries, &entry)
	}

	return entries, nil
}

// GetSyncStatus возвращает статус синхронизации
func (r *DataRepository) GetSyncStatus(userID string) (time.Time, int, bool, error) {
	// В реальной реализации здесь была бы более сложная логика
	// Пока возвращаем заглушки
	return time.Now().Add(-24 * time.Hour), 0, false, nil
}

// ResolveConflict разрешает конфликт синхронизации
func (r *DataRepository) ResolveConflict(userID, conflictID, resolution string, entry *models.DataEntry) error {
	// В реальной реализации здесь была бы логика разрешения конфликтов
	// Пока просто обновляем запись
	return r.UpdateEntry(userID, entry)
}
