package store

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ErrShortLinkSlugTaken 表示短链标识已被占用。
var ErrShortLinkSlugTaken = errors.New("短链标识已被占用")

// ShortLink 是一条短链。
type ShortLink struct {
	ID          uuid.UUID  `json:"id"`
	WorkspaceID uuid.UUID  `json:"workspace_id"`
	SiteID      uuid.UUID  `json:"site_id"`
	Slug        string     `json:"slug"`
	TargetURL   string     `json:"target_url"`
	Title       string     `json:"title"`
	Enabled     bool       `json:"enabled"`
	ExpiresAt   *time.Time `json:"expires_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

// ShortLinkInput 是创建或更新短链的输入。
type ShortLinkInput struct {
	Slug      string
	TargetURL string
	Title     string
	Enabled   bool
	ExpiresAt *time.Time
}

// ShortLinkStats 是单个短链的点击统计。
type ShortLinkStats struct {
	Slug      string     `json:"slug"`
	Clicks    int64      `json:"clicks"`
	Visitors  int64      `json:"visitors"`
	LastClick *time.Time `json:"last_click"`
}

// ShortLinkClick 是一条短链点击明细。
type ShortLinkClick struct {
	CreatedAt    time.Time `json:"created_at"`
	ReferrerHost string    `json:"referrer_host"`
	Channel      string    `json:"channel"`
	UTMSource    string    `json:"utm_source"`
	UTMCampaign  string    `json:"utm_campaign"`
	Browser      string    `json:"browser"`
	OS           string    `json:"os"`
	Device       string    `json:"device"`
}

// ShortLinkDetail 是单个短链在区间内的点击详情。
type ShortLinkDetail struct {
	Clicks       int64             `json:"clicks"`
	Visitors     int64             `json:"visitors"`
	LastClick    *time.Time        `json:"last_click"`
	Series       []TimeSeriesPoint `json:"series"`
	Referrers    []CountStat       `json:"referrers"`
	Channels     []CountStat       `json:"channels"`
	UTMSources   []CountStat       `json:"utm_source"`
	UTMCampaigns []CountStat       `json:"utm_campaign"`
	Browsers     []CountStat       `json:"browser"`
	OS           []CountStat       `json:"os"`
	Devices      []CountStat       `json:"device"`
	Recent       []ShortLinkClick  `json:"recent"`
}

const shortLinkColumns = `
	id, workspace_id, site_id, slug, target_url, coalesce(title, ''), enabled, expires_at, created_at`

func scanShortLink(row pgx.Row) (*ShortLink, error) {
	link := &ShortLink{}
	err := row.Scan(
		&link.ID, &link.WorkspaceID, &link.SiteID, &link.Slug, &link.TargetURL,
		&link.Title, &link.Enabled, &link.ExpiresAt, &link.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return link, nil
}

// ListShortLinks 返回站点下的短链。
func (s *Store) ListShortLinks(ctx context.Context, siteID uuid.UUID) ([]ShortLink, error) {
	rows, err := s.Pool.Query(ctx, `
		select `+shortLinkColumns+`
		from short_links
		where site_id = $1
		order by created_at desc`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]ShortLink, 0, 8)
	for rows.Next() {
		link, err := scanShortLink(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *link)
	}

	return list, rows.Err()
}

// CreateShortLink 创建短链。
func (s *Store) CreateShortLink(ctx context.Context, workspaceID, siteID uuid.UUID, in ShortLinkInput) (*ShortLink, error) {
	link := &ShortLink{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		SiteID:      siteID,
		Slug:        strings.TrimSpace(in.Slug),
		TargetURL:   strings.TrimSpace(in.TargetURL),
		Title:       strings.TrimSpace(in.Title),
		Enabled:     in.Enabled,
		ExpiresAt:   in.ExpiresAt,
		CreatedAt:   time.Now().UTC(),
	}

	_, err := s.Pool.Exec(ctx, `
		insert into short_links (id, workspace_id, site_id, slug, target_url, title, enabled, expires_at, created_at)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		link.ID, link.WorkspaceID, link.SiteID, link.Slug, link.TargetURL,
		nullable(link.Title), link.Enabled, link.ExpiresAt, link.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrShortLinkSlugTaken
		}
		return nil, err
	}

	return link, nil
}

