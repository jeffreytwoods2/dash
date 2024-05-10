package models

import (
	"database/sql"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           int
	CreatedAt    time.Time
	Gamertag     string
	PasswordHash []byte
	Platform     string
}

type UserModel struct {
	DB *sql.DB
}

func (m *UserModel) Insert(gamertag, password string, platform string) error {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO users (gamertag, password_hash, platform)
		VALUES ($1, $2, $3)
	`

	_, err = m.DB.Exec(query, gamertag, passwordHash, platform)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_gamertag_key"`:
			return ErrDuplicateGamertag
		default:
			return err
		}

	}

	return nil
}

func (m *UserModel) Authenticate(gamertag, password string) (int, error) {
	var (
		id           int
		passwordHash []byte
	)

	query := `
		SELECT id, password_hash FROM users WHERE gamertag = $1
	`

	err := m.DB.QueryRow(query, gamertag).Scan(&id, &passwordHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrInvalidCredentials
		} else {
			return 0, err
		}
	}

	err = bcrypt.CompareHashAndPassword(passwordHash, []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return 0, ErrInvalidCredentials
		} else {
			return 0, err
		}
	}

	return id, nil
}

func (m *UserModel) Exists(id int) (bool, error) {
	return false, nil
}
