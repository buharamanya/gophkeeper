package postgres

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/buharamanya/gophkeeper/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDataRepository_CreateEntry(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewDataRepository(db)

	entry := &models.DataEntry{
		Name:         "Test Entry",
		Type:         models.TypeLoginPassword,
		Metadata:     "metadata",
		Data:         []byte("data"),
		Nonce:        []byte("nonce"),
		Version:      1,
		LastSyncTime: time.Now(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	tests := []struct {
		name          string
		setupMock     func()
		expectedID    string
		expectedError error
	}{
		{
			name: "successful creation",
			setupMock: func() {
				mock.ExpectQuery(`INSERT INTO data_entries`).
					WithArgs("user123", entry.Name, entry.Type, entry.Metadata, entry.Data, entry.Nonce,
						entry.Version, entry.LastSyncTime, entry.CreatedAt, entry.UpdatedAt).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("test-id-123"))
			},
			expectedID: "test-id-123",
		},
		{
			name: "database error",
			setupMock: func() {
				mock.ExpectQuery(`INSERT INTO data_entries`).
					WithArgs("user123", entry.Name, entry.Type, entry.Metadata, entry.Data, entry.Nonce,
						entry.Version, entry.LastSyncTime, entry.CreatedAt, entry.UpdatedAt).
					WillReturnError(errors.New("db error"))
			},
			expectedError: errors.New("db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			err := repo.CreateEntry("user123", entry)

			if tt.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, entry.ID)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDataRepository_GetUserEntries(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewDataRepository(db)

	expectedTime := time.Now()
	expectedEntries := []*models.DataEntry{
		{
			ID:           "entry1",
			UserID:       "user123",
			Name:         "Entry 1",
			Type:         models.TypeLoginPassword,
			Metadata:     "meta1",
			Data:         []byte("data1"),
			Nonce:        []byte("nonce1"),
			Version:      1,
			CreatedAt:    expectedTime,
			UpdatedAt:    expectedTime,
			LastSyncTime: expectedTime,
		},
	}

	tests := []struct {
		name          string
		setupMock     func()
		expectedError error
	}{
		{
			name: "successful get",
			setupMock: func() {
				rows := sqlmock.NewRows([]string{
					"id", "user_id", "name", "type", "metadata", "data", "nonce",
					"version", "created_at", "updated_at", "last_sync_time",
				}).AddRow(
					"entry1", "user123", "Entry 1", models.TypeLoginPassword, "meta1",
					[]byte("data1"), []byte("nonce1"), int64(1),
					expectedTime, expectedTime, expectedTime,
				)
				mock.ExpectQuery(`SELECT.*FROM data_entries WHERE user_id = \$1 AND is_deleted = false`).
					WithArgs("user123").
					WillReturnRows(rows)
			},
		},
		{
			name: "database error",
			setupMock: func() {
				mock.ExpectQuery(`SELECT.*FROM data_entries WHERE user_id = \$1 AND is_deleted = false`).
					WithArgs("user123").
					WillReturnError(errors.New("db error"))
			},
			expectedError: errors.New("db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			entries, err := repo.GetUserEntries("user123")

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Nil(t, entries)
			} else {
				assert.NoError(t, err)
				assert.Len(t, entries, 1)
				assert.Equal(t, expectedEntries[0].ID, entries[0].ID)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDataRepository_GetEntryByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewDataRepository(db)

	expectedTime := time.Now()
	expectedEntry := &models.DataEntry{
		ID:           "entry1",
		UserID:       "user123",
		Name:         "Test Entry",
		Type:         models.TypeLoginPassword,
		Metadata:     "metadata",
		Data:         []byte("data"),
		Nonce:        []byte("nonce"),
		Version:      1,
		CreatedAt:    expectedTime,
		UpdatedAt:    expectedTime,
		LastSyncTime: expectedTime,
	}

	tests := []struct {
		name          string
		setupMock     func()
		expectedError error
	}{
		{
			name: "successful get",
			setupMock: func() {
				rows := sqlmock.NewRows([]string{
					"id", "user_id", "name", "type", "metadata", "data", "nonce",
					"version", "created_at", "updated_at", "last_sync_time",
				}).AddRow(
					"entry1", "user123", "Test Entry", models.TypeLoginPassword, "metadata",
					[]byte("data"), []byte("nonce"), int64(1),
					expectedTime, expectedTime, expectedTime,
				)
				mock.ExpectQuery(`SELECT.*FROM data_entries WHERE id = \$1 AND user_id = \$2 AND is_deleted = false`).
					WithArgs("entry1", "user123").
					WillReturnRows(rows)
			},
		},
		{
			name: "entry not found",
			setupMock: func() {
				mock.ExpectQuery(`SELECT.*FROM data_entries WHERE id = \$1 AND user_id = \$2 AND is_deleted = false`).
					WithArgs("entry1", "user123").
					WillReturnError(sql.ErrNoRows)
			},
			expectedError: ErrDataNotFound,
		},
		{
			name: "database error",
			setupMock: func() {
				mock.ExpectQuery(`SELECT.*FROM data_entries WHERE id = \$1 AND user_id = \$2 AND is_deleted = false`).
					WithArgs("entry1", "user123").
					WillReturnError(errors.New("db error"))
			},
			expectedError: errors.New("db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			entry, err := repo.GetEntryByID("user123", "entry1")

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Nil(t, entry)
				if errors.Is(tt.expectedError, ErrDataNotFound) {
					assert.Equal(t, ErrDataNotFound, err)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, expectedEntry.ID, entry.ID)
				assert.Equal(t, expectedEntry.Name, entry.Name)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDataRepository_UpdateEntry(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewDataRepository(db)

	entry := &models.DataEntry{
		ID:           "entry1",
		Name:         "Updated Entry",
		Type:         models.TypeText,
		Metadata:     "updated metadata",
		Data:         []byte("updated data"),
		Nonce:        []byte("updated nonce"),
		Version:      2,
		UpdatedAt:    time.Now(),
		LastSyncTime: time.Now(),
	}

	tests := []struct {
		name          string
		setupMock     func()
		expectedError error
	}{
		{
			name: "successful update",
			setupMock: func() {
				mock.ExpectExec(`UPDATE data_entries`).
					WithArgs(
						entry.Name, entry.Type, entry.Metadata, entry.Data, entry.Nonce, entry.Version,
						entry.UpdatedAt, entry.LastSyncTime, entry.ID, "user123",
					).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
		{
			name: "entry not found",
			setupMock: func() {
				mock.ExpectExec(`UPDATE data_entries`).
					WithArgs(
						entry.Name, entry.Type, entry.Metadata, entry.Data, entry.Nonce, entry.Version,
						entry.UpdatedAt, entry.LastSyncTime, entry.ID, "user123",
					).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedError: ErrDataNotFound,
		},
		{
			name: "database error",
			setupMock: func() {
				mock.ExpectExec(`UPDATE data_entries`).
					WithArgs(
						entry.Name, entry.Type, entry.Metadata, entry.Data, entry.Nonce, entry.Version,
						entry.UpdatedAt, entry.LastSyncTime, entry.ID, "user123",
					).
					WillReturnError(errors.New("db error"))
			},
			expectedError: errors.New("db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			err := repo.UpdateEntry("user123", entry)

			if tt.expectedError != nil {
				assert.Error(t, err)
				if errors.Is(tt.expectedError, ErrDataNotFound) {
					assert.Equal(t, ErrDataNotFound, err)
				}
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDataRepository_UpdateEntryWithVersion(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewDataRepository(db)

	entry := &models.DataEntry{
		ID:           "entry1",
		Name:         "Updated Entry",
		Type:         models.TypeText,
		Metadata:     "updated metadata",
		Data:         []byte("updated data"),
		Nonce:        []byte("updated nonce"),
		Version:      2,
		UpdatedAt:    time.Now(),
		LastSyncTime: time.Now(),
	}

	tests := []struct {
		name          string
		setupMock     func()
		expectedError error
	}{
		{
			name: "successful update with version",
			setupMock: func() {
				mock.ExpectExec(`UPDATE data_entries`).
					WithArgs(
						entry.Name, entry.Type, entry.Metadata, entry.Data, entry.Nonce, entry.Version,
						entry.UpdatedAt, entry.LastSyncTime, entry.ID, "user123", int64(1),
					).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
		{
			name: "version conflict - entry exists",
			setupMock: func() {
				mock.ExpectExec(`UPDATE data_entries`).
					WithArgs(
						entry.Name, entry.Type, entry.Metadata, entry.Data, entry.Nonce, entry.Version,
						entry.UpdatedAt, entry.LastSyncTime, entry.ID, "user123", int64(1),
					).
					WillReturnResult(sqlmock.NewResult(0, 0))

				mock.ExpectQuery(`SELECT EXISTS`).
					WithArgs(entry.ID, "user123").
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
			},
			expectedError: ErrVersionConflict,
		},
		{
			name: "entry not found",
			setupMock: func() {
				mock.ExpectExec(`UPDATE data_entries`).
					WithArgs(
						entry.Name, entry.Type, entry.Metadata, entry.Data, entry.Nonce, entry.Version,
						entry.UpdatedAt, entry.LastSyncTime, entry.ID, "user123", int64(1),
					).
					WillReturnResult(sqlmock.NewResult(0, 0))

				mock.ExpectQuery(`SELECT EXISTS`).
					WithArgs(entry.ID, "user123").
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
			},
			expectedError: ErrDataNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			err := repo.UpdateEntryWithVersion("user123", entry, 1)

			if tt.expectedError != nil {
				assert.Error(t, err)
				if errors.Is(tt.expectedError, ErrVersionConflict) {
					assert.Equal(t, ErrVersionConflict, err)
				} else if errors.Is(tt.expectedError, ErrDataNotFound) {
					assert.Equal(t, ErrDataNotFound, err)
				}
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDataRepository_DeleteEntry(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewDataRepository(db)

	tests := []struct {
		name          string
		setupMock     func()
		expectedError error
	}{
		{
			name: "successful delete",
			setupMock: func() {
				mock.ExpectExec(`UPDATE data_entries SET is_deleted = true`).
					WithArgs("entry1", "user123").
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
		{
			name: "entry not found",
			setupMock: func() {
				mock.ExpectExec(`UPDATE data_entries SET is_deleted = true`).
					WithArgs("entry1", "user123").
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedError: ErrDataNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			err := repo.DeleteEntry("user123", "entry1")

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, ErrDataNotFound, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDataRepository_GetChangesSince(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewDataRepository(db)

	since := time.Now().Add(-time.Hour)
	expectedTime := time.Now()

	tests := []struct {
		name           string
		setupMock      func()
		expectedLength int
		expectedError  error
	}{
		{
			name: "successful get changes",
			setupMock: func() {
				rows := sqlmock.NewRows([]string{
					"id", "user_id", "name", "type", "metadata", "data", "nonce",
					"version", "created_at", "updated_at", "last_sync_time",
				}).AddRow(
					"entry1", "user123", "Updated Entry", models.TypeText, "metadata",
					[]byte("data"), []byte("nonce"), int64(2),
					expectedTime, expectedTime, expectedTime,
				)
				mock.ExpectQuery(`SELECT.*FROM data_entries WHERE user_id = \$1 AND updated_at > \$2 AND is_deleted = false`).
					WithArgs("user123", since).
					WillReturnRows(rows)
			},
			expectedLength: 1,
		},
		{
			name: "no changes - returns empty slice",
			setupMock: func() {
				rows := sqlmock.NewRows([]string{
					"id", "user_id", "name", "type", "metadata", "data", "nonce",
					"version", "created_at", "updated_at", "last_sync_time",
				})
				mock.ExpectQuery(`SELECT.*FROM data_entries WHERE user_id = \$1 AND updated_at > \$2 AND is_deleted = false`).
					WithArgs("user123", since).
					WillReturnRows(rows)
			},
			expectedLength: 0,
		},
		{
			name: "database error",
			setupMock: func() {
				mock.ExpectQuery(`SELECT.*FROM data_entries WHERE user_id = \$1 AND updated_at > \$2 AND is_deleted = false`).
					WithArgs("user123", since).
					WillReturnError(errors.New("db error"))
			},
			expectedError: errors.New("db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			entries, err := repo.GetChangesSince("user123", since)

			if tt.expectedError != nil {
				assert.Error(t, err)
				// При ошибке entries может быть nil
			} else {
				assert.NoError(t, err)
				// entries может быть nil или пустым слайсом
				if entries == nil {
					assert.Equal(t, 0, tt.expectedLength, "Expected empty slice, got nil")
				} else {
					assert.Len(t, entries, tt.expectedLength)
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDataRepository_GetSyncStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewDataRepository(db)

	lastSyncTime := time.Now().Add(-time.Hour)

	tests := []struct {
		name              string
		setupMock         func()
		expectedLastSync  time.Time
		expectedPending   int
		expectedConflicts bool
		expectedError     error
	}{
		{
			name: "successful get sync status with last sync",
			setupMock: func() {
				// Mock для получения времени последней синхронизации
				mock.ExpectQuery(`SELECT MAX\(last_sync_time\) FROM data_entries WHERE user_id = \$1`).
					WithArgs("user123").
					WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(lastSyncTime))

				// Mock для получения количества ожидающих изменений
				mock.ExpectQuery(`SELECT COUNT\(\*\) FROM data_entries WHERE user_id = \$1 AND is_deleted = false`).
					WithArgs("user123").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

				// Mock для проверки конфликтов
				mock.ExpectQuery(`SELECT EXISTS`).
					WithArgs("user123").
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
			},
			expectedLastSync:  lastSyncTime,
			expectedPending:   5,
			expectedConflicts: true,
		},
		{
			name: "successful get sync status without last sync",
			setupMock: func() {
				// Mock для получения времени последней синхронизации (NULL)
				mock.ExpectQuery(`SELECT MAX\(last_sync_time\) FROM data_entries WHERE user_id = \$1`).
					WithArgs("user123").
					WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(nil))

				// Mock для получения количества ожидающих изменений
				mock.ExpectQuery(`SELECT COUNT\(\*\) FROM data_entries WHERE user_id = \$1 AND is_deleted = false`).
					WithArgs("user123").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

				// Mock для проверки конфликтов
				mock.ExpectQuery(`SELECT EXISTS`).
					WithArgs("user123").
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
			},
			expectedLastSync:  time.Time{},
			expectedPending:   0,
			expectedConflicts: false,
		},
		{
			name: "error getting last sync time",
			setupMock: func() {
				mock.ExpectQuery(`SELECT MAX\(last_sync_time\) FROM data_entries WHERE user_id = \$1`).
					WithArgs("user123").
					WillReturnError(errors.New("db error"))
			},
			expectedError: errors.New("db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			lastSync, pending, conflicts, err := repo.GetSyncStatus("user123")

			if tt.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedLastSync, lastSync)
				assert.Equal(t, tt.expectedPending, pending)
				assert.Equal(t, tt.expectedConflicts, conflicts)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDataRepository_ResolveConflict(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewDataRepository(db)

	entry := &models.DataEntry{
		ID:           "entry1",
		Name:         "Resolved Entry",
		Type:         models.TypeLoginPassword,
		Metadata:     "resolved metadata",
		Data:         []byte("resolved data"),
		Nonce:        []byte("resolved nonce"),
		Version:      3,
		UpdatedAt:    time.Now(),
		LastSyncTime: time.Now(),
	}

	tests := []struct {
		name          string
		setupMock     func()
		expectedError error
	}{
		{
			name: "successful conflict resolution",
			setupMock: func() {
				mock.ExpectExec(`UPDATE data_entries`).
					WithArgs(
						entry.Name, entry.Type, entry.Metadata, entry.Data, entry.Nonce, entry.Version,
						entry.UpdatedAt, entry.LastSyncTime, entry.ID, "user123",
					).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
		{
			name: "error during conflict resolution",
			setupMock: func() {
				mock.ExpectExec(`UPDATE data_entries`).
					WithArgs(
						entry.Name, entry.Type, entry.Metadata, entry.Data, entry.Nonce, entry.Version,
						entry.UpdatedAt, entry.LastSyncTime, entry.ID, "user123",
					).
					WillReturnError(errors.New("update error"))
			},
			expectedError: errors.New("update error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			err := repo.ResolveConflict("user123", "conflict-123", "client", entry)

			if tt.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDataRepository_ErrorConstants(t *testing.T) {
	assert.NotEmpty(t, ErrDataNotFound.Error())
	assert.NotEmpty(t, ErrVersionConflict.Error())
}