// UpdateShortLink 更新短链的目标、备注、启用状态与过期时间（slug 不可改）。
func (s *Store) UpdateShortLink(ctx context.Context, siteID, linkID uuid.UUID, in ShortLinkInput) (*ShortLink, error) {
	tag, err := s.Pool.Exec(ctx, `
		update short_links set
			target_url = $3, title = $4, enabled = $5, expires_at = $6
		where id = $1 and site_id = $2`,
		linkID, siteID, strings.TrimSpace(in.TargetURL), nullable(strings.TrimSpace(in.Title)), in.Enabled, in.ExpiresAt,
	)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}

	return scanShortLink(s.Pool.QueryRow(ctx, `
		select `+shortLinkColumns+`
		from short_links
		where id = $1 and site_id = $2`, linkID, siteID))
}

// DeleteShortLink 删除短链。
func (s *Store) DeleteShortLink(ctx context.Context, siteID, linkID uuid.UUID) error {
	tag, err := s.Pool.Exec(ctx, `
		delete from short_links where id = $1 and site_id = $2`, linkID, siteID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// GetShortLinkBySlug 按 slug 查询短链及其站点域名。
func (s *Store) GetShortLinkBySlug(ctx context.Context, slug string) (*ShortLink, string, error) {
	link := &ShortLink{}
	var domain string

	err := s.Pool.QueryRow(ctx, `
		select l.id, l.workspace_id, l.site_id, l.slug, l.target_url,
		       coalesce(l.title, ''), l.enabled, l.expires_at, l.created_at,
		       coalesce(s.domain, '')
		from short_links l
		join sites s on s.id = l.site_id
		where l.slug = $1`, slug,
	).Scan(
		&link.ID, &link.WorkspaceID, &link.SiteID, &link.Slug, &link.TargetURL,
		&link.Title, &link.Enabled, &link.ExpiresAt, &link.CreatedAt, &domain,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", ErrNotFound
		}
		return nil, "", err
	}

	return link, domain, nil
}

// ShortLinkStats 汇总站点下每个短链的点击量。
func (s *Store) ShortLinkStats(ctx context.Context, siteID uuid.UUID) ([]ShortLinkStats, error) {
	rows, err := s.Pool.Query(ctx, `
		select coalesce(name, ''), count(*), count(distinct visitor_id), max(created_at)
		from events
		where site_id = $1 and type = 'link_click'
		group by name`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]ShortLinkStats, 0, 8)
	for rows.Next() {
		var item ShortLinkStats
		if err := rows.Scan(&item.Slug, &item.Clicks, &item.Visitors, &item.LastClick); err != nil {
			return nil, err
		}
		list = append(list, item)
	}

	return list, rows.Err()
}

// GetShortLink 按 id 查询站点下的短链。
func (s *Store) GetShortLink(ctx context.Context, siteID, linkID uuid.UUID) (*ShortLink, error) {
	return scanShortLink(s.Pool.QueryRow(ctx, `
		select `+shortLinkColumns+`
		from short_links
		where id = $1 and site_id = $2`, linkID, siteID))
}

// hitFilter 是短链点击 / 像素曝光事件的公共过滤条件（参数 $1..$5）。
const hitFilter = `
		from events
		where site_id = $1 and type = $2 and name = $3
		  and created_at >= $4 and created_at < $5`

// ShortLinkDetailStats 汇总单条短链在区间内的点击趋势与分布。
func (s *Store) ShortLinkDetailStats(ctx context.Context, siteID uuid.UUID, slug string, from, to time.Time, interval, timezone string) (ShortLinkDetail, error) {
	return s.hitDetailStats(ctx, siteID, "link_click", slug, from, to, interval, timezone)
}

