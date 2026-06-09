package auth

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID           uuid.UUID
	Username     string
	PasswordHash string
}

type Store struct {
	db *pgxpool.Pool
}

type AuthStore interface {
	RegisterUser(ctx context.Context, user *User) error
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func (s *Store) RegisterUser(ctx context.Context, user *User) error {
	_, err := s.db.Exec(ctx, "INSERT INTO users (username, password_hash, created_at) VALUES($1, $2, NOW())", user.Username, user.PasswordHash)
	return err
}
