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

	if err != nil {
		return HandlePgError(err, "create data entry")
	}

	return nil
}

func (r *DataRepository) GetUserEntries(userID string) ([]*models.DataEntry, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, name, type, metadata, data, nonce, version, created_at, updated_at, last_sync_time 
         FROM data_entries WHERE user_id = $1 AND is_deleted = false ORDER BY updated_at DESC`,
		userID,
	)
	if err != nil {
		return nil, HandlePgError(err, "get user entries")
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
			return nil, HandlePgError(err, "scan data entry")
		}
		entries = append(entries, &entry)
	}

	if err := rows.Err(); err != nil {
		return nil, HandlePgError(err, "iterate user entries")
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

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDataNotFound
		}
		return nil, HandlePgError(err, "get data entry by id")
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
		return HandlePgError(err, "update data entry")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return HandlePgError(err, "get rows affected")
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
		return HandlePgError(err, "update data entry with version check")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return HandlePgError(err, "get rows affected for version check")
	}

	if rowsAffected == 0 {
		// Проверяем, существует ли запись
		var exists bool
		err := r.db.QueryRow(
			"SELECT EXISTS(SELECT 1 FROM data_entries WHERE id = $1 AND user_id = $2)",
			entry.ID, userID,
		).Scan(&exists)

		if err != nil {
			return HandlePgError(err, "check data entry existence")
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
		return HandlePgError(err, "delete data entry")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return HandlePgError(err, "get rows affected for delete")
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
		return nil, HandlePgError(err, "get changes since")
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
			return nil, HandlePgError(err, "scan changed entry")
		}
		entries = append(entries, &entry)
	}

	if err := rows.Err(); err != nil {
		return nil, HandlePgError(err, "iterate changed entries")
	}

	return entries, nil
}

// GetSyncStatus возвращает статус синхронизации
func (r *DataRepository) GetSyncStatus(userID string) (time.Time, int, bool, error) {
	// Получаем время последней синхронизации
	var lastSync sql.NullTime
	err := r.db.QueryRow(
		`SELECT MAX(last_sync_time) FROM data_entries WHERE user_id = $1`,
		userID,
	).Scan(&lastSync)
	if err != nil {
		return time.Time{}, 0, false, HandlePgError(err, "get last sync time")
	}

	// Получаем количество ожидающих изменений
	var pendingChanges int
	err = r.db.QueryRow(
		`SELECT COUNT(*) FROM data_entries WHERE user_id = $1 AND is_deleted = false`,
		userID,
	).Scan(&pendingChanges)
	if err != nil {
		return time.Time{}, 0, false, HandlePgError(err, "get pending changes count")
	}

	// Проверяем наличие конфликтов (упрощенная логика)
	var hasConflicts bool
	err = r.db.QueryRow(
		`SELECT EXISTS(
			SELECT 1 FROM data_entries 
			WHERE user_id = $1 AND is_deleted = false 
			AND last_sync_time < updated_at
		)`,
		userID,
	).Scan(&hasConflicts)
	if err != nil {
		return time.Time{}, 0, false, HandlePgError(err, "check conflicts")
	}

	if lastSync.Valid {
		return lastSync.Time, pendingChanges, hasConflicts, nil
	}
	return time.Time{}, pendingChanges, hasConflicts, nil
}

// ResolveConflict разрешает конфликт синхронизации
func (r *DataRepository) ResolveConflict(userID, conflictID, resolution string, entry *models.DataEntry) error {
	return HandlePgError(r.UpdateEntry(userID, entry), "resolve conflict")
}
