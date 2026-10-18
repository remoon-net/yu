package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"remoon.net/yu/backend/internal/auth"
)

// APIKey 是工作区的程序访问凭证（只保存哈希）。
type APIKey struct {
	ID          uuid.UUID  `json:"id"`
	WorkspaceID uuid.UUID  `json:"workspace_id"`
	Name        string     `json:"name"`
	Prefix      string     `json:"prefix"`
	CreatedAt   time.Time  `json:"created_at"`
	LastUsedAt  *time.Time `json:"last_used_at"`
}

// CreateAPIKey 生成 API Key，明文只在创建时返回一次。
func (s *Store) CreateAPIKey(ctx context.Context, workspaceID, userID uuid.UUID, name string) (*APIKey, string, error) {
	token, err := auth.NewToken()
	if err != nil {
		return nil, "", err
	}

	full := "yu_" + token
	key := &APIKey{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		Name:        name,
		Prefix:      full[:11],
		CreatedAt:   time.Now().UTC(),
	}
	user := userID

	_, err = s.Pool.Exec(ctx, `
		insert into api_keys (id, workspace_id, user_id, name, key_prefix, key_hash, created_at)
		values ($1, $2, $3, $4, $5, $6, $7)`,
		key.ID, key.WorkspaceID, user, key.Name, key.Prefix, auth.HashToken(full), key.CreatedAt,
	)
	if err != nil {
		return nil, "", err
	}

	return key, full, nil
}

// ListAPIKeys 返回工作区的 API Key。
func (s *Store) ListAPIKeys(ctx context.Context, workspaceID uuid.UUID) ([]APIKey, error) {
	rows, err := s.Pool.Query(ctx, `
		select id, workspace_id, name, key_prefix, created_at, last_used_at
		from api_keys
		where workspace_id = $1
		order by created_at desc`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]APIKey, 0, 4)
	for rows.Next() {
		var item APIKey
		if err := rows.Scan(&item.ID, &item.WorkspaceID, &item.Name, &item.Prefix, &item.CreatedAt, &item.LastUsedAt); err != nil {
			return nil, err
		}
		list = append(list, item)
	}

	return list, rows.Err()
}

// DeleteAPIKey 删除 API Key。
func (s *Store) DeleteAPIKey(ctx context.Context, workspaceID, keyID uuid.UUID) error {
	tag, err := s.Pool.Exec(ctx, `
		delete from api_keys where id = $1 and workspace_id = $2`, keyID, workspaceID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// GetUserByAPIKey 校验 API Key 并返回对应用户。
func (s *Store) GetUserByAPIKey(ctx context.Context, tokenHash string) (*User, error) {
	u := &User{}
	err := s.Pool.QueryRow(ctx, `
		with updated as (
			update api_keys set last_used_at = now()
			where key_hash = $1
			returning user_id
		)
		select u.id, u.email, u.password_hash, u.name, u.created_at
		from users u
		join updated on updated.user_id = u.id`, tokenHash,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return u, nil
}
