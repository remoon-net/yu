package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// CreateInvite 保存邀请记录（token 只保存哈希）。
func (s *Store) CreateInvite(
	ctx context.Context,
	workspaceID uuid.UUID,
	email, role, tokenHash string,
	invitedBy uuid.UUID,
	expiresAt time.Time,
) (*Invite, error) {
	inv := &Invite{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		Email:       normalizeEmail(email),
		Role:        role,
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now().UTC(),
	}

	_, err := s.Pool.Exec(ctx, `
		insert into invites (id, workspace_id, email, role, token_hash, invited_by, expires_at, created_at)
		values ($1, $2, $3, $4, $5, $6, $7, $8)`,
		inv.ID, inv.WorkspaceID, inv.Email, inv.Role, tokenHash, invitedBy, inv.ExpiresAt, inv.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return inv, nil
}

// GetInviteByTokenHash 通过 token 哈希查询邀请。
func (s *Store) GetInviteByTokenHash(ctx context.Context, tokenHash string) (*Invite, error) {
	inv := &Invite{}
	err := s.Pool.QueryRow(ctx, `
		select id, workspace_id, email, role, expires_at, accepted_at, created_at
		from invites
		where token_hash = $1`, tokenHash,
	).Scan(&inv.ID, &inv.WorkspaceID, &inv.Email, &inv.Role, &inv.ExpiresAt, &inv.AcceptedAt, &inv.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return inv, nil
}

// ListPendingInvites 返回工作区中尚未接受的邀请。
func (s *Store) ListPendingInvites(ctx context.Context, workspaceID uuid.UUID) ([]Invite, error) {
	rows, err := s.Pool.Query(ctx, `
		select id, workspace_id, email, role, expires_at, accepted_at, created_at
		from invites
		where workspace_id = $1 and accepted_at is null and expires_at > now()
		order by created_at desc`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]Invite, 0, 4)
	for rows.Next() {
		var inv Invite
		if err := rows.Scan(&inv.ID, &inv.WorkspaceID, &inv.Email, &inv.Role, &inv.ExpiresAt, &inv.AcceptedAt, &inv.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, inv)
	}

	return list, rows.Err()
}

// DeleteInvite 撤销尚未接受的邀请，使链接立即失效。
func (s *Store) DeleteInvite(ctx context.Context, workspaceID, inviteID uuid.UUID) error {
	tag, err := s.Pool.Exec(ctx, `
		delete from invites
		where id = $1 and workspace_id = $2 and accepted_at is null`, inviteID, workspaceID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// AcceptInvite 接受邀请：必要时创建用户，加入工作区，并标记邀请已使用。
func (s *Store) AcceptInvite(ctx context.Context, inviteID uuid.UUID, email, name, passwordHash string) (*User, error) {
	var user *User

	err := s.inTx(ctx, func(tx pgx.Tx) error {
		existing := &User{}
		err := tx.QueryRow(ctx, `
			select id, email, password_hash, name, created_at
			from users
			where lower(email) = lower($1)`, email,
		).Scan(&existing.ID, &existing.Email, &existing.PasswordHash, &existing.Name, &existing.CreatedAt)

		switch {
		case err == nil:
			user = existing
		case errors.Is(err, pgx.ErrNoRows):
			created, err := createUser(ctx, tx, email, name, passwordHash)
			if err != nil {
				return err
			}
			user = created
		default:
			return err
		}

		if _, err := tx.Exec(ctx, `
			insert into workspace_members (workspace_id, user_id, role)
			select workspace_id, $2, role from invites where id = $1
			on conflict (workspace_id, user_id) do nothing`, inviteID, user.ID,
		); err != nil {
			return err
		}

		tag, err := tx.Exec(ctx, `
			update invites set accepted_at = now()
			where id = $1 and accepted_at is null`, inviteID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return user, nil
}
