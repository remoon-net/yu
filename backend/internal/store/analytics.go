package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// EventInput 是一条待写入的统计事件。
type EventInput struct {
	SiteID       uuid.UUID
	Type         string
	Name         string
	VisitorID    string
	SessionID    string
	URL          string
	Path         string
	Title        string
	Referrer     string
	ReferrerHost string
	UTMSource    string
	UTMMedium    string
	UTMCampaign  string
	Channel      string
	Browser      string
	OS           string
	Device       string
	Screen       string
	Language     string
	Country      string
	Props        string
	CreatedAt    time.Time
}

// InsertEvent 写入事件，并同步维护会话。
func (s *Store) InsertEvent(ctx context.Context, in EventInput) error {
	return s.inTx(ctx, func(tx pgx.Tx) error {
		var props any
		if in.Props != "" {
			props = in.Props
		}

		_, err := tx.Exec(ctx, `
			insert into events (
				site_id, created_at, type, name, visitor_id, session_id,
				url, path, title, referrer, referrer_host,
				utm_source, utm_medium, utm_campaign, channel,
				browser, os, device, screen, language, country, props
			) values (
				$1, $2, $3, $4, $5, $6,
				$7, $8, $9, $10, $11,
				$12, $13, $14, $15,
				$16, $17, $18, $19, $20, $21, $22::jsonb
			)`,
			in.SiteID, in.CreatedAt, in.Type, nullable(in.Name), in.VisitorID, in.SessionID,
			nullable(in.URL), nullable(in.Path), nullable(in.Title), nullable(in.Referrer), nullable(in.ReferrerHost),
			nullable(in.UTMSource), nullable(in.UTMMedium), nullable(in.UTMCampaign), nullable(in.Channel),
			nullable(in.Browser), nullable(in.OS), nullable(in.Device), nullable(in.Screen), nullable(in.Language), nullable(in.Country),
			props,
		)
		if err != nil {
			return err
		}

		if in.Type == "pageview" {
			_, err = tx.Exec(ctx, `
				insert into sessions_web (
					site_id, session_id, visitor_id, started_at, last_seen_at, pageviews, entry_path, exit_path
				) values ($1, $2, $3, $4, $4, 1, $5, $5)
				on conflict (site_id, session_id) do update set
					last_seen_at = excluded.last_seen_at,
					pageviews = sessions_web.pageviews + 1,
					exit_path = excluded.exit_path`,
				in.SiteID, in.SessionID, in.VisitorID, in.CreatedAt, nullable(in.Path),
			)
			return err
		}

		_, err = tx.Exec(ctx, `
			update sessions_web set last_seen_at = $3
			where site_id = $1 and session_id = $2`,
			in.SiteID, in.SessionID, in.CreatedAt,
		)
		return err
	})
}

func nullable(value string) any {
	if value == "" {
		return nil
	}
	return value
}

// StatsOverview 是概览卡片数据。
type StatsOverview struct {
	PV          int64 `json:"pv"`
	UV          int64 `json:"uv"`
	Sessions    int64 `json:"sessions"`
	Bounces     int64 `json:"bounces"`
	DurationSum int64 `json:"duration_sum"`
}

// TimeSeriesPoint 是趋势图上的一个时间桶。
type TimeSeriesPoint struct {
	Bucket time.Time `json:"bucket"`
	PV     int64     `json:"pv"`
	UV     int64     `json:"uv"`
}

// PageStat 是页面维度统计。
type PageStat struct {
	Path     string `json:"path"`
	Title    string `json:"title"`
	Views    int64  `json:"views"`
	Visitors int64  `json:"visitors"`
}

// CountStat 是通用的“值 + 计数”统计项。
type CountStat struct {
	Value string `json:"value"`
	Count int64  `json:"count"`
}

// EventStat 是自定义事件统计。
type EventStat struct {
	Name     string    `json:"name"`
	Count    int64     `json:"count"`
	Visitors int64     `json:"visitors"`
	LastSeen time.Time `json:"last_seen"`
}

// EventPropStat 是事件属性分布。
type EventPropStat struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Count int64  `json:"count"`
}

// RealtimeStats 是实时访客数据。
type RealtimeStats struct {
	Visitors int64       `json:"visitors"`
	Sessions int64       `json:"sessions"`
	Pages    []CountStat `json:"pages"`
}

