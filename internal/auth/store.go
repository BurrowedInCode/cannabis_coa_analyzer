package auth

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID
	Username     string
	PasswordHash string
}

type Store struct {
	db *sql.DB
}

type AuthStore interface {
	RegisterUser(ctx context.Context, user *User) error
}

func (s *Store) RegisterUser(ctx context.Context, user *User) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO users (username, password_hash, created_at) VALUES($1, $2, $3)", user.Username, user.PasswordHash)
	if err != nil {
		return err
	}

	return nil
}
