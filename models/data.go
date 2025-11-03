package models

import "time"

type VaultItem struct {
	ID         string            `json:"id"`          // Уникальный идентификатор записи
	OwnerID    string            `json:"owner_id"`    // ID пользователя-владельца
	Type       ItemType          `json:"type"`        // Тип данных (логин/пароль, текст, бинарные данные, карта)
	Data       map[string]string `json:"data"`        // Основная текстовая информация (логин, пароль, карта и пр.)
	BinaryData []byte            `json:"binary_data"` // Произвольные бинарные данные (например, файлы)
	Meta       map[string]string `json:"meta"`        // Текстовая метаинформация (сайт, описание, одноразовые коды и т.п.)
	CreatedAt  time.Time         `json:"created_at"`  // Время создания записи
	UpdatedAt  time.Time         `json:"updated_at"`  // Время последнего обновления
}

type ItemType string

const (
	ItemTypeLogin    ItemType = "login_password"
	ItemTypeText     ItemType = "text"
	ItemTypeBinary   ItemType = "binary"
	ItemTypeBankCard ItemType = "bank_card"
)
