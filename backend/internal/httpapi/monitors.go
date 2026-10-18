package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"remoon.net/yu/backend/internal/monitor/probe"
	"remoon.net/yu/backend/internal/store"
)

type monitorRequest struct {
	Name         string          `json:"name"`
	Type         string          `json:"type"`
	Config       json.RawMessage `json:"config"`
	IntervalSec  int             `json:"interval_sec"`
	TimeoutMS    int             `json:"timeout_ms"`
	Retries      int             `json:"retries"`
	GroupName    string          `json:"group_name"`
	Tags         []string        `json:"tags"`
	SSLAlertDays int             `json:"ssl_alert_days"`
	Enabled      *bool           `json:"enabled"`
}

// monitorAccess 校验工作区权限并读取路径中的监控。
func (s *Server) monitorAccess(w http.ResponseWriter, r *http.Request, minRole string) (*store.Monitor, bool) {
	ws, _, ok := s.workspaceAccess(w, r, minRole)
	if !ok {
		return nil, false
	}

	monitorID, err := uuid.Parse(chi.URLParam(r, "monitorID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "监控不存在")
		return nil, false
	}

	monitor, err := s.store.GetMonitor(r.Context(), ws.ID, monitorID)
	if err != nil {
		writeError(w, http.StatusNotFound, "监控不存在")
		return nil, false
	}

	return monitor, true
}

func (s *Server) handleListMonitors(w http.ResponseWriter, r *http.Request) {
	ws, _, ok := s.workspaceAccess(w, r, "member")
	if !ok {
		return
	}

	monitors, err := s.store.ListMonitors(r.Context(), ws.ID)
	if err != nil {
		slog.Error("查询监控失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取监控失败")
		return
	}

	writeJSON(w, http.StatusOK, monitors)
}

func (s *Server) handleCreateMonitor(w http.ResponseWriter, r *http.Request) {
	ws, _, ok := s.workspaceAccess(w, r, "admin")
	if !ok {
		return
	}

	var req monitorRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	input, message, ok := buildMonitorInput(ws.ID, req)
	if !ok {
		writeError(w, http.StatusBadRequest, message)
		return
	}

	monitor, err := s.store.CreateMonitor(r.Context(), input)
	if err != nil {
		slog.Error("创建监控失败", "err", err)
		writeError(w, http.StatusInternalServerError, "创建监控失败")
		return
	}

	writeJSON(w, http.StatusCreated, monitor)
}

func (s *Server) handleGetMonitor(w http.ResponseWriter, r *http.Request) {
	monitor, ok := s.monitorAccess(w, r, "member")
	if !ok {
		return
	}

	writeJSON(w, http.StatusOK, monitor)
}

func (s *Server) handleUpdateMonitor(w http.ResponseWriter, r *http.Request) {
	monitor, ok := s.monitorAccess(w, r, "admin")
	if !ok {
		return
	}

	var req monitorRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	input, message, ok := buildMonitorInput(monitor.WorkspaceID, req)
	if !ok {
		writeError(w, http.StatusBadRequest, message)
		return
	}

	updated, err := s.store.UpdateMonitor(r.Context(), monitor.ID, input)
	if err != nil {
		slog.Error("更新监控失败", "err", err)
		writeError(w, http.StatusInternalServerError, "更新监控失败")
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleDeleteMonitor(w http.ResponseWriter, r *http.Request) {
	monitor, ok := s.monitorAccess(w, r, "admin")
	if !ok {
		return
	}

	if err := s.store.DeleteMonitor(r.Context(), monitor.WorkspaceID, monitor.ID); err != nil {
		slog.Error("删除监控失败", "err", err)
		writeError(w, http.StatusInternalServerError, "删除监控失败")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMonitorChecks(w http.ResponseWriter, r *http.Request) {
	monitor, ok := s.monitorAccess(w, r, "member")
	if !ok {
		return
	}

	rn, ok := parseRange(r, "UTC")
	if !ok {
		writeError(w, http.StatusBadRequest, "时间范围不合法")
		return
	}

	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	if limit < 1 {
		limit = 1
	}
	if limit > 500 {
		limit = 500
	}

	checks, err := s.store.ListChecks(r.Context(), monitor.ID, rn.From, rn.To, limit)
	if err != nil {
		slog.Error("查询探测记录失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取探测记录失败")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"checks": checks,
		"range":  rangePayload(rn),
	})
}

func (s *Server) handleMonitorStats(w http.ResponseWriter, r *http.Request) {
	monitor, ok := s.monitorAccess(w, r, "member")
	if !ok {
		return
	}

	rn, ok := parseRange(r, "UTC")
	if !ok {
		writeError(w, http.StatusBadRequest, "时间范围不合法")
		return
	}

	stats, err := s.store.MonitorStats(r.Context(), monitor.ID, rn.From, rn.To, rn.Interval)
	if err != nil {
		slog.Error("查询监控统计失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取监控统计失败")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"stats": stats,
		"range": rangePayload(rn),
	})
}

func rangePayload(rn rangeInfo) map[string]any {
	return map[string]any{
		"from":     rn.From,
		"to":       rn.To,
		"interval": rn.Interval,
	}
}

func buildMonitorInput(workspaceID uuid.UUID, req monitorRequest) (store.MonitorInput, string, bool) {
	name := trimSpace(req.Name)
	if name == "" || len([]rune(name)) > 80 {
		return store.MonitorInput{}, "监控名称需要 1-80 个字符", false
	}

	monitorType := strings.ToLower(trimSpace(req.Type))
	switch monitorType {
	case "http", "tcp", "ping", "dns":
	default:
		return store.MonitorInput{}, "监控类型不合法", false
	}

	config := req.Config
	if len(config) == 0 || string(config) == "null" {
		config = json.RawMessage(`{}`)
	}
	if err := probe.Validate(monitorType, config); err != nil {
		return store.MonitorInput{}, err.Error(), false
	}

	interval := req.IntervalSec
	if interval == 0 {
		interval = 60
	}
	if interval < 10 || interval > 86400 {
		return store.MonitorInput{}, "检查间隔需要在 10-86400 秒之间", false
	}

	timeout := req.TimeoutMS
	if timeout == 0 {
		timeout = 10000
	}
	if timeout < 500 || timeout > 60000 {
		return store.MonitorInput{}, "超时需要在 500-60000 毫秒之间", false
	}

	if req.Retries < 0 || req.Retries > 5 {
		return store.MonitorInput{}, "重试次数需要在 0-5 之间", false
	}

	sslDays := req.SSLAlertDays
	if sslDays == 0 {
		sslDays = 14
	}
	if sslDays < 0 || sslDays > 365 {
		return store.MonitorInput{}, "证书提醒天数需要在 0-365 之间", false
	}

	tags := make([]string, 0, len(req.Tags))
	seen := make(map[string]bool, len(req.Tags))
	for _, tag := range req.Tags {
		tag = trimSpace(tag)
		if tag == "" || len([]rune(tag)) > 30 || seen[tag] {
			continue
		}
		seen[tag] = true
		tags = append(tags, tag)
		if len(tags) >= 20 {
			break
		}
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	return store.MonitorInput{
		WorkspaceID:  workspaceID,
		Name:         name,
		Type:         monitorType,
		Config:       config,
		IntervalSec:  interval,
		TimeoutMS:    timeout,
		Retries:      req.Retries,
		GroupName:    trimSpace(req.GroupName),
		Tags:         tags,
		SSLAlertDays: sslDays,
		Enabled:      enabled,
	}, "", true
}
