package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"remoon.net/yu/backend/internal/monitor"
	"remoon.net/yu/backend/internal/store"
)

type channelRequest struct {
	Type    string          `json:"type"`
	Name    string          `json:"name"`
	Config  json.RawMessage `json:"config"`
	Enabled *bool           `json:"enabled"`
}

func (s *Server) channelAccess(w http.ResponseWriter, r *http.Request, minRole string) (*store.NotificationChannel, bool) {
	ws, _, ok := s.workspaceAccess(w, r, minRole)
	if !ok {
		return nil, false
	}

	channelID, err := uuid.Parse(chi.URLParam(r, "channelID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "通知渠道不存在")
		return nil, false
	}

	channel, err := s.store.GetChannel(r.Context(), ws.ID, channelID)
	if err != nil {
		writeError(w, http.StatusNotFound, "通知渠道不存在")
		return nil, false
	}

	return channel, true
}

func (s *Server) handleListChannels(w http.ResponseWriter, r *http.Request) {
	ws, _, ok := s.workspaceAccess(w, r, "member")
	if !ok {
		return
	}

	channels, err := s.store.ListChannels(r.Context(), ws.ID)
	if err != nil {
		slog.Error("查询通知渠道失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取通知渠道失败")
		return
	}

	writeJSON(w, http.StatusOK, channels)
}

func (s *Server) handleCreateChannel(w http.ResponseWriter, r *http.Request) {
	ws, _, ok := s.workspaceAccess(w, r, "admin")
	if !ok {
		return
	}

	var req channelRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	channelType := trimSpace(req.Type)
	if channelType != "smtp" && channelType != "webhook" {
		writeError(w, http.StatusBadRequest, "通知类型不合法")
		return
	}

	name := trimSpace(req.Name)
	if name == "" || len([]rune(name)) > 60 {
		writeError(w, http.StatusBadRequest, "渠道名称需要 1-60 个字符")
		return
	}

	config := req.Config
	if len(config) == 0 || string(config) == "null" {
		config = json.RawMessage(`{}`)
	}
	if err := monitor.ValidateChannelConfig(channelType, config); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	channel, err := s.store.CreateChannel(r.Context(), ws.ID, channelType, name, config, enabled)
	if err != nil {
		slog.Error("创建通知渠道失败", "err", err)
		writeError(w, http.StatusInternalServerError, "创建通知渠道失败")
		return
	}

	writeJSON(w, http.StatusCreated, channel)
}

func (s *Server) handleUpdateChannel(w http.ResponseWriter, r *http.Request) {
	channel, ok := s.channelAccess(w, r, "admin")
	if !ok {
		return
	}

	var req channelRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	name := trimSpace(req.Name)
	if name == "" || len([]rune(name)) > 60 {
		writeError(w, http.StatusBadRequest, "渠道名称需要 1-60 个字符")
		return
	}

	config := req.Config
	if len(config) == 0 || string(config) == "null" {
		config = channel.Config
	}
	if err := monitor.ValidateChannelConfig(channel.Type, config); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	enabled := channel.Enabled
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	updated, err := s.store.UpdateChannel(r.Context(), channel.WorkspaceID, channel.ID, name, config, enabled)
	if err != nil {
		slog.Error("更新通知渠道失败", "err", err)
		writeError(w, http.StatusInternalServerError, "更新通知渠道失败")
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleDeleteChannel(w http.ResponseWriter, r *http.Request) {
	channel, ok := s.channelAccess(w, r, "admin")
	if !ok {
		return
	}

	if err := s.store.DeleteChannel(r.Context(), channel.WorkspaceID, channel.ID); err != nil {
		slog.Error("删除通知渠道失败", "err", err)
		writeError(w, http.StatusInternalServerError, "删除通知渠道失败")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleTestChannel(w http.ResponseWriter, r *http.Request) {
	channel, ok := s.channelAccess(w, r, "admin")
	if !ok {
		return
	}

	workspaceName := ""
	if ws, err := s.store.GetWorkspace(r.Context(), channel.WorkspaceID); err == nil {
		workspaceName = ws.Name
	}

	sender := monitor.NewSender(s.mail)
	err := sender.Send(r.Context(), *channel, monitor.AlertMessage{
		Event:         "test",
		WorkspaceName: workspaceName,
		MonitorName:   "测试监控",
		MonitorType:   "http",
		Target:        "https://example.com",
		Status:        "up",
		LatencyMS:     42,
		CheckedAt:     time.Now().UTC(),
	})

	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

type incidentRequest struct {
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	Severity  string     `json:"severity"`
	MonitorID *uuid.UUID `json:"monitor_id"`
}

func (s *Server) handleListIncidents(w http.ResponseWriter, r *http.Request) {
	ws, _, ok := s.workspaceAccess(w, r, "member")
	if !ok {
		return
	}

	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}

	incidents, err := s.store.ListIncidents(r.Context(), ws.ID, limit)
	if err != nil {
		slog.Error("查询事件失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取事件失败")
		return
	}

	writeJSON(w, http.StatusOK, incidents)
}

func (s *Server) handleCreateIncident(w http.ResponseWriter, r *http.Request) {
	ws, _, ok := s.workspaceAccess(w, r, "admin")
	if !ok {
		return
	}

	var req incidentRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	title := trimSpace(req.Title)
	if title == "" || len([]rune(title)) > 200 {
		writeError(w, http.StatusBadRequest, "事件标题需要 1-200 个字符")
		return
	}

	severity := trimSpace(req.Severity)
	if severity == "" {
		severity = "major"
	}
	switch severity {
	case "minor", "major", "critical":
	default:
		writeError(w, http.StatusBadRequest, "事件级别不合法")
		return
	}

	if req.MonitorID != nil {
		if _, err := s.store.GetMonitor(r.Context(), ws.ID, *req.MonitorID); err != nil {
			writeError(w, http.StatusBadRequest, "关联的监控不存在")
			return
		}
	}

	incident, err := s.store.CreateIncident(
		r.Context(), ws.ID, req.MonitorID, title, trimSpace(req.Body), severity, "manual",
	)
	if err != nil {
		slog.Error("创建事件失败", "err", err)
		writeError(w, http.StatusInternalServerError, "创建事件失败")
		return
	}

	writeJSON(w, http.StatusCreated, incident)
}

func (s *Server) handleResolveIncident(w http.ResponseWriter, r *http.Request) {
	ws, _, ok := s.workspaceAccess(w, r, "admin")
	if !ok {
		return
	}

	incidentID, err := uuid.Parse(chi.URLParam(r, "incidentID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "事件不存在")
		return
	}

	if err := s.store.ResolveIncident(r.Context(), ws.ID, incidentID); err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "事件不存在")
			return
		}
		slog.Error("关闭事件失败", "err", err)
		writeError(w, http.StatusInternalServerError, "关闭事件失败")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type maintenanceRequest struct {
	Title      string      `json:"title"`
	MonitorIDs []uuid.UUID `json:"monitor_ids"`
	StartsAt   time.Time   `json:"starts_at"`
	EndsAt     time.Time   `json:"ends_at"`
}

func (s *Server) handleListMaintenance(w http.ResponseWriter, r *http.Request) {
	ws, _, ok := s.workspaceAccess(w, r, "member")
	if !ok {
		return
	}

	windows, err := s.store.ListMaintenanceWindows(r.Context(), ws.ID)
	if err != nil {
		slog.Error("查询维护窗口失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取维护窗口失败")
		return
	}

	writeJSON(w, http.StatusOK, windows)
}

func (s *Server) handleCreateMaintenance(w http.ResponseWriter, r *http.Request) {
	ws, _, ok := s.workspaceAccess(w, r, "admin")
	if !ok {
		return
	}

	var req maintenanceRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	title := trimSpace(req.Title)
	if title == "" || len([]rune(title)) > 120 {
		writeError(w, http.StatusBadRequest, "维护标题需要 1-120 个字符")
		return
	}

	if req.StartsAt.IsZero() || req.EndsAt.IsZero() || !req.StartsAt.Before(req.EndsAt) {
		writeError(w, http.StatusBadRequest, "维护时间范围不合法")
		return
	}

	for _, monitorID := range req.MonitorIDs {
		if _, err := s.store.GetMonitor(r.Context(), ws.ID, monitorID); err != nil {
			writeError(w, http.StatusBadRequest, "关联的监控不存在")
			return
		}
	}

	window, err := s.store.CreateMaintenanceWindow(
		r.Context(), ws.ID, title, req.MonitorIDs, req.StartsAt.UTC(), req.EndsAt.UTC(),
	)
	if err != nil {
		slog.Error("创建维护窗口失败", "err", err)
		writeError(w, http.StatusInternalServerError, "创建维护窗口失败")
		return
	}

	writeJSON(w, http.StatusCreated, window)
}

func (s *Server) handleDeleteMaintenance(w http.ResponseWriter, r *http.Request) {
	ws, _, ok := s.workspaceAccess(w, r, "admin")
	if !ok {
		return
	}

	windowID, err := uuid.Parse(chi.URLParam(r, "windowID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "维护窗口不存在")
		return
	}

	if err := s.store.DeleteMaintenanceWindow(r.Context(), ws.ID, windowID); err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "维护窗口不存在")
			return
		}
		slog.Error("删除维护窗口失败", "err", err)
		writeError(w, http.StatusInternalServerError, "删除维护窗口失败")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type muteRequest struct {
	Minutes int `json:"minutes"`
}

func (s *Server) handleMuteMonitor(w http.ResponseWriter, r *http.Request) {
	monitor, ok := s.monitorAccess(w, r, "admin")
	if !ok {
		return
	}

	var req muteRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if req.Minutes < 0 || req.Minutes > 10080 {
		writeError(w, http.StatusBadRequest, "静默时长需要在 0-10080 分钟之间")
		return
	}

	var until *time.Time
	if req.Minutes > 0 {
		value := time.Now().UTC().Add(time.Duration(req.Minutes) * time.Minute)
		until = &value
	}

	if err := s.store.SetMuted(r.Context(), monitor.ID, until); err != nil {
		slog.Error("设置静默失败", "err", err)
		writeError(w, http.StatusInternalServerError, "设置静默失败")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
