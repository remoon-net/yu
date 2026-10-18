package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Monitor 是被监控的目标。
type Monitor struct {
	ID            uuid.UUID       `json:"id"`
	WorkspaceID   uuid.UUID       `json:"workspace_id"`
	Name          string          `json:"name"`
	Type          string          `json:"type"`
	Config        json.RawMessage `json:"config"`
	IntervalSec   int             `json:"interval_sec"`
	TimeoutMS     int             `json:"timeout_ms"`
	Retries       int             `json:"retries"`
	GroupName     string          `json:"group_name"`
	Tags          []string        `json:"tags"`
	SSLAlertDays  int             `json:"ssl_alert_days"`
	Enabled       bool            `json:"enabled"`
	NextRunAt     time.Time       `json:"next_run_at"`
	LastStatus    string          `json:"last_status"`
	LastCheckedAt *time.Time      `json:"last_checked_at"`
	LastLatencyMS *int            `json:"last_latency_ms"`
	LastError     string          `json:"last_error"`
	CreatedAt     time.Time       `json:"created_at"`
}

// MonitorInput 是创建或更新监控时的输入。
type MonitorInput struct {
	WorkspaceID  uuid.UUID
	Name         string
	Type         string
	Config       json.RawMessage
	IntervalSec  int
	TimeoutMS    int
	Retries      int
	GroupName    string
	Tags         []string
	SSLAlertDays int
	Enabled      bool
}

// CheckResult 是一次探测的结果。
type CheckResult struct {
	Status        string
	LatencyMS     int
	StatusCode    int
	Error         string
	CertExpiresAt *time.Time
}

// Check 是一条探测记录。
type Check struct {
	ID            int64      `json:"id"`
	CheckedAt     time.Time  `json:"checked_at"`
	Status        string     `json:"status"`
	LatencyMS     int        `json:"latency_ms"`
	StatusCode    int        `json:"status_code"`
	Error         string     `json:"error"`
	CertExpiresAt *time.Time `json:"cert_expires_at"`
}

// MonitorBucket 是监控统计的时间桶。
type MonitorBucket struct {
	Bucket time.Time `json:"bucket"`
	Total  int       `json:"total"`
	Up     int       `json:"up"`
	P50    float64   `json:"p50"`
	P95    float64   `json:"p95"`
	P99    float64   `json:"p99"`
}

// MonitorStats 是监控区间统计。
type MonitorStats struct {
	Checks       int64           `json:"checks"`
	Up           int64           `json:"up"`
	Down         int64           `json:"down"`
	Availability float64         `json:"availability"`
	P50          float64         `json:"p50"`
	P95          float64         `json:"p95"`
	P99          float64         `json:"p99"`
	Series       []MonitorBucket `json:"series"`
	Recent       []Check         `json:"recent"`
}

const monitorColumns = `
	id, workspace_id, name, type, config, interval_sec, timeout_ms, retries,
	coalesce(group_name, ''), tags, ssl_alert_days, enabled, next_run_at,
	coalesce(last_status, ''), last_checked_at, last_latency_ms, coalesce(last_error, ''), created_at`

func scanMonitor(row pgx.Row) (*Monitor, error) {
	m := &Monitor{}
	err := row.Scan(
		&m.ID, &m.WorkspaceID, &m.Name, &m.Type, &m.Config, &m.IntervalSec, &m.TimeoutMS, &m.Retries,
		&m.GroupName, &m.Tags, &m.SSLAlertDays, &m.Enabled, &m.NextRunAt,
		&m.LastStatus, &m.LastCheckedAt, &m.LastLatencyMS, &m.LastError, &m.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return m, nil
}

// CreateMonitor 创建监控。
func (s *Store) CreateMonitor(ctx context.Context, in MonitorInput) (*Monitor, error) {
	id := uuid.New()

	_, err := s.Pool.Exec(ctx, `
		insert into monitors (
			id, workspace_id, name, type, config, interval_sec, timeout_ms, retries,
			group_name, tags, ssl_alert_days, enabled, next_run_at
		) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, now())`,
		id, in.WorkspaceID, in.Name, in.Type, in.Config, in.IntervalSec, in.TimeoutMS, in.Retries,
		nullable(in.GroupName), in.Tags, in.SSLAlertDays, in.Enabled,
	)
	if err != nil {
		return nil, err
	}

	return s.GetMonitor(ctx, in.WorkspaceID, id)
}

// ListMonitors 返回工作区下的监控。
func (s *Store) ListMonitors(ctx context.Context, workspaceID uuid.UUID) ([]Monitor, error) {
	rows, err := s.Pool.Query(ctx, `
		select `+monitorColumns+`
		from monitors
		where workspace_id = $1
		order by created_at desc`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]Monitor, 0, 8)
	for rows.Next() {
		m, err := scanMonitor(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *m)
	}

	return list, rows.Err()
}

