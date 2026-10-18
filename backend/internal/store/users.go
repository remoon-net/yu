package store

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// CountUsers 返回用户总数，用于判断是否需要初始化管理员。
func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var n int
	err := s.Pool.QueryRow(ctx, `select count(*) from users`).Scan(&n)
	return n, err
}

// GetUserByEmail 按邮箱（大小写不敏感）查询用户。
func (s *Store) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	u := &User{}
	err := s.Pool.QueryRow(ctx, `
		select id, email, password_hash, name, created_at
		from users
		where lower(email) = lower($1)`, email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return u, nil
}

// CreateUser 创建用户。
func (s *Store) CreateUser(ctx context.Context, email, name, passwordHash string) (*User, error) {
	return createUser(ctx, s.Pool, email, name, passwordHash)
}

func createUser(ctx context.Context, q querier, email, name, passwordHash string) (*User, error) {
	u := &User{
		ID:           uuid.New(),
		Email:        normalizeEmail(email),
		Name:         strings.TrimSpace(name),
		PasswordHash: passwordHash,
		CreatedAt:    time.Now().UTC(),
	}

	_, err := q.Exec(ctx, `
		insert into users (id, email, password_hash, name, created_at)
		values ($1, $2, $3, $4, $5)`,
		u.ID, u.Email, u.PasswordHash, u.Name, u.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return u, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
