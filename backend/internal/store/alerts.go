package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// AlertState 是监控的告警状态机。
type AlertState struct {
	MonitorID            uuid.UUID  `json:"monitor_id"`
	State                string     `json:"state"`
	ConsecutiveFailures  int        `json:"consecutive_failures"`
	ConsecutiveSuccesses int        `json:"consecutive_successes"`
	Since                time.Time  `json:"since"`
	DownSince            *time.Time `json:"down_since"`
	LastNotifiedAt       *time.Time `json:"last_notified_at"`
	MutedUntil           *time.Time `json:"muted_until"`
	IncidentID           *uuid.UUID `json:"incident_id"`
	SSLNotifiedAt        *time.Time `json:"ssl_notified_at"`
}

// NotificationChannel 是告警通知渠道。
type NotificationChannel struct {
	ID          uuid.UUID       `json:"id"`
	WorkspaceID uuid.UUID       `json:"workspace_id"`
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Config      json.RawMessage `json:"config"`
	Enabled     bool            `json:"enabled"`
	CreatedAt   time.Time       `json:"created_at"`
}

// Incident 是一次故障事件。
type Incident struct {
	ID          uuid.UUID  `json:"id"`
	WorkspaceID uuid.UUID  `json:"workspace_id"`
	MonitorID   *uuid.UUID `json:"monitor_id"`
	MonitorName string     `json:"monitor_name"`
	Title       string     `json:"title"`
	Body        string     `json:"body"`
	Severity    string     `json:"severity"`
	Source      string     `json:"source"`
	StartedAt   time.Time  `json:"started_at"`
	ResolvedAt  *time.Time `json:"resolved_at"`
}

// MaintenanceWindow 是一段维护窗口。
type MaintenanceWindow struct {
	ID          uuid.UUID   `json:"id"`
	WorkspaceID uuid.UUID   `json:"workspace_id"`
	Title       string      `json:"title"`
	MonitorIDs  []uuid.UUID `json:"monitor_ids"`
	StartsAt    time.Time   `json:"starts_at"`
	EndsAt      time.Time   `json:"ends_at"`
}

// GetOrCreateAlertState 读取告警状态，不存在时初始化。
func (s *Store) GetOrCreateAlertState(ctx context.Context, monitorID uuid.UUID) (*AlertState, error) {
	if _, err := s.Pool.Exec(ctx, `
		insert into alert_states (monitor_id) values ($1)
		on conflict (monitor_id) do nothing`, monitorID); err != nil {
		return nil, err
	}

	state := &AlertState{}
	err := s.Pool.QueryRow(ctx, `
		select monitor_id, state, consecutive_failures, consecutive_successes, since,
		       down_since, last_notified_at, muted_until, incident_id, ssl_notified_at
		from alert_states
		where monitor_id = $1`, monitorID,
	).Scan(
		&state.MonitorID, &state.State, &state.ConsecutiveFailures, &state.ConsecutiveSuccesses,
		&state.Since, &state.DownSince, &state.LastNotifiedAt, &state.MutedUntil,
		&state.IncidentID, &state.SSLNotifiedAt,
	)
	if err != nil {
		return nil, err
	}

	return state, nil
}

// SaveAlertState 持久化告警状态。
func (s *Store) SaveAlertState(ctx context.Context, state *AlertState) error {
	_, err := s.Pool.Exec(ctx, `
		update alert_states set
			state = $2, consecutive_failures = $3, consecutive_successes = $4,
			since = $5, down_since = $6, last_notified_at = $7, muted_until = $8,
			incident_id = $9, ssl_notified_at = $10, updated_at = now()
		where monitor_id = $1`,
		state.MonitorID, state.State, state.ConsecutiveFailures, state.ConsecutiveSuccesses,
		state.Since, state.DownSince, state.LastNotifiedAt, state.MutedUntil,
		state.IncidentID, state.SSLNotifiedAt,
	)
	return err
}

