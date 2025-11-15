package app

import (
	"errors"
	"testing"
	"time"

	"github.com/buharamanya/gophkeeper/internal/models"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// DataRepositoryInterface для тестов
type DataRepositoryInterface interface {
	CreateEntry(userID string, entry *models.DataEntry) error
	GetUserEntries(userID string) ([]*models.DataEntry, error)
	GetEntryByID(userID, entryID string) (*models.DataEntry, error)
	UpdateEntry(userID string, entry *models.DataEntry) error
	DeleteEntry(userID, entryID string) error
	GetChangesSince(userID string, since time.Time) ([]*models.DataEntry, error)
	GetSyncStatus(userID string) (time.Time, int, bool, error)
	ResolveConflict(userID, conflictID, resolution string, entry *models.DataEntry) error
}

// TestDataService для тестов
type TestDataService struct {
	dataRepo DataRepositoryInterface
	logger   *zap.Logger
}

// NewTestDataService создает тестовый сервис
func NewTestDataService(dataRepo DataRepositoryInterface, logger *zap.Logger) *TestDataService {
	return &TestDataService{
		dataRepo: dataRepo,
		logger:   logger,
	}
}

// Копируем методы из оригинального DataService
func (s *TestDataService) CreateEntry(userID string, entry *models.DataEntry) (string, error) {
	s.logger.Debug("Creating data entry",
		zap.String("user_id", userID),
		zap.String("entry_name", entry.Name),
		zap.String("entry_type", string(entry.Type)),
	)

	// Устанавливаем начальную версию и временные метки
	entry.Version = 1
	entry.LastSyncTime = time.Now().UTC()
	entry.CreatedAt = time.Now().UTC()
	entry.UpdatedAt = time.Now().UTC()
	entry.UserID = userID

	err := s.dataRepo.CreateEntry(userID, entry)
	if err != nil {
		s.logger.Error("Failed to create data entry",
			zap.String("user_id", userID),
			zap.String("entry_name", entry.Name),
			zap.Error(err),
		)
		return "", err
	}

	s.logger.Info("Data entry created successfully",
		zap.String("user_id", userID),
		zap.String("entry_id", entry.ID),
		zap.String("entry_name", entry.Name),
		zap.Int64("version", entry.Version),
	)
	return entry.ID, nil
}

func (s *TestDataService) GetUserEntries(userID string) ([]*models.DataEntry, error) {
	s.logger.Debug("Getting user entries", zap.String("user_id", userID))

	entries, err := s.dataRepo.GetUserEntries(userID)
	if err != nil {
		s.logger.Error("Failed to get user entries",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return nil, err
	}

	s.logger.Debug("Retrieved user entries",
		zap.String("user_id", userID),
		zap.Int("count", len(entries)),
	)
	return entries, nil
}

func (s *TestDataService) GetEntry(userID, entryID string) (*models.DataEntry, error) {
	s.logger.Debug("Getting data entry",
		zap.String("user_id", userID),
		zap.String("entry_id", entryID),
	)

	entry, err := s.dataRepo.GetEntryByID(userID, entryID)
	if err != nil {
		s.logger.Warn("Data entry not found",
			zap.String("user_id", userID),
			zap.String("entry_id", entryID),
			zap.Error(err),
		)
		return nil, err
	}

	return entry, nil
}

func (s *TestDataService) UpdateEntryWithVersion(userID string, entry *models.DataEntry, expectedVersion int64) error {
	s.logger.Debug("Updating data entry with version check",
		zap.String("user_id", userID),
		zap.String("entry_id", entry.ID),
		zap.String("entry_name", entry.Name),
		zap.Int64("expected_version", expectedVersion),
		zap.Int64("current_version", entry.Version),
	)

	// Получаем текущую запись для проверки версии
	currentEntry, err := s.dataRepo.GetEntryByID(userID, entry.ID)
	if err != nil {
		s.logger.Error("Failed to get current entry for version check",
			zap.String("user_id", userID),
			zap.String("entry_id", entry.ID),
			zap.Error(err),
		)
		return err
	}

	// Проверяем, что версия не изменилась
	if currentEntry.Version != expectedVersion {
		s.logger.Warn("Version conflict detected",
			zap.String("user_id", userID),
			zap.String("entry_id", entry.ID),
			zap.Int64("expected_version", expectedVersion),
			zap.Int64("current_version", currentEntry.Version),
			zap.Int64("submitted_version", entry.Version),
		)
		return ErrVersionConflict
	}

	// Обновляем версию и временные метки
	entry.Version = expectedVersion + 1
	entry.LastSyncTime = time.Now().UTC()
	entry.UpdatedAt = time.Now().UTC()

	err = s.dataRepo.UpdateEntry(userID, entry)
	if err != nil {
		s.logger.Error("Failed to update data entry",
			zap.String("user_id", userID),
			zap.String("entry_id", entry.ID),
			zap.Error(err),
		)
		return err
	}

	s.logger.Info("Data entry updated successfully with version check",
		zap.String("user_id", userID),
		zap.String("entry_id", entry.ID),
		zap.Int64("old_version", expectedVersion),
		zap.Int64("new_version", entry.Version),
	)
	return nil
}

func (s *TestDataService) UpdateEntry(userID string, entry *models.DataEntry) error {
	s.logger.Debug("Updating data entry (legacy method)",
		zap.String("user_id", userID),
		zap.String("entry_id", entry.ID),
	)

	// Получаем текущую версию
	currentEntry, err := s.dataRepo.GetEntryByID(userID, entry.ID)
	if err != nil {
		s.logger.Error("Failed to get current entry for update",
			zap.String("user_id", userID),
			zap.String("entry_id", entry.ID),
			zap.Error(err),
		)
		return err
	}

	// Увеличиваем версию
	entry.Version = currentEntry.Version + 1
	entry.LastSyncTime = time.Now().UTC()
	entry.UpdatedAt = time.Now().UTC()

	err = s.dataRepo.UpdateEntry(userID, entry)
	if err != nil {
		s.logger.Error("Failed to update data entry",
			zap.String("user_id", userID),
			zap.String("entry_id", entry.ID),
			zap.Error(err),
		)
		return err
	}

	s.logger.Info("Data entry updated successfully",
		zap.String("user_id", userID),
		zap.String("entry_id", entry.ID),
		zap.Int64("old_version", currentEntry.Version),
		zap.Int64("new_version", entry.Version),
	)
	return nil
}

func (s *TestDataService) DeleteEntry(userID, entryID string) error {
	s.logger.Debug("Deleting data entry",
		zap.String("user_id", userID),
		zap.String("entry_id", entryID),
	)

	err := s.dataRepo.DeleteEntry(userID, entryID)
	if err != nil {
		s.logger.Error("Failed to delete data entry",
			zap.String("user_id", userID),
			zap.String("entry_id", entryID),
			zap.Error(err),
		)
		return err
	}

	s.logger.Info("Data entry deleted successfully",
		zap.String("user_id", userID),
		zap.String("entry_id", entryID),
	)
	return nil
}

func (s *TestDataService) GetChangesSince(userID string, since time.Time) ([]*models.DataEntry, error) {
	s.logger.Debug("Getting changes since",
		zap.String("user_id", userID),
		zap.Time("since", since),
	)
	return s.dataRepo.GetChangesSince(userID, since)
}

func (s *TestDataService) GetSyncStatus(userID string) (time.Time, int, bool, error) {
	return s.dataRepo.GetSyncStatus(userID)
}

func (s *TestDataService) ResolveConflict(userID, conflictID, resolution string, entry *models.DataEntry) error {
	s.logger.Info("Resolving conflict",
		zap.String("user_id", userID),
		zap.String("conflict_id", conflictID),
		zap.String("resolution", resolution),
	)
	return s.dataRepo.ResolveConflict(userID, conflictID, resolution, entry)
}

// MockDataRepository для тестов
type MockDataRepository struct {
	CreateEntryFunc     func(userID string, entry *models.DataEntry) error
	GetUserEntriesFunc  func(userID string) ([]*models.DataEntry, error)
	GetEntryByIDFunc    func(userID, entryID string) (*models.DataEntry, error)
	UpdateEntryFunc     func(userID string, entry *models.DataEntry) error
	DeleteEntryFunc     func(userID, entryID string) error
	GetChangesSinceFunc func(userID string, since time.Time) ([]*models.DataEntry, error)
	GetSyncStatusFunc   func(userID string) (time.Time, int, bool, error)
	ResolveConflictFunc func(userID, conflictID, resolution string, entry *models.DataEntry) error
}

func (m *MockDataRepository) CreateEntry(userID string, entry *models.DataEntry) error {
	if m.CreateEntryFunc != nil {
		return m.CreateEntryFunc(userID, entry)
	}
	return nil
}

func (m *MockDataRepository) GetUserEntries(userID string) ([]*models.DataEntry, error) {
	if m.GetUserEntriesFunc != nil {
		return m.GetUserEntriesFunc(userID)
	}
	return []*models.DataEntry{}, nil
}

func (m *MockDataRepository) GetEntryByID(userID, entryID string) (*models.DataEntry, error) {
	if m.GetEntryByIDFunc != nil {
		return m.GetEntryByIDFunc(userID, entryID)
	}
	return &models.DataEntry{}, nil
}

func (m *MockDataRepository) UpdateEntry(userID string, entry *models.DataEntry) error {
	if m.UpdateEntryFunc != nil {
		return m.UpdateEntryFunc(userID, entry)
	}
	return nil
}

func (m *MockDataRepository) DeleteEntry(userID, entryID string) error {
	if m.DeleteEntryFunc != nil {
		return m.DeleteEntryFunc(userID, entryID)
	}
	return nil
}

func (m *MockDataRepository) GetChangesSince(userID string, since time.Time) ([]*models.DataEntry, error) {
	if m.GetChangesSinceFunc != nil {
		return m.GetChangesSinceFunc(userID, since)
	}
	return []*models.DataEntry{}, nil
}

func (m *MockDataRepository) GetSyncStatus(userID string) (time.Time, int, bool, error) {
	if m.GetSyncStatusFunc != nil {
		return m.GetSyncStatusFunc(userID)
	}
	return time.Time{}, 0, false, nil
}

func (m *MockDataRepository) ResolveConflict(userID, conflictID, resolution string, entry *models.DataEntry) error {
	if m.ResolveConflictFunc != nil {
		return m.ResolveConflictFunc(userID, conflictID, resolution, entry)
	}
	return nil
}

// Тесты
func TestDataService_CreateEntry(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name          string
		setupMock     func() *MockDataRepository
		entry         *models.DataEntry
		expectedID    string
		expectedError error
	}{
		{
			name: "successful creation",
			setupMock: func() *MockDataRepository {
				return &MockDataRepository{
					CreateEntryFunc: func(userID string, entry *models.DataEntry) error {
						entry.ID = "test-id-123"
						return nil
					},
				}
			},
			entry: &models.DataEntry{
				Name:     "Test Entry",
				Type:     models.TypeLoginPassword,
				Metadata: "test metadata",
				Data:     []byte("test data"),
				Nonce:    []byte("test nonce"),
			},
			expectedID: "test-id-123",
		},
		{
			name: "repository error",
			setupMock: func() *MockDataRepository {
				return &MockDataRepository{
					CreateEntryFunc: func(userID string, entry *models.DataEntry) error {
						return errors.New("repository error")
					},
				}
			},
			entry: &models.DataEntry{
				Name:     "Test Entry",
				Type:     models.TypeText,
				Metadata: "test metadata",
			},
			expectedError: errors.New("repository error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := tt.setupMock()
			service := NewTestDataService(mockRepo, logger)

			entryID, err := service.CreateEntry("user123", tt.entry)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, entryID)
				assert.Equal(t, int64(1), tt.entry.Version)
				assert.NotZero(t, tt.entry.CreatedAt)
				assert.NotZero(t, tt.entry.UpdatedAt)
				assert.NotZero(t, tt.entry.LastSyncTime)
				assert.Equal(t, "user123", tt.entry.UserID)
			}
		})
	}
}

