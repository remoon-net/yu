package store

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ErrPixelSlugTaken 表示像素标识已被占用。
var ErrPixelSlugTaken = errors.New("像素标识已被占用")

// Pixel 是一个遥测像素。
type Pixel struct {
	ID          uuid.UUID  `json:"id"`
	WorkspaceID uuid.UUID  `json:"workspace_id"`
	SiteID      uuid.UUID  `json:"site_id"`
	Slug        string     `json:"slug"`
	Name        string     `json:"name"`
	Enabled     bool       `json:"enabled"`
	ExpiresAt   *time.Time `json:"expires_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

// PixelInput 是创建或更新像素的输入。
type PixelInput struct {
	Slug      string
	Name      string
	Enabled   bool
	ExpiresAt *time.Time
}

// PixelStats 是单个像素的曝光统计。
type PixelStats struct {
	Slug     string     `json:"slug"`
	Hits     int64      `json:"hits"`
	Visitors int64      `json:"visitors"`
	LastHit  *time.Time `json:"last_hit"`
}

const pixelColumns = `
	id, workspace_id, site_id, slug, coalesce(name, ''), enabled, expires_at, created_at`

func scanPixel(row pgx.Row) (*Pixel, error) {
	pixel := &Pixel{}
	err := row.Scan(
		&pixel.ID, &pixel.WorkspaceID, &pixel.SiteID, &pixel.Slug,
		&pixel.Name, &pixel.Enabled, &pixel.ExpiresAt, &pixel.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return pixel, nil
}

// ListPixels 返回站点下的像素。
func (s *Store) ListPixels(ctx context.Context, siteID uuid.UUID) ([]Pixel, error) {
	rows, err := s.Pool.Query(ctx, `
		select `+pixelColumns+`
		from pixels
		where site_id = $1
		order by created_at desc`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]Pixel, 0, 8)
	for rows.Next() {
		pixel, err := scanPixel(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *pixel)
	}

	return list, rows.Err()
}

// GetPixel 按 id 查询站点下的像素。
func (s *Store) GetPixel(ctx context.Context, siteID, pixelID uuid.UUID) (*Pixel, error) {
	return scanPixel(s.Pool.QueryRow(ctx, `
		select `+pixelColumns+`
		from pixels
		where id = $1 and site_id = $2`, pixelID, siteID))
}

// CreatePixel 创建像素。
func (s *Store) CreatePixel(ctx context.Context, workspaceID, siteID uuid.UUID, in PixelInput) (*Pixel, error) {
	pixel := &Pixel{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		SiteID:      siteID,
		Slug:        strings.TrimSpace(in.Slug),
		Name:        strings.TrimSpace(in.Name),
		Enabled:     in.Enabled,
		ExpiresAt:   in.ExpiresAt,
		CreatedAt:   time.Now().UTC(),
	}

	_, err := s.Pool.Exec(ctx, `
		insert into pixels (id, workspace_id, site_id, slug, name, enabled, expires_at, created_at)
		values ($1, $2, $3, $4, $5, $6, $7, $8)`,
		pixel.ID, pixel.WorkspaceID, pixel.SiteID, pixel.Slug,
		nullable(pixel.Name), pixel.Enabled, pixel.ExpiresAt, pixel.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrPixelSlugTaken
		}
		return nil, err
	}

	return pixel, nil
}

// UpdatePixel 更新像素的名称、启用状态与过期时间（slug 不可改）。
func (s *Store) UpdatePixel(ctx context.Context, siteID, pixelID uuid.UUID, in PixelInput) (*Pixel, error) {
	tag, err := s.Pool.Exec(ctx, `
		update pixels set
			name = $3, enabled = $4, expires_at = $5
		where id = $1 and site_id = $2`,
		pixelID, siteID, nullable(strings.TrimSpace(in.Name)), in.Enabled, in.ExpiresAt,
	)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}

	return scanPixel(s.Pool.QueryRow(ctx, `
		select `+pixelColumns+`
		from pixels
		where id = $1 and site_id = $2`, pixelID, siteID))
}

// DeletePixel 删除像素。
func (s *Store) DeletePixel(ctx context.Context, siteID, pixelID uuid.UUID) error {
	tag, err := s.Pool.Exec(ctx, `
		delete from pixels where id = $1 and site_id = $2`, pixelID, siteID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// GetPixelBySlug 按 slug 查询像素。
func (s *Store) GetPixelBySlug(ctx context.Context, slug string) (*Pixel, error) {
	return scanPixel(s.Pool.QueryRow(ctx, `
		select `+pixelColumns+`
		from pixels
		where slug = $1`, slug))
}

// PixelStats 汇总站点下每个像素的曝光量。
func (s *Store) PixelStats(ctx context.Context, siteID uuid.UUID) ([]PixelStats, error) {
	rows, err := s.Pool.Query(ctx, `
		select coalesce(name, ''), count(*), count(distinct visitor_id), max(created_at)
		from events
		where site_id = $1 and type = 'pixel'
		group by name`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]PixelStats, 0, 8)
	for rows.Next() {
		var item PixelStats
		if err := rows.Scan(&item.Slug, &item.Hits, &item.Visitors, &item.LastHit); err != nil {
			return nil, err
		}
		list = append(list, item)
	}

	return list, rows.Err()
}

// PixelDetailStats 汇总单个像素在区间内的曝光趋势与分布。
//
// 返回结构与短链详情一致（Clicks 字段表示曝光次数），前端可复用同一个面板。
func (s *Store) PixelDetailStats(ctx context.Context, siteID uuid.UUID, slug string, from, to time.Time, interval, timezone string) (ShortLinkDetail, error) {
	return s.hitDetailStats(ctx, siteID, "pixel", slug, from, to, interval, timezone)
}
