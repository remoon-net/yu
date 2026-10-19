package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// StatusPage 是公开状态页。
type StatusPage struct {
	ID          uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	HasPassword bool      `json:"has_password"`
	Theme       string    `json:"theme"`
	CreatedAt   time.Time `json:"created_at"`
}

// StatusPageItem 是状态页上的一个监控组件。
type StatusPageItem struct {
	MonitorID   uuid.UUID `json:"monitor_id"`
	DisplayName string    `json:"display_name"`
	GroupName   string    `json:"group_name"`
	SortOrder   int       `json:"sort_order"`
}

// StatusPageItemView 是公开状态页上的组件视图。
type StatusPageItemView struct {
	MonitorID     uuid.UUID  `json:"monitor_id"`
	Name          string     `json:"name"`
	GroupName     string     `json:"group_name"`
	Type          string     `json:"type"`
	Status        string     `json:"status"`
	LastCheckedAt *time.Time `json:"last_checked_at"`
	LatencyMS     *int       `json:"latency_ms"`
	Error         string     `json:"error"`
	Uptime24h     float64    `json:"uptime_24h"`
}

// CreateStatusPage 创建状态页。
func (s *Store) CreateStatusPage(
	ctx context.Context,
	workspaceID uuid.UUID,
	slug, name, description string,
	passwordHash *string,
	theme string,
) (*StatusPage, error) {
	id := uuid.New()

	_, err := s.Pool.Exec(ctx, `
		insert into status_pages (id, workspace_id, slug, name, description, password_hash, theme)
		values ($1, $2, $3, $4, $5, $6, $7)`,
		id, workspaceID, slug, name, nullable(description), passwordHash, theme,
	)
	if err != nil {
		return nil, err
	}

	return s.GetStatusPage(ctx, workspaceID, id)
}

// ListStatusPages 返回工作区的状态页。
func (s *Store) ListStatusPages(ctx context.Context, workspaceID uuid.UUID) ([]StatusPage, error) {
	rows, err := s.Pool.Query(ctx, `
		select id, workspace_id, slug, name, coalesce(description, ''),
		       password_hash is not null, theme, created_at
		from status_pages
		where workspace_id = $1
		order by created_at`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]StatusPage, 0, 4)
	for rows.Next() {
		var item StatusPage
		if err := rows.Scan(
			&item.ID, &item.WorkspaceID, &item.Slug, &item.Name,
			&item.Description, &item.HasPassword, &item.Theme, &item.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, item)
	}

	return list, rows.Err()
}

// GetStatusPage 按 ID 查询状态页。
func (s *Store) GetStatusPage(ctx context.Context, workspaceID, pageID uuid.UUID) (*StatusPage, error) {
	item := &StatusPage{}
	err := s.Pool.QueryRow(ctx, `
		select id, workspace_id, slug, name, coalesce(description, ''),
		       password_hash is not null, theme, created_at
		from status_pages
		where id = $1 and workspace_id = $2`, pageID, workspaceID,
	).Scan(
		&item.ID, &item.WorkspaceID, &item.Slug, &item.Name,
		&item.Description, &item.HasPassword, &item.Theme, &item.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return item, nil
}

// GetStatusPageBySlug 公开访问时按 slug 查询。
func (s *Store) GetStatusPageBySlug(ctx context.Context, slug string) (*StatusPage, string, error) {
	item := &StatusPage{}
	var passwordHash *string

	err := s.Pool.QueryRow(ctx, `
		select id, workspace_id, slug, name, coalesce(description, ''),
		       password_hash is not null, theme, created_at, password_hash
		from status_pages
		where slug = $1`, slug,
	).Scan(
		&item.ID, &item.WorkspaceID, &item.Slug, &item.Name,
		&item.Description, &item.HasPassword, &item.Theme, &item.CreatedAt, &passwordHash,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", ErrNotFound
		}
		return nil, "", err
	}

	hash := ""
	if passwordHash != nil {
		hash = *passwordHash
	}

	return item, hash, nil
}

// UpdateStatusPage 更新状态页，clearPassword 为 true 时清除密码。
func (s *Store) UpdateStatusPage(
	ctx context.Context,
	workspaceID, pageID uuid.UUID,
	name, description, theme string,
	clearPassword bool,
	passwordHash *string,
) (*StatusPage, error) {
	tag, err := s.Pool.Exec(ctx, `
		update status_pages set
			name = $3, description = $4, theme = $5,
			password_hash = case when $6 then null else coalesce($7, password_hash) end
		where id = $1 and workspace_id = $2`,
		pageID, workspaceID, name, nullable(description), theme, clearPassword, passwordHash,
	)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}

	return s.GetStatusPage(ctx, workspaceID, pageID)
}