func TestDataService_GetUserEntries(t *testing.T) {
	logger := zap.NewNop()
	expectedEntries := []*models.DataEntry{
		{
			ID:        "entry1",
			Name:      "Test 1",
			Type:      models.TypeLoginPassword,
			UserID:    "user123",
			Version:   1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "entry2",
			Name:      "Test 2",
			Type:      models.TypeCard,
			UserID:    "user123",
			Version:   1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	mockRepo := &MockDataRepository{
		GetUserEntriesFunc: func(userID string) ([]*models.DataEntry, error) {
			return expectedEntries, nil
		},
	}

	service := NewTestDataService(mockRepo, logger)

	entries, err := service.GetUserEntries("user123")

	assert.NoError(t, err)
	assert.Equal(t, expectedEntries, entries)
}

func TestDataService_GetEntry(t *testing.T) {
	logger := zap.NewNop()
	expectedEntry := &models.DataEntry{
		ID:        "test-id",
		Name:      "Test Entry",
		Type:      models.TypeLoginPassword,
		UserID:    "user123",
		Version:   1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  "test metadata",
		Data:      []byte("encrypted data"),
		Nonce:     []byte("nonce"),
	}

	tests := []struct {
		name          string
		setupMock     func() *MockDataRepository
		expectedEntry *models.DataEntry
		expectedError error
	}{
		{
			name: "successful get",
			setupMock: func() *MockDataRepository {
				return &MockDataRepository{
					GetEntryByIDFunc: func(userID, entryID string) (*models.DataEntry, error) {
						return expectedEntry, nil
					},
				}
			},
			expectedEntry: expectedEntry,
		},
		{
			name: "entry not found",
			setupMock: func() *MockDataRepository {
				return &MockDataRepository{
					GetEntryByIDFunc: func(userID, entryID string) (*models.DataEntry, error) {
						return nil, ErrEntryNotFound
					},
				}
			},
			expectedError: ErrEntryNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := tt.setupMock()
			service := NewTestDataService(mockRepo, logger)

			entry, err := service.GetEntry("user123", "test-id")

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Nil(t, entry)
				assert.Equal(t, tt.expectedError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedEntry, entry)
			}
		})
	}
}

