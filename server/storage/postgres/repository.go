package postgres

import (
	"database/sql"
	"errors"

	"github.com/buharamanya/gophkeeper/internal/models"
)

var ErrDataNotFound = errors.New("data not found")

type DataRepository struct {
	db *sql.DB
}

func NewDataRepository(db *sql.DB) *DataRepository {
	return &DataRepository{db: db}
}

func (r *DataRepository) CreateEntry(userID string, entry *models.DataEntry) error {
	err := r.db.QueryRow(
		`INSERT INTO data_entries (user_id, name, type, metadata, data, nonce, version) 
         VALUES ($1, $2, $3, $4, $5, $6, $7) 
         RETURNING id, created_at, updated_at`,
		userID, entry.Name, entry.Type, entry.Metadata, entry.Data, entry.Nonce, entry.Version,
	).Scan(&entry.ID, &entry.CreatedAt, &entry.UpdatedAt)

	return err
}

func (r *DataRepository) GetUserEntries(userID string) ([]*models.DataEntry, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, name, type, metadata, data, nonce, version, created_at, updated_at 
         FROM data_entries WHERE user_id = $1 ORDER BY created_at DESC`,
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
			&entry.Data, &entry.Nonce, &entry.Version, &entry.CreatedAt, &entry.UpdatedAt,
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
		`SELECT id, user_id, name, type, metadata, data, nonce, version, created_at, updated_at 
         FROM data_entries WHERE id = $1 AND user_id = $2`,
		entryID, userID,
	).Scan(
		&entry.ID, &entry.UserID, &entry.Name, &entry.Type, &entry.Metadata,
		&entry.Data, &entry.Nonce, &entry.Version, &entry.CreatedAt, &entry.UpdatedAt,
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
         SET name = $1, type = $2, metadata = $3, data = $4, nonce = $5, version = $6, updated_at = CURRENT_TIMESTAMP 
         WHERE id = $7 AND user_id = $8`,
		entry.Name, entry.Type, entry.Metadata, entry.Data, entry.Nonce, entry.Version+1,
		entry.ID, userID,
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

func (r *DataRepository) DeleteEntry(userID, entryID string) error {
	result, err := r.db.Exec(
		"DELETE FROM data_entries WHERE id = $1 AND user_id = $2",
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