// GetMonitor 按工作区与 ID 查询监控。
func (s *Store) GetMonitor(ctx context.Context, workspaceID, monitorID uuid.UUID) (*Monitor, error) {
	return scanMonitor(s.Pool.QueryRow(ctx, `
		select `+monitorColumns+`
		from monitors
		where id = $1 and workspace_id = $2`, monitorID, workspaceID))
}

// UpdateMonitor 更新监控配置。
func (s *Store) UpdateMonitor(ctx context.Context, monitorID uuid.UUID, in MonitorInput) (*Monitor, error) {
	tag, err := s.Pool.Exec(ctx, `
		update monitors set
			name = $2, type = $3, config = $4, interval_sec = $5, timeout_ms = $6,
			retries = $7, group_name = $8, tags = $9, ssl_alert_days = $10, enabled = $11,
			next_run_at = now()
		where id = $1 and workspace_id = $12`,
		monitorID, in.Name, in.Type, in.Config, in.IntervalSec, in.TimeoutMS,
		in.Retries, nullable(in.GroupName), in.Tags, in.SSLAlertDays, in.Enabled, in.WorkspaceID,
	)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}

	return s.GetMonitor(ctx, in.WorkspaceID, monitorID)
}

// DeleteMonitor 删除监控及其历史数据。
func (s *Store) DeleteMonitor(ctx context.Context, workspaceID, monitorID uuid.UUID) error {
	tag, err := s.Pool.Exec(ctx, `
		delete from monitors where id = $1 and workspace_id = $2`, monitorID, workspaceID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ListDueMonitors 返回到达执行时间的监控。
func (s *Store) ListDueMonitors(ctx context.Context, limit int) ([]Monitor, error) {
	rows, err := s.Pool.Query(ctx, `
		select `+monitorColumns+`
		from monitors
		where enabled and next_run_at <= now()
		order by next_run_at
		limit $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]Monitor, 0, limit)
	for rows.Next() {
		m, err := scanMonitor(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *m)
	}

	return list, rows.Err()
}

// SetNextRun 提前占用下一次执行时间，避免重复调度。
func (s *Store) SetNextRun(ctx context.Context, monitorID uuid.UUID, next time.Time) error {
	_, err := s.Pool.Exec(ctx, `update monitors set next_run_at = $2 where id = $1`, monitorID, next)
	return err
}

// RecordCheck 写入探测结果并更新监控的最新状态。
func (s *Store) RecordCheck(ctx context.Context, monitorID uuid.UUID, checkedAt time.Time, result CheckResult, nextRun time.Time) error {
	return s.inTx(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			insert into monitor_checks (monitor_id, checked_at, status, latency_ms, status_code, error, cert_expires_at)
			values ($1, $2, $3, $4, $5, $6, $7)`,
			monitorID, checkedAt, result.Status, result.LatencyMS, result.StatusCode,
			nullable(result.Error), result.CertExpiresAt,
		)
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, `
			update monitors set
				last_status = $2, last_checked_at = $3, last_latency_ms = $4,
				last_error = $5, next_run_at = $6
			where id = $1`,
			monitorID, result.Status, checkedAt, result.LatencyMS, nullable(result.Error), nextRun,
		)
		return err
	})
}

// ListChecks 返回区间内的探测记录（按时间倒序）。
func (s *Store) ListChecks(ctx context.Context, monitorID uuid.UUID, from, to time.Time, limit int) ([]Check, error) {
	rows, err := s.Pool.Query(ctx, `
		select id, checked_at, status, coalesce(latency_ms, 0), coalesce(status_code, 0),
		       coalesce(error, ''), cert_expires_at
		from monitor_checks
		where monitor_id = $1 and checked_at >= $2 and checked_at < $3
		order by checked_at desc
		limit $4`, monitorID, from, to, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]Check, 0, limit)
	for rows.Next() {
		var item Check
		if err := rows.Scan(
			&item.ID, &item.CheckedAt, &item.Status, &item.LatencyMS,
			&item.StatusCode, &item.Error, &item.CertExpiresAt,
		); err != nil {
			return nil, err
		}
		list = append(list, item)
	}

	return list, rows.Err()
}

