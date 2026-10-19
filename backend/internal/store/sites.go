package store

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"remoon.net/yu/backend/internal/auth"
)

// Site 是一个被统计的网站。
type Site struct {
	ID           uuid.UUID `json:"id"`
	WorkspaceID  uuid.UUID `json:"workspace_id"`
	Name         string    `json:"name"`
	Domain       string    `json:"domain"`
	Token        string    `json:"token"`
	ShareToken   string    `json:"share_token"`
	Timezone     string    `json:"timezone"`
	FilterBots   bool      `json:"filter_bots"`
	ExcludeCIDRs []string  `json:"exclude_cidrs"`
	CreatedAt    time.Time `json:"created_at"`
}

// CreateSite 创建站点并生成采集 token 与分享 token。
func (s *Store) CreateSite(ctx context.Context, workspaceID uuid.UUID, name, domain, timezone string) (*Site, error) {
	token, err := auth.NewToken()
	if err != nil {
		return nil, err
	}

	shareToken, err := auth.NewToken()
	if err != nil {
		return nil, err
	}

	site := &Site{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		Name:        strings.TrimSpace(name),
		Domain:      normalizeDomain(domain),
		Token:       token,
		ShareToken:  shareToken,
		Timezone:    timezone,
		CreatedAt:   time.Now().UTC(),
	}

	_, err = s.Pool.Exec(ctx, `
		insert into sites (id, workspace_id, name, domain, token, share_token, timezone, created_at)
		values ($1, $2, $3, $4, $5, $6, $7, $8)`,
		site.ID, site.WorkspaceID, site.Name, site.Domain, site.Token, site.ShareToken, site.Timezone, site.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return s.GetSite(ctx, workspaceID, site.ID)
}

// ListSites 返回工作区下的站点。
func (s *Store) ListSites(ctx context.Context, workspaceID uuid.UUID) ([]Site, error) {
	rows, err := s.Pool.Query(ctx, `
		select id, workspace_id, name, domain, token, share_token, timezone, filter_bots, exclude_cidrs, created_at
		from sites
		where workspace_id = $1
		order by created_at`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]Site, 0, 4)
	for rows.Next() {
		var site Site
		if err := rows.Scan(
			&site.ID, &site.WorkspaceID, &site.Name, &site.Domain,
			&site.Token, &site.ShareToken, &site.Timezone, &site.FilterBots, &site.ExcludeCIDRs, &site.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, site)
	}

	return list, rows.Err()
}

// GetSite 按工作区与站点 ID 查询站点。
func (s *Store) GetSite(ctx context.Context, workspaceID, siteID uuid.UUID) (*Site, error) {
	return s.scanSite(s.Pool.QueryRow(ctx, `
		select id, workspace_id, name, domain, token, share_token, timezone, filter_bots, exclude_cidrs, created_at
		from sites
		where id = $1 and workspace_id = $2`, siteID, workspaceID))
}

// GetSiteByID 按站点 ID 查询站点（内部使用，不校验工作区）。
func (s *Store) GetSiteByID(ctx context.Context, siteID uuid.UUID) (*Site, error) {
	return s.scanSite(s.Pool.QueryRow(ctx, `
		select id, workspace_id, name, domain, token, share_token, timezone, filter_bots, exclude_cidrs, created_at
		from sites
		where id = $1`, siteID))
}

// GetSiteByToken 按采集 token 查询站点。
func (s *Store) GetSiteByToken(ctx context.Context, token string) (*Site, error) {
	return s.scanSite(s.Pool.QueryRow(ctx, `
		select id, workspace_id, name, domain, token, share_token, timezone, filter_bots, exclude_cidrs, created_at
		from sites
		where token = $1`, token))
}

func (s *Store) scanSite(row pgx.Row) (*Site, error) {
	site := &Site{}
	err := row.Scan(
		&site.ID, &site.WorkspaceID, &site.Name, &site.Domain,
		&site.Token, &site.ShareToken, &site.Timezone, &site.FilterBots, &site.ExcludeCIDRs, &site.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return site, nil
}

// UpdateSite 更新站点基础信息与流量过滤配置。
func (s *Store) UpdateSite(ctx context.Context, workspaceID, siteID uuid.UUID, name, domain, timezone string, filterBots bool, excludeCIDRs []string) (*Site, error) {
	_, err := s.Pool.Exec(ctx, `
		update sites
		set name = $3, domain = $4, timezone = $5, filter_bots = $6, exclude_cidrs = $7
		where id = $1 and workspace_id = $2`,
		siteID, workspaceID, strings.TrimSpace(name), normalizeDomain(domain), timezone, filterBots, excludeCIDRs,
	)
	if err != nil {
		return nil, err
	}

	return s.GetSite(ctx, workspaceID, siteID)
}

// DeleteSite 删除站点，级联删除事件与会话。
func (s *Store) DeleteSite(ctx context.Context, workspaceID, siteID uuid.UUID) error {
	tag, err := s.Pool.Exec(ctx, `
		delete from sites where id = $1 and workspace_id = $2`, siteID, workspaceID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func normalizeDomain(domain string) string {
	domain = strings.ToLower(strings.TrimSpace(domain))
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimSuffix(domain, "/")
	if i := strings.Index(domain, "/"); i >= 0 {
		domain = domain[:i]
	}
	return domain
}
