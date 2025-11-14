package app

import (
	"time"

	"github.com/buharamanya/gophkeeper/internal/models"
	"github.com/buharamanya/gophkeeper/server/storage/postgres"
)

type DataService struct {
	dataRepo *postgres.DataRepository
}

func NewDataService(dataRepo *postgres.DataRepository) *DataService {
	return &DataService{
		dataRepo: dataRepo,
	}
}

func (s *DataService) CreateEntry(userID string, entry *models.DataEntry) (string, error) {
	entry.LastSyncTime = time.Now().UTC()
	err := s.dataRepo.CreateEntry(userID, entry)
	if err != nil {
		return "", err
	}
	return entry.ID, nil
}

func (s *DataService) GetUserEntries(userID string) ([]*models.DataEntry, error) {
	return s.dataRepo.GetUserEntries(userID)
}

func (s *DataService) GetEntry(userID, entryID string) (*models.DataEntry, error) {
	return s.dataRepo.GetEntryByID(userID, entryID)
}

func (s *DataService) UpdateEntry(userID string, entry *models.DataEntry) error {
	entry.LastSyncTime = time.Now().UTC()
	entry.Version++ // Увеличиваем версию при обновлении
	return s.dataRepo.UpdateEntry(userID, entry)
}

func (s *DataService) DeleteEntry(userID, entryID string) error {
	return s.dataRepo.DeleteEntry(userID, entryID)
}

// GetChangesSince возвращает изменения после указанного времени
func (s *DataService) GetChangesSince(userID string, since time.Time) ([]*models.DataEntry, error) {
	return s.dataRepo.GetChangesSince(userID, since)
}

// GetSyncStatus возвращает статус синхронизации пользователя
func (s *DataService) GetSyncStatus(userID string) (time.Time, int, bool, error) {
	return s.dataRepo.GetSyncStatus(userID)
}

// ResolveConflict разрешает конфликт синхронизации
func (s *DataService) ResolveConflict(userID, conflictID, resolution string, entry *models.DataEntry) error {
	return s.dataRepo.ResolveConflict(userID, conflictID, resolution, entry)
}
