package models

import "time"

type DataType string

const (
	TypeLoginPassword DataType = "login_password"
	TypeText          DataType = "text"
	TypeBinary        DataType = "binary"
	TypeCard          DataType = "card"
)

type DataEntry struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	Type      DataType  `json:"type"`
	Metadata  string    `json:"metadata"`
	Data      []byte    `json:"data"`
	Nonce     []byte    `json:"nonce"`
	Version   int64     `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type LoginPasswordData struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type CardData struct {
	Number string `json:"number"`
	Expiry string `json:"expiry"`
	Holder string `json:"holder"`
	CVV    string `json:"cvv"`
}