// SetMuted 设置或解除静默。
func (s *Store) SetMuted(ctx context.Context, monitorID uuid.UUID, until *time.Time) error {
	_, err := s.Pool.Exec(ctx, `
		insert into alert_states (monitor_id, muted_until) values ($1, $2)
		on conflict (monitor_id) do update set muted_until = excluded.muted_until, updated_at = now()`,
		monitorID, until,
	)
	return err
}

// ListChannels 返回工作区的通知渠道。
func (s *Store) ListChannels(ctx context.Context, workspaceID uuid.UUID) ([]NotificationChannel, error) {
	rows, err := s.Pool.Query(ctx, `
		select id, workspace_id, type, name, config, enabled, created_at
		from notification_channels
		where workspace_id = $1
		order by created_at`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]NotificationChannel, 0, 4)
	for rows.Next() {
		var item NotificationChannel
		if err := rows.Scan(&item.ID, &item.WorkspaceID, &item.Type, &item.Name, &item.Config, &item.Enabled, &item.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, item)
	}

	return list, rows.Err()
}

// GetChannel 读取单个通知渠道。
func (s *Store) GetChannel(ctx context.Context, workspaceID, channelID uuid.UUID) (*NotificationChannel, error) {
	item := &NotificationChannel{}
	err := s.Pool.QueryRow(ctx, `
		select id, workspace_id, type, name, config, enabled, created_at
		from notification_channels
		where id = $1 and workspace_id = $2`, channelID, workspaceID,
	).Scan(&item.ID, &item.WorkspaceID, &item.Type, &item.Name, &item.Config, &item.Enabled, &item.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return item, nil
}

// CreateChannel 创建通知渠道。
func (s *Store) CreateChannel(ctx context.Context, workspaceID uuid.UUID, channelType, name string, config json.RawMessage, enabled bool) (*NotificationChannel, error) {
	id := uuid.New()

	_, err := s.Pool.Exec(ctx, `
		insert into notification_channels (id, workspace_id, type, name, config, enabled)
		values ($1, $2, $3, $4, $5, $6)`,
		id, workspaceID, channelType, name, config, enabled,
	)
	if err != nil {
		return nil, err
	}

	return s.GetChannel(ctx, workspaceID, id)
}

// UpdateChannel 更新通知渠道。
func (s *Store) UpdateChannel(ctx context.Context, workspaceID, channelID uuid.UUID, name string, config json.RawMessage, enabled bool) (*NotificationChannel, error) {
	tag, err := s.Pool.Exec(ctx, `
		update notification_channels set name = $3, config = $4, enabled = $5
		where id = $1 and workspace_id = $2`,
		channelID, workspaceID, name, config, enabled,
	)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}

	return s.GetChannel(ctx, workspaceID, channelID)
}

// DeleteChannel 删除通知渠道。
func (s *Store) DeleteChannel(ctx context.Context, workspaceID, channelID uuid.UUID) error {
	tag, err := s.Pool.Exec(ctx, `
		delete from notification_channels where id = $1 and workspace_id = $2`, channelID, workspaceID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateIncident 创建故障事件。
func (s *Store) CreateIncident(
	ctx context.Context,
	workspaceID uuid.UUID,
	monitorID *uuid.UUID,
	title, body, severity, source string,
) (*Incident, error) {
	incident := &Incident{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		MonitorID:   monitorID,
		Title:       title,
		Body:        body,
		Severity:    severity,
		Source:      source,
		StartedAt:   time.Now().UTC(),
	}

	_, err := s.Pool.Exec(ctx, `
		insert into incidents (id, workspace_id, monitor_id, title, body, severity, source, started_at)
		values ($1, $2, $3, $4, $5, $6, $7, $8)`,
		incident.ID, incident.WorkspaceID, incident.MonitorID, incident.Title,
		nullable(incident.Body), incident.Severity, incident.Source, incident.StartedAt,
	)
	if err != nil {
		return nil, err
	}

	return incident, nil
}

// ResolveIncident 标记事件已恢复。
func (s *Store) ResolveIncident(ctx context.Context, workspaceID, incidentID uuid.UUID) error {
	tag, err := s.Pool.Exec(ctx, `
		update incidents set resolved_at = now()
		where id = $1 and workspace_id = $2 and resolved_at is null`,
		incidentID, workspaceID,
	)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		var exists bool
		if err := s.Pool.QueryRow(ctx, `
			select exists (select 1 from incidents where id = $1 and workspace_id = $2)`,
			incidentID, workspaceID,
		).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return ErrNotFound
		}
	}

	return nil
}

