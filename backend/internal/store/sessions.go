package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// CreateAuthSession 保存会话（id 为 token 的 SHA-256）。
func (s *Store) CreateAuthSession(ctx context.Context, id string, userID uuid.UUID, expiresAt time.Time) error {
	_, err := s.Pool.Exec(ctx, `
		insert into sessions_auth (id, user_id, expires_at)
		values ($1, $2, $3)`, id, userID, expiresAt)
	return err
}

// GetUserBySession 通过会话 ID 查询未过期会话对应的用户。
func (s *Store) GetUserBySession(ctx context.Context, sessionID string) (*User, error) {
	u := &User{}
	err := s.Pool.QueryRow(ctx, `
		select u.id, u.email, u.password_hash, u.name, u.created_at
		from sessions_auth s
		join users u on u.id = s.user_id
		where s.id = $1 and s.expires_at > now()`, sessionID,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return u, nil
}

// DeleteAuthSession 删除指定会话。
func (s *Store) DeleteAuthSession(ctx context.Context, id string) error {
	_, err := s.Pool.Exec(ctx, `delete from sessions_auth where id = $1`, id)
	return err
}

// DeleteExpiredSessions 清理过期会话，返回删除条数。
func (s *Store) DeleteExpiredSessions(ctx context.Context) (int64, error) {
	tag, err := s.Pool.Exec(ctx, `delete from sessions_auth where expires_at <= now()`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
