package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Slugify 把名称转换为可用于 URL 的标识，保留中英文与数字。
func Slugify(name string) string {
	var b strings.Builder
	prevDash := false

	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		if !prevDash && b.Len() > 0 {
			b.WriteByte('-')
			prevDash = true
		}
	}

	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "workspace"
	}

	runes := []rune(out)
	if len(runes) > 40 {
		out = string(runes[:40])
	}

	return out
}

// CreateWorkspace 创建工作区，并把 ownerID 设为 owner。
func (s *Store) CreateWorkspace(ctx context.Context, name string, ownerID uuid.UUID) (*Workspace, error) {
	base := Slugify(name)

	for attempt := 0; attempt < 20; attempt++ {
		ws := &Workspace{
			ID:        uuid.New(),
			Name:      strings.TrimSpace(name),
			Slug:      base,
			CreatedAt: time.Now().UTC(),
		}
		if attempt > 0 {
			ws.Slug = fmt.Sprintf("%s-%d", base, attempt+1)
		}

		err := s.inTx(ctx, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, `
				insert into workspaces (id, name, slug, created_at)
				values ($1, $2, $3, $4)`,
				ws.ID, ws.Name, ws.Slug, ws.CreatedAt,
			); err != nil {
				return err
			}

			_, err := tx.Exec(ctx, `
				insert into workspace_members (workspace_id, user_id, role)
				values ($1, $2, 'owner')`, ws.ID, ownerID)
			return err
		})

		if isUniqueViolation(err) {
			continue
		}
		if err != nil {
			return nil, err
		}

		return ws, nil
	}

	return nil, errors.New("无法生成唯一的工作区标识")
}

// ListWorkspacesByUser 返回用户所属的全部工作区及其角色。
func (s *Store) ListWorkspacesByUser(ctx context.Context, userID uuid.UUID) ([]WorkspaceWithRole, error) {
	rows, err := s.Pool.Query(ctx, `
		select w.id, w.name, w.slug, w.created_at, m.role
		from workspaces w
		join workspace_members m on m.workspace_id = w.id
		where m.user_id = $1
		order by w.created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]WorkspaceWithRole, 0, 4)
	for rows.Next() {
		var item WorkspaceWithRole
		if err := rows.Scan(&item.ID, &item.Name, &item.Slug, &item.CreatedAt, &item.Role); err != nil {
			return nil, err
		}
		list = append(list, item)
	}

	return list, rows.Err()
}

// GetWorkspace 按 ID 查询工作区。
func (s *Store) GetWorkspace(ctx context.Context, id uuid.UUID) (*Workspace, error) {
	ws := &Workspace{}
	err := s.Pool.QueryRow(ctx, `
		select id, name, slug, created_at
		from workspaces
		where id = $1`, id,
	).Scan(&ws.ID, &ws.Name, &ws.Slug, &ws.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return ws, nil
}

// MemberRole 返回用户在工作区中的角色。
func (s *Store) MemberRole(ctx context.Context, workspaceID, userID uuid.UUID) (string, error) {
	var role string
	err := s.Pool.QueryRow(ctx, `
		select role from workspace_members
		where workspace_id = $1 and user_id = $2`, workspaceID, userID,
	).Scan(&role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	return role, nil
}

// ListMembers 返回工作区成员列表。
func (s *Store) ListMembers(ctx context.Context, workspaceID uuid.UUID) ([]Member, error) {
	rows, err := s.Pool.Query(ctx, `
		select u.id, u.email, u.name, m.role, m.created_at
		from workspace_members m
		join users u on u.id = m.user_id
		where m.workspace_id = $1
		order by m.created_at`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]Member, 0, 8)
	for rows.Next() {
		var item Member
		if err := rows.Scan(&item.UserID, &item.Email, &item.Name, &item.Role, &item.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, item)
	}

	return list, rows.Err()
}

// SetMemberRole 修改成员角色。
func (s *Store) SetMemberRole(ctx context.Context, workspaceID, userID uuid.UUID, role string) error {
	tag, err := s.Pool.Exec(ctx, `
		update workspace_members set role = $3
		where workspace_id = $1 and user_id = $2`, workspaceID, userID, role)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// RemoveMember 移除成员。
func (s *Store) RemoveMember(ctx context.Context, workspaceID, userID uuid.UUID) error {
	tag, err := s.Pool.Exec(ctx, `
		delete from workspace_members
		where workspace_id = $1 and user_id = $2`, workspaceID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CountOwners 统计工作区 owner 数量，避免移除最后一个 owner。
func (s *Store) CountOwners(ctx context.Context, workspaceID uuid.UUID) (int, error) {
	var n int
	err := s.Pool.QueryRow(ctx, `
		select count(*) from workspace_members
		where workspace_id = $1 and role = 'owner'`, workspaceID,
	).Scan(&n)
	return n, err
}