// ListIncidents 返回工作区的故障事件。
func (s *Store) ListIncidents(ctx context.Context, workspaceID uuid.UUID, limit int) ([]Incident, error) {
	rows, err := s.Pool.Query(ctx, `
		select i.id, i.workspace_id, i.monitor_id, coalesce(m.name, ''),
		       i.title, coalesce(i.body, ''), i.severity, i.source, i.started_at, i.resolved_at
		from incidents i
		left join monitors m on m.id = i.monitor_id
		where i.workspace_id = $1
		order by i.started_at desc
		limit $2`, workspaceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]Incident, 0, limit)
	for rows.Next() {
		var item Incident
		if err := rows.Scan(
			&item.ID, &item.WorkspaceID, &item.MonitorID, &item.MonitorName,
			&item.Title, &item.Body, &item.Severity, &item.Source, &item.StartedAt, &item.ResolvedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, item)
	}

	return list, rows.Err()
}

// ListMaintenanceWindows 返回工作区的维护窗口。
func (s *Store) ListMaintenanceWindows(ctx context.Context, workspaceID uuid.UUID) ([]MaintenanceWindow, error) {
	rows, err := s.Pool.Query(ctx, `
		select id, workspace_id, title, monitor_ids, starts_at, ends_at
		from maintenance_windows
		where workspace_id = $1
		order by starts_at desc`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]MaintenanceWindow, 0, 4)
	for rows.Next() {
		var item MaintenanceWindow
		if err := rows.Scan(&item.ID, &item.WorkspaceID, &item.Title, &item.MonitorIDs, &item.StartsAt, &item.EndsAt); err != nil {
			return nil, err
		}
		list = append(list, item)
	}

	return list, rows.Err()
}

// CreateMaintenanceWindow 创建维护窗口。
func (s *Store) CreateMaintenanceWindow(
	ctx context.Context,
	workspaceID uuid.UUID,
	title string,
	monitorIDs []uuid.UUID,
	startsAt, endsAt time.Time,
) (*MaintenanceWindow, error) {
	window := &MaintenanceWindow{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		Title:       title,
		MonitorIDs:  monitorIDs,
		StartsAt:    startsAt,
		EndsAt:      endsAt,
	}

	_, err := s.Pool.Exec(ctx, `
		insert into maintenance_windows (id, workspace_id, title, monitor_ids, starts_at, ends_at)
		values ($1, $2, $3, $4, $5, $6)`,
		window.ID, window.WorkspaceID, window.Title, window.MonitorIDs, window.StartsAt, window.EndsAt,
	)
	if err != nil {
		return nil, err
	}

	return window, nil
}

// DeleteMaintenanceWindow 删除维护窗口。
func (s *Store) DeleteMaintenanceWindow(ctx context.Context, workspaceID, windowID uuid.UUID) error {
	tag, err := s.Pool.Exec(ctx, `
		delete from maintenance_windows where id = $1 and workspace_id = $2`, windowID, workspaceID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// IsUnderMaintenance 判断监控在指定时间是否处于维护窗口。
func (s *Store) IsUnderMaintenance(ctx context.Context, workspaceID, monitorID uuid.UUID, at time.Time) (bool, error) {
	var covered bool
	err := s.Pool.QueryRow(ctx, `
		select exists (
			select 1 from maintenance_windows
			where workspace_id = $1 and starts_at <= $3 and ends_at > $3
			  and (cardinality(monitor_ids) = 0 or $2 = any (monitor_ids))
		)`, workspaceID, monitorID, at).Scan(&covered)
	return covered, err
}