func TestDataService_UpdateEntryWithVersion(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name          string
		setupMock     func() *MockDataRepository
		entry         *models.DataEntry
		expectedError error
	}{
		{
			name: "successful update with version check",
			setupMock: func() *MockDataRepository {
				return &MockDataRepository{
					GetEntryByIDFunc: func(userID, entryID string) (*models.DataEntry, error) {
						return &models.DataEntry{
							ID:        "test-id",
							UserID:    "user123",
							Version:   1,
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						}, nil
					},
					UpdateEntryFunc: func(userID string, entry *models.DataEntry) error {
						return nil
					},
				}
			},
			entry: &models.DataEntry{
				ID:      "test-id",
				UserID:  "user123",
				Version: 1,
				Name:    "Updated Entry",
				Type:    models.TypeText,
			},
		},
		{
			name: "version conflict",
			setupMock: func() *MockDataRepository {
				return &MockDataRepository{
					GetEntryByIDFunc: func(userID, entryID string) (*models.DataEntry, error) {
						return &models.DataEntry{
							ID:        "test-id",
							UserID:    "user123",
							Version:   2, // Версия в базе новее
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						}, nil
					},
				}
			},
			entry: &models.DataEntry{
				ID:      "test-id",
				UserID:  "user123",
				Version: 1, // Ожидаемая версия устарела
				Name:    "Updated Entry",
				Type:    models.TypeBinary,
			},
			expectedError: ErrVersionConflict,
		},
		{
			name: "entry not found during version check",
			setupMock: func() *MockDataRepository {
				return &MockDataRepository{
					GetEntryByIDFunc: func(userID, entryID string) (*models.DataEntry, error) {
						return nil, ErrEntryNotFound
					},
				}
			},
			entry: &models.DataEntry{
				ID:   "test-id",
				Name: "Updated Entry",
				Type: models.TypeCard,
			},
			expectedError: ErrEntryNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := tt.setupMock()
			service := NewTestDataService(mockRepo, logger)

			err := service.UpdateEntryWithVersion("user123", tt.entry, 1)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, int64(2), tt.entry.Version) // Версия должна увеличиться
				assert.NotZero(t, tt.entry.UpdatedAt)
				assert.NotZero(t, tt.entry.LastSyncTime)
			}
		})
	}
}

