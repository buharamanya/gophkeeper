package app

import (
	"errors"
	"time"

	"github.com/buharamanya/gophkeeper/internal/models"
	"github.com/buharamanya/gophkeeper/server/storage/postgres"
	"go.uber.org/zap"
)

// Ошибки для оптимистической блокировки
var (
	ErrVersionConflict = errors.New("version conflict: data was modified by another operation")
	ErrEntryNotFound   = errors.New("data entry not found")
)

type DataService struct {
	dataRepo *postgres.DataRepository
	logger   *zap.Logger
}

func NewDataService(dataRepo *postgres.DataRepository, logger *zap.Logger) *DataService {
	return &DataService{
		dataRepo: dataRepo,
		logger:   logger,
	}
}

func (s *DataService) CreateEntry(userID string, entry *models.DataEntry) (string, error) {
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

func (s *DataService) GetUserEntries(userID string) ([]*models.DataEntry, error) {
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

func (s *DataService) GetEntry(userID, entryID string) (*models.DataEntry, error) {
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

// UpdateEntryWithVersion обновляет запись с проверкой версии (оптимистическая блокировка)
func (s *DataService) UpdateEntryWithVersion(userID string, entry *models.DataEntry, expectedVersion int64) error {
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

// UpdateEntry обновляет запись (обратная совместимость)
func (s *DataService) UpdateEntry(userID string, entry *models.DataEntry) error {
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

func (s *DataService) DeleteEntry(userID, entryID string) error {
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

// Методы для синхронизации
func (s *DataService) GetChangesSince(userID string, since time.Time) ([]*models.DataEntry, error) {
	s.logger.Debug("Getting changes since",
		zap.String("user_id", userID),
		zap.Time("since", since),
	)
	return s.dataRepo.GetChangesSince(userID, since)
}

func (s *DataService) GetSyncStatus(userID string) (time.Time, int, bool, error) {
	return s.dataRepo.GetSyncStatus(userID)
}

func (s *DataService) ResolveConflict(userID, conflictID, resolution string, entry *models.DataEntry) error {
	s.logger.Info("Resolving conflict",
		zap.String("user_id", userID),
		zap.String("conflict_id", conflictID),
		zap.String("resolution", resolution),
	)
	return s.dataRepo.ResolveConflict(userID, conflictID, resolution, entry)
}