// Overview 返回区间内的核心指标。
func (s *Store) Overview(ctx context.Context, siteID uuid.UUID, from, to time.Time) (StatsOverview, error) {
	var out StatsOverview

	if err := s.Pool.QueryRow(ctx, `
		select
			count(*) filter (where type = 'pageview'),
			count(distinct visitor_id) filter (where type = 'pageview')
		from events
		where site_id = $1 and created_at >= $2 and created_at < $3`,
		siteID, from, to,
	).Scan(&out.PV, &out.UV); err != nil {
		return out, err
	}

	if err := s.Pool.QueryRow(ctx, `
		select
			count(*),
			count(*) filter (where pageviews = 1),
			coalesce(sum(extract(epoch from (last_seen_at - started_at))), 0)::bigint
		from sessions_web
		where site_id = $1 and started_at >= $2 and started_at < $3`,
		siteID, from, to,
	).Scan(&out.Sessions, &out.Bounces, &out.DurationSum); err != nil {
		return out, err
	}

	return out, nil
}

// Series 返回按时间桶聚合的 PV/UV。
func (s *Store) Series(ctx context.Context, siteID uuid.UUID, from, to time.Time, interval, timezone string) ([]TimeSeriesPoint, error) {
	rows, err := s.Pool.Query(ctx, `
		select
			(date_trunc($4, created_at at time zone $5) at time zone $5) as bucket,
			count(*) as pv,
			count(distinct visitor_id) as uv
		from events
		where site_id = $1 and created_at >= $2 and created_at < $3 and type = 'pageview'
		group by 1
		order by 1`,
		siteID, from, to, interval, timezone,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	points := make([]TimeSeriesPoint, 0, 32)
	for rows.Next() {
		var point TimeSeriesPoint
		if err := rows.Scan(&point.Bucket, &point.PV, &point.UV); err != nil {
			return nil, err
		}
		points = append(points, point)
	}

	return points, rows.Err()
}

// TopPages 返回访问量最高的页面。
func (s *Store) TopPages(ctx context.Context, siteID uuid.UUID, from, to time.Time) ([]PageStat, error) {
	rows, err := s.Pool.Query(ctx, `
		select coalesce(path, '/') as path, coalesce(max(title), '') as title,
		       count(*) as views, count(distinct visitor_id) as visitors
		from events
		where site_id = $1 and created_at >= $2 and created_at < $3 and type = 'pageview'
		group by 1
		order by views desc
		limit 100`,
		siteID, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]PageStat, 0, 32)
	for rows.Next() {
		var item PageStat
		if err := rows.Scan(&item.Path, &item.Title, &item.Views, &item.Visitors); err != nil {
			return nil, err
		}
		list = append(list, item)
	}

	return list, rows.Err()
}

// SessionPages 返回入口页或退出页排行。
func (s *Store) SessionPages(ctx context.Context, siteID uuid.UUID, from, to time.Time, entry bool) ([]CountStat, error) {
	column := "exit_path"
	if entry {
		column = "entry_path"
	}

	query := fmt.Sprintf(`
		select %s as value, count(*) as count
		from sessions_web
		where site_id = $1 and started_at >= $2 and started_at < $3 and %s is not null
		group by 1
		order by count desc
		limit 100`, column, column)

	return s.queryCountStats(ctx, query, siteID, from, to)
}

// Referrers 返回来源域名排行。
func (s *Store) Referrers(ctx context.Context, siteID uuid.UUID, from, to time.Time) ([]CountStat, error) {
	return s.queryCountStats(ctx, `
		select coalesce(nullif(referrer_host, ''), '(direct)') as value, count(*) as count
		from events
		where site_id = $1 and created_at >= $2 and created_at < $3 and type = 'pageview'
		group by 1
		order by count desc
		limit 100`, siteID, from, to)
}

// Channels 返回渠道排行。
func (s *Store) Channels(ctx context.Context, siteID uuid.UUID, from, to time.Time) ([]CountStat, error) {
	return s.queryCountStats(ctx, `
		select coalesce(nullif(channel, ''), '(unknown)') as value, count(*) as count
		from events
		where site_id = $1 and created_at >= $2 and created_at < $3 and type = 'pageview'
		group by 1
		order by count desc`, siteID, from, to)
}

// UTMStats 返回指定 UTM 字段的排行。
func (s *Store) UTMStats(ctx context.Context, siteID uuid.UUID, from, to time.Time, field string) ([]CountStat, error) {
	column := ""
	switch field {
	case "utm_source":
		column = "utm_source"
	case "utm_medium":
		column = "utm_medium"
	case "utm_campaign":
		column = "utm_campaign"
	default:
		return nil, fmt.Errorf("不支持的 UTM 字段: %s", field)
	}

	query := fmt.Sprintf(`
		select %s as value, count(*) as count
		from events
		where site_id = $1 and created_at >= $2 and created_at < $3 and type = 'pageview' and %s <> ''
		group by 1
		order by count desc
		limit 50`, column, column)

	return s.queryCountStats(ctx, query, siteID, from, to)
}

var audienceFields = map[string]bool{
	"browser":  true,
	"os":       true,
	"device":   true,
	"screen":   true,
	"language": true,
	"country":  true,
}

