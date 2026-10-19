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
	SiteID         uuid.UUID
	Type           string
	Name           string
	VisitorID      string
	SessionID      string
	URL            string
	Path           string
	Title          string
	Referrer       string
	ReferrerHost   string
	UTMSource      string
	UTMMedium      string
	UTMCampaign    string
	UTMTerm        string
	UTMContent     string
	Channel        string
	Browser        string
	BrowserVersion string
	OS             string
	OSVersion      string
	Device         string
	DeviceBrand    string
	Screen         string
	Language       string
	Country        string
	Region         string
	City           string
	Props          string
	CreatedAt      time.Time
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
				utm_source, utm_medium, utm_campaign, utm_term, utm_content, channel,
				browser, browser_version, os, os_version, device, device_brand, screen, language, country, region, city, props
			) values (
				$1, $2, $3, $4, $5, $6,
				$7, $8, $9, $10, $11,
				$12, $13, $14, $15, $16, $17,
				$18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29::jsonb
			)`,
			in.SiteID, in.CreatedAt, in.Type, nullable(in.Name), in.VisitorID, in.SessionID,
			nullable(in.URL), nullable(in.Path), nullable(in.Title), nullable(in.Referrer), nullable(in.ReferrerHost),
			nullable(in.UTMSource), nullable(in.UTMMedium), nullable(in.UTMCampaign), nullable(in.UTMTerm), nullable(in.UTMContent), nullable(in.Channel),
			nullable(in.Browser), nullable(in.BrowserVersion), nullable(in.OS), nullable(in.OSVersion), nullable(in.Device), nullable(in.DeviceBrand), nullable(in.Screen), nullable(in.Language), nullable(in.Country), nullable(in.Region), nullable(in.City),
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
	PV                int64 `json:"pv"`
	UV                int64 `json:"uv"`
	Sessions          int64 `json:"sessions"`
	Bounces           int64 `json:"bounces"`
	DurationSum       int64 `json:"duration_sum"`
	NewVisitors       int64 `json:"new_visitors"`
	ReturningVisitors int64 `json:"returning_visitors"`
	DAU               int64 `json:"dau"`
	WAU               int64 `json:"wau"`
	MAU               int64 `json:"mau"`
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

// SiteStatsSummary 是工作区内单个站点在区间内的统计。
type SiteStatsSummary struct {
	SiteID   uuid.UUID `json:"site_id"`
	Name     string    `json:"name"`
	Domain   string    `json:"domain"`
	PV       int64     `json:"pv"`
	UV       int64     `json:"uv"`
	Sessions int64     `json:"sessions"`
}

// Overview 返回区间内的核心指标。
func (s *Store) Overview(ctx context.Context, siteID uuid.UUID, from, to time.Time, filter StatsFilter) (StatsOverview, error) {
	var out StatsOverview

	scope, scopeArgs := filter.scopeSQL(4)
	args := append([]any{siteID, from, to}, scopeArgs...)

	if err := s.Pool.QueryRow(ctx, `
		select
			count(*) filter (where type = 'pageview'),
			count(distinct visitor_id) filter (where type = 'pageview')
		from events
		where site_id = $1 and created_at >= $2 and created_at < $3`+scope,
		args...,
	).Scan(&out.PV, &out.UV); err != nil {
		return out, err
	}

	if err := s.Pool.QueryRow(ctx, `
		select
			count(*),
			count(*) filter (where pageviews = 1),
			coalesce(sum(extract(epoch from (last_seen_at - started_at))), 0)::bigint
		from sessions_web
		where site_id = $1 and started_at >= $2 and started_at < $3`+scope,
		args...,
	).Scan(&out.Sessions, &out.Bounces, &out.DurationSum); err != nil {
		return out, err
	}

	if err := s.decorateSiteAudience(ctx, siteID, from, to, filter, &out); err != nil {
		return out, err
	}

	return out, nil
}

func (s *Store) decorateSiteAudience(ctx context.Context, siteID uuid.UUID, from, to time.Time, filter StatsFilter, out *StatsOverview) error {
	scope, scopeArgs := filter.scopeSQL(4)
	args := append([]any{siteID, from, to}, scopeArgs...)

	if err := s.Pool.QueryRow(ctx, `
		with active as (
			select distinct visitor_id
			from sessions_web
			where site_id = $1 and started_at >= $2 and started_at < $3`+scope+`
		), first_seen as (
			select visitor_id, min(started_at) as first_at
			from sessions_web
			where site_id = $1
			group by visitor_id
		)
		select
			count(*) filter (where f.first_at >= $2),
			count(*) filter (where f.first_at < $2)
		from active a
		join first_seen f on f.visitor_id = a.visitor_id`,
		args...,
	).Scan(&out.NewVisitors, &out.ReturningVisitors); err != nil {
		return err
	}

	// DAU/WAU/MAU 不按 from 截断，只看 to 往前 30 天；有空筛选时 $2 不会被引用，
	// 需要单独生成参数，避免 pgx 无法推断未使用参数的类型。
	dauArgs := []any{siteID, to}
	endpoint := "$2"
	if scope != "" {
		dauArgs = args
		endpoint = "$3"
	}

	dauQuery := fmt.Sprintf(`
		select
			count(distinct visitor_id) filter (where started_at >= %s::timestamptz - interval '1 day'),
			count(distinct visitor_id) filter (where started_at >= %s::timestamptz - interval '7 days'),
			count(distinct visitor_id) filter (where started_at >= %s::timestamptz - interval '30 days')
		from sessions_web
		where site_id = $1 and started_at >= %s::timestamptz - interval '30 days' and started_at < %s`+scope,
		endpoint, endpoint, endpoint, endpoint, endpoint)

	return s.Pool.QueryRow(ctx, dauQuery, dauArgs...).Scan(&out.DAU, &out.WAU, &out.MAU)
}

// Series 返回按时间桶聚合的 PV/UV。
func (s *Store) Series(ctx context.Context, siteID uuid.UUID, from, to time.Time, interval, timezone string, filter StatsFilter) ([]TimeSeriesPoint, error) {
	scope, scopeArgs := filter.scopeSQL(6)
	args := append([]any{siteID, from, to, interval, timezone}, scopeArgs...)

	rows, err := s.Pool.Query(ctx, `
		select
			(date_trunc($4, created_at at time zone $5) at time zone $5) as bucket,
			count(*) as pv,
			count(distinct visitor_id) as uv
		from events
		where site_id = $1 and created_at >= $2 and created_at < $3 and type = 'pageview'`+scope+`
		group by 1
		order by 1`,
		args...,
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

var durationLabels = []string{"<10 秒", "10-30 秒", "30-60 秒", "1-3 分", "3-10 分", "≥10 分"}

func durationBucketStats(counts [6]int64) []CountStat {
	out := make([]CountStat, 0, len(durationLabels))
	for i, label := range durationLabels {
		out = append(out, CountStat{Value: label, Count: counts[i]})
	}
	return out
}

// DurationBuckets 返回站点会话时长分布。
func (s *Store) DurationBuckets(ctx context.Context, siteID uuid.UUID, from, to time.Time, filter StatsFilter) ([]CountStat, error) {
	scope, scopeArgs := filter.scopeSQL(4)
	args := append([]any{siteID, from, to}, scopeArgs...)

	var counts [6]int64
	if err := s.Pool.QueryRow(ctx, `
		select
			count(*) filter (where d < 10),
			count(*) filter (where d >= 10 and d < 30),
			count(*) filter (where d >= 30 and d < 60),
			count(*) filter (where d >= 60 and d < 180),
			count(*) filter (where d >= 180 and d < 600),
			count(*) filter (where d >= 600)
		from (
			select extract(epoch from (last_seen_at - started_at)) as d
			from sessions_web
			where site_id = $1 and started_at >= $2 and started_at < $3`+scope+`
		) t`,
		args...,
	).Scan(&counts[0], &counts[1], &counts[2], &counts[3], &counts[4], &counts[5]); err != nil {
		return nil, err
	}

	return durationBucketStats(counts), nil
}

// WorkspaceDurationBuckets 返回工作区会话时长分布。
func (s *Store) WorkspaceDurationBuckets(ctx context.Context, workspaceID uuid.UUID, from, to time.Time) ([]CountStat, error) {
	var counts [6]int64
	if err := s.Pool.QueryRow(ctx, `
		select
			count(*) filter (where d < 10),
			count(*) filter (where d >= 10 and d < 30),
			count(*) filter (where d >= 30 and d < 60),
			count(*) filter (where d >= 60 and d < 180),
			count(*) filter (where d >= 180 and d < 600),
			count(*) filter (where d >= 600)
		from (
			select extract(epoch from (sw.last_seen_at - sw.started_at)) as d
			from sessions_web sw
			join sites s on s.id = sw.site_id
			where s.workspace_id = $1 and sw.started_at >= $2 and sw.started_at < $3
		) t`,
		workspaceID, from, to,
	).Scan(&counts[0], &counts[1], &counts[2], &counts[3], &counts[4], &counts[5]); err != nil {
		return nil, err
	}

	return durationBucketStats(counts), nil
}

// WorkspaceOverview 返回工作区全部站点在区间内的核心指标。
func (s *Store) WorkspaceOverview(ctx context.Context, workspaceID uuid.UUID, from, to time.Time) (StatsOverview, error) {
	var out StatsOverview

	if err := s.Pool.QueryRow(ctx, `
		select
			count(*) filter (where e.type = 'pageview'),
			count(distinct e.visitor_id) filter (where e.type = 'pageview')
		from events e
		join sites s on s.id = e.site_id
		where s.workspace_id = $1 and e.created_at >= $2 and e.created_at < $3`,
		workspaceID, from, to,
	).Scan(&out.PV, &out.UV); err != nil {
		return out, err
	}

	if err := s.Pool.QueryRow(ctx, `
		select
			count(*),
			count(*) filter (where sw.pageviews = 1),
			coalesce(sum(extract(epoch from (sw.last_seen_at - sw.started_at))), 0)::bigint
		from sessions_web sw
		join sites s on s.id = sw.site_id
		where s.workspace_id = $1 and sw.started_at >= $2 and sw.started_at < $3`,
		workspaceID, from, to,
	).Scan(&out.Sessions, &out.Bounces, &out.DurationSum); err != nil {
		return out, err
	}

	if err := s.decorateWorkspaceAudience(ctx, workspaceID, from, to, &out); err != nil {
		return out, err
	}

	return out, nil
}

func (s *Store) decorateWorkspaceAudience(ctx context.Context, workspaceID uuid.UUID, from, to time.Time, out *StatsOverview) error {
	if err := s.Pool.QueryRow(ctx, `
		with active as (
			select distinct sw.visitor_id
			from sessions_web sw
			join sites s on s.id = sw.site_id
			where s.workspace_id = $1 and sw.started_at >= $2 and sw.started_at < $3
		), first_seen as (
			select sw.visitor_id, min(sw.started_at) as first_at
			from sessions_web sw
			join sites s on s.id = sw.site_id
			where s.workspace_id = $1
			group by sw.visitor_id
		)
		select
			count(*) filter (where f.first_at >= $2),
			count(*) filter (where f.first_at < $2)
		from active a
		join first_seen f on f.visitor_id = a.visitor_id`,
		workspaceID, from, to,
	).Scan(&out.NewVisitors, &out.ReturningVisitors); err != nil {
		return err
	}

	return s.Pool.QueryRow(ctx, `
		select
			count(distinct sw.visitor_id) filter (where sw.started_at >= $2::timestamptz - interval '1 day'),
			count(distinct sw.visitor_id) filter (where sw.started_at >= $2::timestamptz - interval '7 days'),
			count(distinct sw.visitor_id) filter (where sw.started_at >= $2::timestamptz - interval '30 days')
		from sessions_web sw
		join sites s on s.id = sw.site_id
		where s.workspace_id = $1 and sw.started_at >= $2::timestamptz - interval '30 days' and sw.started_at < $2`,
		workspaceID, to,
	).Scan(&out.DAU, &out.WAU, &out.MAU)
}

// WorkspaceSeries 返回工作区全部站点按时间桶聚合的 PV/UV。
func (s *Store) WorkspaceSeries(ctx context.Context, workspaceID uuid.UUID, from, to time.Time, interval, timezone string) ([]TimeSeriesPoint, error) {
	rows, err := s.Pool.Query(ctx, `
		select
			(date_trunc($4, e.created_at at time zone $5) at time zone $5) as bucket,
			count(*) as pv,
			count(distinct e.visitor_id) as uv
		from events e
		join sites s on s.id = e.site_id
		where s.workspace_id = $1 and e.created_at >= $2 and e.created_at < $3 and e.type = 'pageview'
		group by 1
		order by 1`,
		workspaceID, from, to, interval, timezone,
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

// WorkspaceSiteStats 返回工作区内每个站点在区间内的统计。
func (s *Store) WorkspaceSiteStats(ctx context.Context, workspaceID uuid.UUID, from, to time.Time) ([]SiteStatsSummary, error) {
	rows, err := s.Pool.Query(ctx, `
		with pv as (
			select
				e.site_id,
				count(*) filter (where e.type = 'pageview') as pv,
				count(distinct e.visitor_id) filter (where e.type = 'pageview') as uv
			from events e
			join sites s on s.id = e.site_id
			where s.workspace_id = $1 and e.created_at >= $2 and e.created_at < $3
			group by e.site_id
		), se as (
			select sw.site_id, count(*) as sessions
			from sessions_web sw
			join sites s on s.id = sw.site_id
			where s.workspace_id = $1 and sw.started_at >= $2 and sw.started_at < $3
			group by sw.site_id
		)
		select s.id, s.name, coalesce(s.domain, ''), coalesce(pv.pv, 0), coalesce(pv.uv, 0), coalesce(se.sessions, 0)
		from sites s
		left join pv on pv.site_id = s.id
		left join se on se.site_id = s.id
		where s.workspace_id = $1
		order by s.created_at`,
		workspaceID, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]SiteStatsSummary, 0, 4)
	for rows.Next() {
		var item SiteStatsSummary
		if err := rows.Scan(&item.SiteID, &item.Name, &item.Domain, &item.PV, &item.UV, &item.Sessions); err != nil {
			return nil, err
		}
		list = append(list, item)
	}

	return list, rows.Err()
}

// TopPages 返回访问量最高的页面。
func (s *Store) TopPages(ctx context.Context, siteID uuid.UUID, from, to time.Time, filter StatsFilter) ([]PageStat, error) {
	scope, scopeArgs := filter.scopeSQL(4)
	args := append([]any{siteID, from, to}, scopeArgs...)

	rows, err := s.Pool.Query(ctx, `
		select coalesce(path, '/') as path, coalesce(max(title), '') as title,
		       count(*) as views, count(distinct visitor_id) as visitors
		from events
		where site_id = $1 and created_at >= $2 and created_at < $3 and type = 'pageview'`+scope+`
		group by 1
		order by views desc
		limit 100`,
		args...,
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
func (s *Store) SessionPages(ctx context.Context, siteID uuid.UUID, from, to time.Time, entry bool, filter StatsFilter) ([]CountStat, error) {
	column := "exit_path"
	if entry {
		column = "entry_path"
	}

	scope, scopeArgs := filter.scopeSQL(4)
	args := append([]any{siteID, from, to}, scopeArgs...)

	query := fmt.Sprintf(`
		select %s as value, count(*) as count
		from sessions_web
		where site_id = $1 and started_at >= $2 and started_at < $3 and %s is not null%s
		group by 1
		order by count desc
		limit 100`, column, column, scope)

	return s.queryCountStats(ctx, query, args...)
}

// Referrers 返回来源域名排行。
func (s *Store) Referrers(ctx context.Context, siteID uuid.UUID, from, to time.Time, filter StatsFilter) ([]CountStat, error) {
	scope, scopeArgs := filter.scopeSQL(4)
	args := append([]any{siteID, from, to}, scopeArgs...)

	return s.queryCountStats(ctx, `
		select coalesce(nullif(referrer_host, ''), '(direct)') as value, count(*) as count
		from events
		where site_id = $1 and created_at >= $2 and created_at < $3 and type = 'pageview'`+scope+`
		group by 1
		order by count desc
		limit 100`, args...)
}

// ReferrerURLs 返回完整来源页面排行。
func (s *Store) ReferrerURLs(ctx context.Context, siteID uuid.UUID, from, to time.Time, filter StatsFilter) ([]CountStat, error) {
	scope, scopeArgs := filter.scopeSQL(4)
	args := append([]any{siteID, from, to}, scopeArgs...)

	return s.queryCountStats(ctx, `
		select referrer as value, count(*) as count
		from events
		where site_id = $1 and created_at >= $2 and created_at < $3 and type = 'pageview'
		  and coalesce(referrer, '') <> ''`+scope+`
		group by 1
		order by count desc
		limit 50`, args...)
}

// Channels 返回渠道排行。
func (s *Store) Channels(ctx context.Context, siteID uuid.UUID, from, to time.Time, filter StatsFilter) ([]CountStat, error) {
	scope, scopeArgs := filter.scopeSQL(4)
	args := append([]any{siteID, from, to}, scopeArgs...)

	return s.queryCountStats(ctx, `
		select coalesce(nullif(channel, ''), '(unknown)') as value, count(*) as count
		from events
		where site_id = $1 and created_at >= $2 and created_at < $3 and type = 'pageview'`+scope+`
		group by 1
		order by count desc`, args...)
}

// UTMStats 返回指定 UTM 字段的排行。
func (s *Store) UTMStats(ctx context.Context, siteID uuid.UUID, from, to time.Time, field string, filter StatsFilter) ([]CountStat, error) {
	column := ""
	switch field {
	case "utm_source":
		column = "utm_source"
	case "utm_medium":
		column = "utm_medium"
	case "utm_campaign":
		column = "utm_campaign"
	case "utm_term":
		column = "utm_term"
	case "utm_content":
		column = "utm_content"
	default:
		return nil, fmt.Errorf("不支持的 UTM 字段: %s", field)
	}

	scope, scopeArgs := filter.scopeSQL(4)
	args := append([]any{siteID, from, to}, scopeArgs...)

	query := fmt.Sprintf(`
		select %s as value, count(*) as count
		from events
		where site_id = $1 and created_at >= $2 and created_at < $3 and type = 'pageview' and %s <> ''%s
		group by 1
		order by count desc
		limit 50`, column, column, scope)

	return s.queryCountStats(ctx, query, args...)
}

var audienceFields = map[string]bool{
	"browser":         true,
	"browser_version": true,
	"os":              true,
	"os_version":      true,
	"device":          true,
	"device_brand":    true,
	"screen":          true,
	"language":        true,
	"country":         true,
	"region":          true,
	"city":            true,
}

// Breakdown 返回访客画像某一维度的排行。
func (s *Store) Breakdown(ctx context.Context, siteID uuid.UUID, from, to time.Time, field string, filter StatsFilter) ([]CountStat, error) {
	if !audienceFields[field] {
		return nil, fmt.Errorf("不支持的维度: %s", field)
	}

	scope, scopeArgs := filter.scopeSQL(4)
	args := append([]any{siteID, from, to}, scopeArgs...)

	query := fmt.Sprintf(`
		select %s as value, count(*) as count
		from events
		where site_id = $1 and created_at >= $2 and created_at < $3 and type = 'pageview' and %s is not null and %s <> ''%s
		group by 1
		order by count desc
		limit 50`, field, field, field, scope)

	return s.queryCountStats(ctx, query, args...)
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
func (s *Store) CustomEvents(ctx context.Context, siteID uuid.UUID, from, to time.Time, filter StatsFilter) ([]EventStat, error) {
	scope, scopeArgs := filter.scopeSQL(4)
	args := append([]any{siteID, from, to}, scopeArgs...)

	rows, err := s.Pool.Query(ctx, `
		select name, count(*) as count, count(distinct visitor_id) as visitors, max(created_at) as last_seen
		from events
		where site_id = $1 and created_at >= $2 and created_at < $3
		  and type = 'custom' and name is not null and name <> ''`+scope+`
		group by name
		order by count desc
		limit 100`,
		args...,
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
func (s *Store) EventSeries(ctx context.Context, siteID uuid.UUID, from, to time.Time, name, interval, timezone string, filter StatsFilter) ([]TimeSeriesPoint, error) {
	scope, scopeArgs := filter.scopeSQL(7)
	args := append([]any{siteID, from, to, name, interval, timezone}, scopeArgs...)

	rows, err := s.Pool.Query(ctx, `
		select
			(date_trunc($5, created_at at time zone $6) at time zone $6) as bucket,
			count(*) as pv,
			count(distinct visitor_id) as uv
		from events
		where site_id = $1 and created_at >= $2 and created_at < $3
		  and type = 'custom' and name = $4`+scope+`
		group by 1
		order by 1`,
		args...,
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
func (s *Store) EventProps(ctx context.Context, siteID uuid.UUID, from, to time.Time, name string, filter StatsFilter) ([]EventPropStat, error) {
	scope, scopeArgs := filter.scopeSQLFor("e", 5)
	args := append([]any{siteID, from, to, name}, scopeArgs...)

	rows, err := s.Pool.Query(ctx, `
		select p.key, p.value, count(*) as count
		from events e
		cross join lateral jsonb_each_text(coalesce(e.props, '{}'::jsonb)) as p(key, value)
		where e.site_id = $1 and e.created_at >= $2 and e.created_at < $3
		  and e.type = 'custom' and e.name = $4`+scope+`
		group by p.key, p.value
		order by p.key, count desc
		limit 200`,
		args...,
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
