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
	GetUserByUserName(ctx context.Context, username string) (*User, error)
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func (s *Store) RegisterUser(ctx context.Context, user *User) error {
	_, err := s.db.Exec(ctx, "INSERT INTO users (username, password_hash) VALUES($1, $2)", user.Username, user.PasswordHash)
	return err
}

func (s *Store) GetUserByUserName(ctx context.Context, username string) (*User, error) {
	row := s.db.QueryRow(ctx, "SELECT id, username, password_hash FROM users WHERE username = $1", username)

	user := User{}

	if err := row.Scan(&user.ID, &user.Username, &user.PasswordHash); err != nil {
		return nil, err
	}

	return &user, nil
}