func TestDataService_UpdateEntry(t *testing.T) {
	logger := zap.NewNop()

	mockRepo := &MockDataRepository{
		GetEntryByIDFunc: func(userID, entryID string) (*models.DataEntry, error) {
			return &models.DataEntry{
				ID:        "test-id",
				UserID:    "user123",
				Version:   1,
				Name:      "Original Entry",
				Type:      models.TypeLoginPassword,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
		UpdateEntryFunc: func(userID string, entry *models.DataEntry) error {
			return nil
		},
	}

	service := NewTestDataService(mockRepo, logger)

	entry := &models.DataEntry{
		ID:   "test-id",
		Name: "Updated Entry",
		Type: models.TypeLoginPassword,
	}

	err := service.UpdateEntry("user123", entry)

	assert.NoError(t, err)
	assert.Equal(t, int64(2), entry.Version) // Версия должна увеличиться
	assert.NotZero(t, entry.UpdatedAt)
	assert.NotZero(t, entry.LastSyncTime)
}

func TestDataService_DeleteEntry(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name          string
		setupMock     func() *MockDataRepository
		expectedError error
	}{
		{
			name: "successful deletion",
			setupMock: func() *MockDataRepository {
				return &MockDataRepository{
					DeleteEntryFunc: func(userID, entryID string) error {
						return nil
					},
				}
			},
		},
		{
			name: "repository error during deletion",
			setupMock: func() *MockDataRepository {
				return &MockDataRepository{
					DeleteEntryFunc: func(userID, entryID string) error {
						return errors.New("delete error")
					},
				}
			},
			expectedError: errors.New("delete error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := tt.setupMock()
			service := NewTestDataService(mockRepo, logger)

			err := service.DeleteEntry("user123", "test-id")

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDataService_GetChangesSince(t *testing.T) {
	logger := zap.NewNop()
	since := time.Now().Add(-time.Hour)
	expectedChanges := []*models.DataEntry{
		{
			ID:        "entry1",
			Name:      "Change 1",
			Type:      models.TypeText,
			UserID:    "user123",
			Version:   2,
			CreatedAt: time.Now().Add(-2 * time.Hour),
			UpdatedAt: time.Now().Add(-30 * time.Minute),
		},
		{
			ID:        "entry2",
			Name:      "Change 2",
			Type:      models.TypeBinary,
			UserID:    "user123",
			Version:   3,
			CreatedAt: time.Now().Add(-3 * time.Hour),
			UpdatedAt: time.Now().Add(-15 * time.Minute),
		},
	}

	mockRepo := &MockDataRepository{
		GetChangesSinceFunc: func(userID string, sinceTime time.Time) ([]*models.DataEntry, error) {
			assert.Equal(t, since, sinceTime)
			return expectedChanges, nil
		},
	}

	service := NewTestDataService(mockRepo, logger)

	changes, err := service.GetChangesSince("user123", since)

	assert.NoError(t, err)
	assert.Equal(t, expectedChanges, changes)
}

func TestDataService_GetSyncStatus(t *testing.T) {
	logger := zap.NewNop()
	expectedLastSync := time.Now().Add(-time.Hour)
	expectedPending := 5
	expectedConflicts := true

	mockRepo := &MockDataRepository{
		GetSyncStatusFunc: func(userID string) (time.Time, int, bool, error) {
			return expectedLastSync, expectedPending, expectedConflicts, nil
		},
	}

	service := NewTestDataService(mockRepo, logger)

	lastSync, pending, conflicts, err := service.GetSyncStatus("user123")

	assert.NoError(t, err)
	assert.Equal(t, expectedLastSync, lastSync)
	assert.Equal(t, expectedPending, pending)
	assert.Equal(t, expectedConflicts, conflicts)
}

func TestDataService_ResolveConflict(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name          string
		setupMock     func() *MockDataRepository
		expectedError error
	}{
		{
			name: "successful conflict resolution",
			setupMock: func() *MockDataRepository {
				return &MockDataRepository{
					ResolveConflictFunc: func(userID, conflictID, resolution string, entry *models.DataEntry) error {
						return nil
					},
				}
			},
		},
		{
			name: "repository error during conflict resolution",
			setupMock: func() *MockDataRepository {
				return &MockDataRepository{
					ResolveConflictFunc: func(userID, conflictID, resolution string, entry *models.DataEntry) error {
						return errors.New("resolution error")
					},
				}
			},
			expectedError: errors.New("resolution error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := tt.setupMock()
			service := NewTestDataService(mockRepo, logger)

			entry := &models.DataEntry{
				ID:   "test-entry",
				Name: "Test Entry",
				Type: models.TypeLoginPassword,
			}
			err := service.ResolveConflict("user123", "conflict-123", "client", entry)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDataService_ErrorConstants(t *testing.T) {
	// Проверяем, что константы ошибок определены
	assert.NotEmpty(t, ErrVersionConflict.Error())
	assert.NotEmpty(t, ErrEntryNotFound.Error())
}

func TestDataService_DataTypeConstants(t *testing.T) {
	// Проверяем, что типы данных определены
	assert.Equal(t, models.DataType("login_password"), models.TypeLoginPassword)
	assert.Equal(t, models.DataType("text"), models.TypeText)
	assert.Equal(t, models.DataType("binary"), models.TypeBinary)
	assert.Equal(t, models.DataType("card"), models.TypeCard)
}