// MonitorStats 计算区间可用率、延迟分位与趋势。
func (s *Store) MonitorStats(ctx context.Context, monitorID uuid.UUID, from, to time.Time, interval string) (MonitorStats, error) {
	var stats MonitorStats

	if err := s.Pool.QueryRow(ctx, `
		select
			count(*),
			count(*) filter (where status = 'up'),
			coalesce(percentile_cont(0.5) within group (order by latency_ms)
				filter (where status = 'up' and latency_ms is not null), 0),
			coalesce(percentile_cont(0.95) within group (order by latency_ms)
				filter (where status = 'up' and latency_ms is not null), 0),
			coalesce(percentile_cont(0.99) within group (order by latency_ms)
				filter (where status = 'up' and latency_ms is not null), 0)
		from monitor_checks
		where monitor_id = $1 and checked_at >= $2 and checked_at < $3`,
		monitorID, from, to,
	).Scan(&stats.Checks, &stats.Up, &stats.P50, &stats.P95, &stats.P99); err != nil {
		return stats, err
	}

	stats.Down = stats.Checks - stats.Up
	if stats.Checks > 0 {
		stats.Availability = float64(stats.Up) / float64(stats.Checks) * 100
	}

	rows, err := s.Pool.Query(ctx, `
		select
			date_trunc($4, checked_at) as bucket,
			count(*),
			count(*) filter (where status = 'up'),
			coalesce(percentile_cont(0.5) within group (order by latency_ms)
				filter (where status = 'up' and latency_ms is not null), 0),
			coalesce(percentile_cont(0.95) within group (order by latency_ms)
				filter (where status = 'up' and latency_ms is not null), 0),
			coalesce(percentile_cont(0.99) within group (order by latency_ms)
				filter (where status = 'up' and latency_ms is not null), 0)
		from monitor_checks
		where monitor_id = $1 and checked_at >= $2 and checked_at < $3
		group by 1
		order by 1`,
		monitorID, from, to, interval,
	)
	if err != nil {
		return stats, err
	}
	defer rows.Close()

	stats.Series = make([]MonitorBucket, 0, 32)
	for rows.Next() {
		var bucket MonitorBucket
		if err := rows.Scan(&bucket.Bucket, &bucket.Total, &bucket.Up, &bucket.P50, &bucket.P95, &bucket.P99); err != nil {
			return stats, err
		}
		stats.Series = append(stats.Series, bucket)
	}
	if err := rows.Err(); err != nil {
		return stats, err
	}

	recent, err := s.ListChecks(ctx, monitorID, from, to, 50)
	if err != nil {
		return stats, err
	}
	stats.Recent = recent

	return stats, nil
}

// AggregateMonitorHourly 将指定小时的原始探测记录聚合到小时表。
func (s *Store) AggregateMonitorHourly(ctx context.Context, from, to time.Time) error {
	_, err := s.Pool.Exec(ctx, `
		insert into monitor_stats_hourly (monitor_id, bucket, total, up_count, p50, p95, p99)
		select
			monitor_id,
			date_trunc('hour', checked_at) as bucket,
			count(*),
			count(*) filter (where status = 'up'),
			coalesce(percentile_cont(0.5) within group (order by latency_ms)
				filter (where status = 'up' and latency_ms is not null), 0)::int,
			coalesce(percentile_cont(0.95) within group (order by latency_ms)
				filter (where status = 'up' and latency_ms is not null), 0)::int,
			coalesce(percentile_cont(0.99) within group (order by latency_ms)
				filter (where status = 'up' and latency_ms is not null), 0)::int
		from monitor_checks
		where checked_at >= $1 and checked_at < $2
		group by monitor_id, date_trunc('hour', checked_at)
		on conflict (monitor_id, bucket) do update set
			total = excluded.total,
			up_count = excluded.up_count,
			p50 = excluded.p50,
			p95 = excluded.p95,
			p99 = excluded.p99`,
		from, to,
	)
	return err
}
