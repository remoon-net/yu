package monitor

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"strconv"
	"time"

	"remoon.net/yu/backend/internal/monitor/probe"
	"remoon.net/yu/backend/internal/store"
)

const notifyInterval = 30 * time.Minute

// Evaluator 负责维护告警状态机并触发通知与事件。
type Evaluator struct {
	store  *store.Store
	sender *Sender
}

// NewEvaluator 创建告警评估器。
func NewEvaluator(st *store.Store, sender *Sender) *Evaluator {
	return &Evaluator{store: st, sender: sender}
}

// Process 在一次探测完成后推进状态机。
func (e *Evaluator) Process(ctx context.Context, monitor store.Monitor, result probe.Result, checkedAt time.Time) {
	state, err := e.store.GetOrCreateAlertState(ctx, monitor.ID)
	if err != nil {
		slog.Error("读取告警状态失败", "err", err, "monitor", monitor.ID)
		return
	}

	previous := state.State
	muted := state.MutedUntil != nil && state.MutedUntil.After(checkedAt)

	maintenance, err := e.store.IsUnderMaintenance(ctx, monitor.WorkspaceID, monitor.ID, checkedAt)
	if err != nil {
		slog.Warn("读取维护窗口失败", "err", err, "monitor", monitor.ID)
	}

	channels, err := e.store.ListChannels(ctx, monitor.WorkspaceID)
	if err != nil {
		slog.Warn("读取通知渠道失败", "err", err, "workspace", monitor.WorkspaceID)
	}

	workspaceName := ""
	if workspace, err := e.store.GetWorkspace(ctx, monitor.WorkspaceID); err == nil {
		workspaceName = workspace.Name
	}

	threshold := monitor.AlertAfter
	if threshold <= 0 {
		threshold = 2
	}

	switch result.Status {
	case "up":
		state.ConsecutiveFailures = 0
		state.ConsecutiveSuccesses++

		switch previous {
		case "down":
			state.State = "recovering"
			state.Since = checkedAt

		case "recovering":
			state.State = "up"
			state.Since = checkedAt
			state.DownSince = nil

			if state.IncidentID != nil {
				if err := e.store.ResolveIncident(ctx, monitor.WorkspaceID, *state.IncidentID); err != nil {
					slog.Warn("关闭事件失败", "err", err, "incident", *state.IncidentID)
				} else {
					state.IncidentID = nil
				}
			}

			if !muted && !maintenance {
				e.send(channels, monitor, workspaceName, "monitor.up", result, checkedAt, 0)
				state.LastNotifiedAt = &checkedAt
			}

		case "pending":
			state.State = "up"
			state.Since = checkedAt
			state.DownSince = nil

		default:
			state.State = "up"
		}

	case "down":
		state.ConsecutiveSuccesses = 0
		state.ConsecutiveFailures++

		enteredDown := false

		switch previous {
		case "up", "pending", "recovering":
			if state.ConsecutiveFailures >= threshold {
				state.State = "down"
				state.Since = checkedAt
				state.DownSince = &checkedAt
				enteredDown = true
			} else {
				state.State = "pending"
			}
		case "down":
			// 保持 down
		}

		if state.State == "down" {
			if state.IncidentID == nil && !maintenance {
				incident, err := e.store.CreateIncident(
					ctx, monitor.WorkspaceID, &monitor.ID,
					monitor.Name+" 不可用", result.Error, "major", "auto",
				)
				if err != nil {
					slog.Warn("创建事件失败", "err", err, "monitor", monitor.ID)
				} else {
					state.IncidentID = &incident.ID
				}
			}

			// 新进入 down 状态时立即通知；持续 down 时才按间隔去重。
			if !muted && !maintenance && (enteredDown || shouldNotify(state.LastNotifiedAt, checkedAt)) {
				e.send(channels, monitor, workspaceName, "monitor.down", result, checkedAt, 0)
				state.LastNotifiedAt = &checkedAt
			}
		}
	}

	if monitor.Type == "http" && result.CertExpiresAt != nil && monitor.SSLAlertDays > 0 {
		daysLeft := int(math.Floor(result.CertExpiresAt.Sub(checkedAt).Hours() / 24))

		if daysLeft <= monitor.SSLAlertDays && shouldNotifySSL(state.SSLNotifiedAt, checkedAt) {
			if !muted && !maintenance {
				e.send(channels, monitor, workspaceName, "monitor.ssl_expiring", result, checkedAt, daysLeft)
				state.SSLNotifiedAt = &checkedAt
			}
		}
	}

	if err := e.store.SaveAlertState(ctx, state); err != nil {
		slog.Error("保存告警状态失败", "err", err, "monitor", monitor.ID)
	}
}

func (e *Evaluator) send(
	channels []store.NotificationChannel,
	monitor store.Monitor,
	workspaceName, event string,
	result probe.Result,
	checkedAt time.Time,
	daysLeft int,
) {
	if len(channels) == 0 {
		return
	}

	e.sender.SendAll(context.Background(), channels, AlertMessage{
		Event:         event,
		WorkspaceName: workspaceName,
		MonitorName:   monitor.Name,
		MonitorType:   monitor.Type,
		Target:        monitorTarget(monitor),
		Status:        result.Status,
		LatencyMS:     result.LatencyMS,
		Error:         result.Error,
		CheckedAt:     checkedAt,
		CertExpiresAt: result.CertExpiresAt,
		DaysLeft:      daysLeft,
	})
}

func shouldNotify(last *time.Time, now time.Time) bool {
	if last == nil {
		return true
	}
	return now.Sub(*last) >= notifyInterval
}

func shouldNotifySSL(last *time.Time, now time.Time) bool {
	if last == nil {
		return true
	}
	lastYear, lastMonth, lastDay := last.UTC().Date()
	nowYear, nowMonth, nowDay := now.UTC().Date()
	return lastYear != nowYear || lastMonth != nowMonth || lastDay != nowDay
}

func monitorTarget(monitor store.Monitor) string {
	switch monitor.Type {
	case "http":
		var cfg struct {
			URL string `json:"url"`
		}
		_ = json.Unmarshal(monitor.Config, &cfg)
		return cfg.URL

	case "tcp":
		var cfg struct {
			Host string `json:"host"`
			Port int    `json:"port"`
		}
		_ = json.Unmarshal(monitor.Config, &cfg)
		return cfg.Host + ":" + strconv.Itoa(cfg.Port)

	case "ping":
		var cfg struct {
			Host string `json:"host"`
		}
		_ = json.Unmarshal(monitor.Config, &cfg)
		return cfg.Host

	case "dns":
		var cfg struct {
			Name       string `json:"name"`
			RecordType string `json:"record_type"`
		}
		_ = json.Unmarshal(monitor.Config, &cfg)
		return cfg.Name + " " + cfg.RecordType
	}

	return ""
}
