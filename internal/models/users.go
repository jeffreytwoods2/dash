package models

import (
	"database/sql"
	"time"
)

type User struct {
	ID           int
	CreatedAt    time.Time
	Gamertag     string
	PasswordHash []byte
	Java         bool
}

type UserModel struct {
	DB *sql.DB
}

func (m *UserModel) Insert(gamertag, password string, java bool) error {
	return nil
}

func (m *UserModel) Authenticate(gamertag, password string) (int, error) {
	return 0, nil
}

func (m *UserModel) Exists(id int) (bool, error) {
	return false, nil
}