// DeleteStatusPage 删除状态页。
func (s *Store) DeleteStatusPage(ctx context.Context, workspaceID, pageID uuid.UUID) error {
	tag, err := s.Pool.Exec(ctx, `
		delete from status_pages where id = $1 and workspace_id = $2`, pageID, workspaceID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ReplaceStatusPageItems 覆盖状态页组件。
func (s *Store) ReplaceStatusPageItems(ctx context.Context, workspaceID, pageID uuid.UUID, items []StatusPageItem) error {
	if _, err := s.GetStatusPage(ctx, workspaceID, pageID); err != nil {
		return err
	}

	return s.inTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `delete from status_page_items where status_page_id = $1`, pageID); err != nil {
			return err
		}

		for index, item := range items {
			_, err := tx.Exec(ctx, `
				insert into status_page_items (status_page_id, monitor_id, display_name, group_name, sort_order)
				values ($1, $2, $3, $4, $5)`,
				pageID, item.MonitorID, nullable(item.DisplayName), nullable(item.GroupName), index,
			)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// ListStatusPageItems 返回状态页组件（管理端）。
func (s *Store) ListStatusPageItems(ctx context.Context, pageID uuid.UUID) ([]StatusPageItem, error) {
	rows, err := s.Pool.Query(ctx, `
		select monitor_id, coalesce(display_name, ''), coalesce(group_name, ''), sort_order
		from status_page_items
		where status_page_id = $1
		order by sort_order`, pageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]StatusPageItem, 0, 8)
	for rows.Next() {
		var item StatusPageItem
		if err := rows.Scan(&item.MonitorID, &item.DisplayName, &item.GroupName, &item.SortOrder); err != nil {
			return nil, err
		}
		list = append(list, item)
	}

	return list, rows.Err()
}

// ListStatusPageItemsView 返回公开状态页组件与实时状态。
func (s *Store) ListStatusPageItemsView(ctx context.Context, pageID uuid.UUID) ([]StatusPageItemView, error) {
	rows, err := s.Pool.Query(ctx, `
		select i.monitor_id, coalesce(nullif(i.display_name, ''), m.name),
		       coalesce(nullif(i.group_name, ''), '默认'),
		       m.type, coalesce(m.last_status, ''), m.last_checked_at,
		       m.last_latency_ms, coalesce(m.last_error, '')
		from status_page_items i
		join monitors m on m.id = i.monitor_id
		where i.status_page_id = $1
		order by i.sort_order`, pageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]StatusPageItemView, 0, 8)
	for rows.Next() {
		var item StatusPageItemView
		if err := rows.Scan(
			&item.MonitorID, &item.Name, &item.GroupName, &item.Type,
			&item.Status, &item.LastCheckedAt, &item.LatencyMS, &item.Error,
		); err != nil {
			return nil, err
		}
		list = append(list, item)
	}

	return list, rows.Err()
}

// MonitorUptime24h 返回一组监控的 24 小时可用率。
func (s *Store) MonitorUptime24h(ctx context.Context, monitorIDs []uuid.UUID) (map[uuid.UUID]float64, error) {
	result := make(map[uuid.UUID]float64, len(monitorIDs))
	if len(monitorIDs) == 0 {
		return result, nil
	}

	rows, err := s.Pool.Query(ctx, `
		select monitor_id, count(*), count(*) filter (where status = 'up')
		from monitor_checks
		where monitor_id = any ($1) and checked_at >= now() - interval '24 hours'
		group by monitor_id`, monitorIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var monitorID uuid.UUID
		var total, up int
		if err := rows.Scan(&monitorID, &total, &up); err != nil {
			return nil, err
		}
		if total > 0 {
			result[monitorID] = float64(up) / float64(total) * 100
		}
	}

	return result, rows.Err()
}

// GetSiteByShareToken 按分享 token 查询站点。
func (s *Store) GetSiteByShareToken(ctx context.Context, token string) (*Site, error) {
	return s.scanSite(s.Pool.QueryRow(ctx, `
		select id, workspace_id, name, domain, token, share_token, timezone, created_at
		from sites where share_token = $1`, token))
}

// SetSiteSharePassword 设置或清除站点分享密码。
func (s *Store) SetSiteSharePassword(ctx context.Context, workspaceID, siteID uuid.UUID, passwordHash *string) error {
	tag, err := s.Pool.Exec(ctx, `
		update sites set share_password_hash = $3
		where id = $1 and workspace_id = $2`, siteID, workspaceID, passwordHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// GetSiteSharePassword 返回站点分享密码哈希。
func (s *Store) GetSiteSharePassword(ctx context.Context, siteID uuid.UUID) (string, error) {
	var hash *string
	err := s.Pool.QueryRow(ctx, `
		select share_password_hash from sites where id = $1`, siteID).Scan(&hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	if hash == nil {
		return "", nil
	}
	return *hash, nil
}
