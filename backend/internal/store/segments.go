package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Segment 是保存的访客分段。
type Segment struct {
	ID        uuid.UUID   `json:"id"`
	SiteID    uuid.UUID   `json:"site_id"`
	Name      string      `json:"name"`
	Filters   StatsFilter `json:"filters"`
	CreatedAt time.Time   `json:"created_at"`
}

// SegmentInput 是创建或更新分段的输入。
type SegmentInput struct {
	Name    string
	Filters StatsFilter
}

const segmentColumns = `id, site_id, name, filters, created_at`

func scanSegment(row pgx.Row) (*Segment, error) {
	var (
		segment Segment
		raw     []byte
	)
	if err := row.Scan(&segment.ID, &segment.SiteID, &segment.Name, &raw, &segment.CreatedAt); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &segment.Filters); err != nil {
		return nil, err
	}
	return &segment, nil
}

// CreateSegment 创建分段。
func (s *Store) CreateSegment(ctx context.Context, siteID uuid.UUID, in SegmentInput) (*Segment, error) {
	id := uuid.New()
	raw, err := json.Marshal(in.Filters)
	if err != nil {
		return nil, err
	}

	if _, err := s.Pool.Exec(ctx, `
		insert into site_segments (id, site_id, name, filters, created_at)
		values ($1, $2, $3, $4::jsonb, now())`,
		id, siteID, in.Name, raw,
	); err != nil {
		return nil, err
	}

	return s.GetSegment(ctx, siteID, id)
}

// ListSegments 返回站点下的分段。
func (s *Store) ListSegments(ctx context.Context, siteID uuid.UUID) ([]Segment, error) {
	rows, err := s.Pool.Query(ctx, `
		select `+segmentColumns+`
		from site_segments
		where site_id = $1
		order by created_at`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]Segment, 0, 8)
	for rows.Next() {
		segment, err := scanSegment(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *segment)
	}
	return list, rows.Err()
}

// GetSegment 按站点与 ID 查询分段。
func (s *Store) GetSegment(ctx context.Context, siteID, segmentID uuid.UUID) (*Segment, error) {
	return scanSegment(s.Pool.QueryRow(ctx, `
		select `+segmentColumns+`
		from site_segments
		where id = $1 and site_id = $2`, segmentID, siteID))
}

// UpdateSegment 更新分段。
func (s *Store) UpdateSegment(ctx context.Context, siteID, segmentID uuid.UUID, in SegmentInput) (*Segment, error) {
	raw, err := json.Marshal(in.Filters)
	if err != nil {
		return nil, err
	}

	tag, err := s.Pool.Exec(ctx, `
		update site_segments set name = $2, filters = $3::jsonb
		where id = $1 and site_id = $4`,
		segmentID, in.Name, raw, siteID,
	)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return s.GetSegment(ctx, siteID, segmentID)
}

// DeleteSegment 删除分段。
func (s *Store) DeleteSegment(ctx context.Context, siteID, segmentID uuid.UUID) error {
	tag, err := s.Pool.Exec(ctx, `
		delete from site_segments where id = $1 and site_id = $2`, segmentID, siteID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