// hitDetailStats 汇总某类命中事件（link_click / pixel）在区间内的趋势与分布。
func (s *Store) hitDetailStats(ctx context.Context, siteID uuid.UUID, eventType, name string, from, to time.Time, interval, timezone string) (ShortLinkDetail, error) {
	out := ShortLinkDetail{
		Series:       make([]TimeSeriesPoint, 0, 32),
		Referrers:    make([]CountStat, 0, 8),
		Channels:     make([]CountStat, 0, 8),
		UTMSources:   make([]CountStat, 0, 8),
		UTMCampaigns: make([]CountStat, 0, 8),
		Browsers:     make([]CountStat, 0, 8),
		OS:           make([]CountStat, 0, 8),
		Devices:      make([]CountStat, 0, 8),
		Recent:       make([]ShortLinkClick, 0, 20),
	}

	if err := s.Pool.QueryRow(ctx, `
		select count(*), count(distinct visitor_id), max(created_at)
		`+hitFilter,
		siteID, eventType, name, from, to,
	).Scan(&out.Clicks, &out.Visitors, &out.LastClick); err != nil {
		return out, err
	}

	rows, err := s.Pool.Query(ctx, `
		select
			(date_trunc($6, created_at at time zone $7) at time zone $7) as bucket,
			count(*) as pv,
			count(distinct visitor_id) as uv
		`+hitFilter+`
		group by 1
		order by 1`, siteID, eventType, name, from, to, interval, timezone)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var point TimeSeriesPoint
		if err := rows.Scan(&point.Bucket, &point.PV, &point.UV); err != nil {
			rows.Close()
			return out, err
		}
		out.Series = append(out.Series, point)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return out, err
	}

	dimensions := []struct {
		dest  *[]CountStat
		query string
		extra string
	}{
		{&out.Referrers, `select coalesce(nullif(referrer_host, ''), '(direct)') as value, count(*) as count`, ""},
		{&out.Channels, `select coalesce(nullif(channel, ''), '(unknown)') as value, count(*) as count`, ""},
		{&out.UTMSources, `select utm_source as value, count(*) as count`, `
		and coalesce(utm_source, '') <> ''`},
		{&out.UTMCampaigns, `select utm_campaign as value, count(*) as count`, `
		and coalesce(utm_campaign, '') <> ''`},
		{&out.Browsers, `select browser as value, count(*) as count`, `
		and coalesce(browser, '') <> ''`},
		{&out.OS, `select os as value, count(*) as count`, `
		and coalesce(os, '') <> ''`},
		{&out.Devices, `select device as value, count(*) as count`, `
		and coalesce(device, '') <> ''`},
	}

	for _, dimension := range dimensions {
		list, err := s.queryCountStats(ctx, dimension.query+hitFilter+dimension.extra+`
		group by 1
		order by count desc
		limit 20`, siteID, eventType, name, from, to)
		if err != nil {
			return out, err
		}
		*dimension.dest = list
	}

	recentRows, err := s.Pool.Query(ctx, `
		select created_at, coalesce(referrer_host, ''), coalesce(channel, ''),
		       coalesce(utm_source, ''), coalesce(utm_campaign, ''),
		       coalesce(browser, ''), coalesce(os, ''), coalesce(device, '')
		`+hitFilter+`
		order by created_at desc
		limit 50`, siteID, eventType, name, from, to)
	if err != nil {
		return out, err
	}
	defer recentRows.Close()

	for recentRows.Next() {
		var item ShortLinkClick
		if err := recentRows.Scan(
			&item.CreatedAt, &item.ReferrerHost, &item.Channel,
			&item.UTMSource, &item.UTMCampaign,
			&item.Browser, &item.OS, &item.Device,
		); err != nil {
			return out, err
		}
		out.Recent = append(out.Recent, item)
	}

	return out, recentRows.Err()
}