// Breakdown 返回访客画像某一维度的排行。
func (s *Store) Breakdown(ctx context.Context, siteID uuid.UUID, from, to time.Time, field string) ([]CountStat, error) {
	if !audienceFields[field] {
		return nil, fmt.Errorf("不支持的维度: %s", field)
	}

	query := fmt.Sprintf(`
		select %s as value, count(*) as count
		from events
		where site_id = $1 and created_at >= $2 and created_at < $3 and type = 'pageview' and %s is not null and %s <> ''
		group by 1
		order by count desc
		limit 50`, field, field, field)

	return s.queryCountStats(ctx, query, siteID, from, to)
}

func (s *Store) queryCountStats(ctx context.Context, query string, args ...any) ([]CountStat, error) {
	rows, err := s.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]CountStat, 0, 16)
	for rows.Next() {
		var item CountStat
		if err := rows.Scan(&item.Value, &item.Count); err != nil {
			return nil, err
		}
		list = append(list, item)
	}

	return list, rows.Err()
}

// CustomEvents 返回自定义事件排行。
func (s *Store) CustomEvents(ctx context.Context, siteID uuid.UUID, from, to time.Time) ([]EventStat, error) {
	rows, err := s.Pool.Query(ctx, `
		select name, count(*) as count, count(distinct visitor_id) as visitors, max(created_at) as last_seen
		from events
		where site_id = $1 and created_at >= $2 and created_at < $3
		  and type = 'custom' and name is not null and name <> ''
		group by name
		order by count desc
		limit 100`,
		siteID, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]EventStat, 0, 16)
	for rows.Next() {
		var item EventStat
		if err := rows.Scan(&item.Name, &item.Count, &item.Visitors, &item.LastSeen); err != nil {
			return nil, err
		}
		list = append(list, item)
	}

	return list, rows.Err()
}

// EventSeries 返回单个自定义事件的时间趋势。
func (s *Store) EventSeries(ctx context.Context, siteID uuid.UUID, from, to time.Time, name, interval, timezone string) ([]TimeSeriesPoint, error) {
	rows, err := s.Pool.Query(ctx, `
		select
			(date_trunc($5, created_at at time zone $6) at time zone $6) as bucket,
			count(*) as pv,
			count(distinct visitor_id) as uv
		from events
		where site_id = $1 and created_at >= $2 and created_at < $3
		  and type = 'custom' and name = $4
		group by 1
		order by 1`,
		siteID, from, to, name, interval, timezone,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	points := make([]TimeSeriesPoint, 0, 32)
	for rows.Next() {
		var point TimeSeriesPoint
		if err := rows.Scan(&point.Bucket, &point.PV, &point.UV); err != nil {
			return nil, err
		}
		points = append(points, point)
	}

	return points, rows.Err()
}

// EventProps 返回自定义事件的属性分布。
func (s *Store) EventProps(ctx context.Context, siteID uuid.UUID, from, to time.Time, name string) ([]EventPropStat, error) {
	rows, err := s.Pool.Query(ctx, `
		select p.key, p.value, count(*) as count
		from events e
		cross join lateral jsonb_each_text(coalesce(e.props, '{}'::jsonb)) as p(key, value)
		where e.site_id = $1 and e.created_at >= $2 and e.created_at < $3
		  and e.type = 'custom' and e.name = $4
		group by p.key, p.value
		order by p.key, count desc
		limit 200`,
		siteID, from, to, name,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]EventPropStat, 0, 16)
	for rows.Next() {
		var item EventPropStat
		if err := rows.Scan(&item.Key, &item.Value, &item.Count); err != nil {
			return nil, err
		}
		list = append(list, item)
	}

	return list, rows.Err()
}

// Realtime 返回最近 30 分钟的实时数据。
func (s *Store) Realtime(ctx context.Context, siteID uuid.UUID) (RealtimeStats, error) {
	var out RealtimeStats

	if err := s.Pool.QueryRow(ctx, `
		select count(distinct visitor_id)
		from events
		where site_id = $1 and created_at >= now() - interval '30 minutes'`,
		siteID,
	).Scan(&out.Visitors); err != nil {
		return out, err
	}

	if err := s.Pool.QueryRow(ctx, `
		select count(*)
		from sessions_web
		where site_id = $1 and last_seen_at >= now() - interval '5 minutes'`,
		siteID,
	).Scan(&out.Sessions); err != nil {
		return out, err
	}

	pages, err := s.queryCountStats(ctx, `
		select coalesce(path, '/') as value, count(*) as count
		from events
		where site_id = $1 and type = 'pageview' and created_at >= now() - interval '30 minutes'
		group by 1
		order by count desc
		limit 10`, siteID)
	if err != nil {
		return out, err
	}
	out.Pages = pages

	return out, nil
}
