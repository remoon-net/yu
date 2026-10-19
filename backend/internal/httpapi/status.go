package httpapi

import (
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"remoon.net/yu/backend/internal/auth"
	"remoon.net/yu/backend/internal/store"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,48}[a-z0-9]$`)

type statusPageRequest struct {
	Slug          string `json:"slug"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Theme         string `json:"theme"`
	Password      string `json:"password"`
	ClearPassword bool   `json:"clear_password"`
}

func (s *Server) statusPageAccess(w http.ResponseWriter, r *http.Request, minRole string) (*store.StatusPage, bool) {
	ws, _, ok := s.workspaceAccess(w, r, minRole)
	if !ok {
		return nil, false
	}

	pageID, err := uuid.Parse(chi.URLParam(r, "pageID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "状态页不存在")
		return nil, false
	}

	page, err := s.store.GetStatusPage(r.Context(), ws.ID, pageID)
	if err != nil {
		writeError(w, http.StatusNotFound, "状态页不存在")
		return nil, false
	}

	return page, true
}

func (s *Server) handleListStatusPages(w http.ResponseWriter, r *http.Request) {
	ws, _, ok := s.workspaceAccess(w, r, "member")
	if !ok {
		return
	}

	pages, err := s.store.ListStatusPages(r.Context(), ws.ID)
	if err != nil {
		slog.Error("查询状态页失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取状态页失败")
		return
	}

	writeJSON(w, http.StatusOK, pages)
}

func (s *Server) handleCreateStatusPage(w http.ResponseWriter, r *http.Request) {
	ws, _, ok := s.workspaceAccess(w, r, "admin")
	if !ok {
		return
	}

	var req statusPageRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	name := trimSpace(req.Name)
	if name == "" || len([]rune(name)) > 80 {
		writeError(w, http.StatusBadRequest, "状态页名称需要 1-80 个字符")
		return
	}

	slug := strings.ToLower(trimSpace(req.Slug))
	if !slugPattern.MatchString(slug) {
		writeError(w, http.StatusBadRequest, "slug 只能包含小写字母、数字与连字符，长度 2-50")
		return
	}

	theme := trimSpace(req.Theme)
	if theme != "light" {
		theme = "dark"
	}

	var passwordHash *string
	if req.Password != "" {
		hash, err := auth.HashPassword(req.Password)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		passwordHash = &hash
	}

	page, err := s.store.CreateStatusPage(
		r.Context(), ws.ID, slug, name, trimSpace(req.Description), passwordHash, theme,
	)
	if err != nil {
		if store.IsUniqueViolation(err) {
			writeError(w, http.StatusConflict, "slug 已被占用")
			return
		}
		slog.Error("创建状态页失败", "err", err)
		writeError(w, http.StatusInternalServerError, "创建状态页失败")
		return
	}

	writeJSON(w, http.StatusCreated, page)
}

func (s *Server) handleGetStatusPage(w http.ResponseWriter, r *http.Request) {
	page, ok := s.statusPageAccess(w, r, "member")
	if !ok {
		return
	}

	items, err := s.store.ListStatusPageItems(r.Context(), page.ID)
	if err != nil {
		slog.Error("查询状态页组件失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取状态页失败")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"page": page, "items": items})
}

func (s *Server) handleUpdateStatusPage(w http.ResponseWriter, r *http.Request) {
	page, ok := s.statusPageAccess(w, r, "admin")
	if !ok {
		return
	}

	var req statusPageRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	name := trimSpace(req.Name)
	if name == "" || len([]rune(name)) > 80 {
		writeError(w, http.StatusBadRequest, "状态页名称需要 1-80 个字符")
		return
	}

	theme := trimSpace(req.Theme)
	if theme != "light" {
		theme = "dark"
	}

	var passwordHash *string
	if req.Password != "" {
		hash, err := auth.HashPassword(req.Password)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		passwordHash = &hash
	}

	updated, err := s.store.UpdateStatusPage(
		r.Context(), page.WorkspaceID, page.ID,
		name, trimSpace(req.Description), theme, req.ClearPassword, passwordHash,
	)
	if err != nil {
		slog.Error("更新状态页失败", "err", err)
		writeError(w, http.StatusInternalServerError, "更新状态页失败")
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleDeleteStatusPage(w http.ResponseWriter, r *http.Request) {
	page, ok := s.statusPageAccess(w, r, "admin")
	if !ok {
		return
	}

	if err := s.store.DeleteStatusPage(r.Context(), page.WorkspaceID, page.ID); err != nil {
		slog.Error("删除状态页失败", "err", err)
		writeError(w, http.StatusInternalServerError, "删除状态页失败")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type statusItemInput struct {
	MonitorID   uuid.UUID `json:"monitor_id"`
	DisplayName string    `json:"display_name"`
	GroupName   string    `json:"group_name"`
}

func (s *Server) handlePutStatusPageItems(w http.ResponseWriter, r *http.Request) {
	page, ok := s.statusPageAccess(w, r, "admin")
	if !ok {
		return
	}

	var req struct {
		Items []statusItemInput `json:"items"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	if len(req.Items) > 100 {
		writeError(w, http.StatusBadRequest, "最多支持 100 个组件")
		return
	}

	items := make([]store.StatusPageItem, 0, len(req.Items))
	for _, item := range req.Items {
		if _, err := s.store.GetMonitor(r.Context(), page.WorkspaceID, item.MonitorID); err != nil {
			writeError(w, http.StatusBadRequest, "关联的监控不存在")
			return
		}

		items = append(items, store.StatusPageItem{
			MonitorID:   item.MonitorID,
			DisplayName: truncate(item.DisplayName, 60),
			GroupName:   truncate(item.GroupName, 60),
		})
	}

	if err := s.store.ReplaceStatusPageItems(r.Context(), page.WorkspaceID, page.ID, items); err != nil {
		slog.Error("保存状态页组件失败", "err", err)
		writeError(w, http.StatusInternalServerError, "保存状态页组件失败")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleSetSharePassword(w http.ResponseWriter, r *http.Request) {
	site, ok := s.siteAccess(w, r, "admin")
	if !ok {
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	var passwordHash *string
	if req.Password != "" {
		hash, err := auth.HashPassword(req.Password)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		passwordHash = &hash
	}

	if err := s.store.SetSiteSharePassword(r.Context(), site.WorkspaceID, site.ID, passwordHash); err != nil {
		slog.Error("设置分享密码失败", "err", err)
		writeError(w, http.StatusInternalServerError, "设置分享密码失败")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handlePublicStatus 返回公开状态页数据。
func (s *Server) handlePublicStatus(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	page, passwordHash, err := s.store.GetStatusPageBySlug(r.Context(), slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "状态页不存在")
		return
	}

	if passwordHash != "" && !auth.VerifyPassword(passwordHash, r.URL.Query().Get("password")) {
		writeError(w, http.StatusUnauthorized, "需要访问密码")
		return
	}

	items, err := s.store.ListStatusPageItemsView(r.Context(), page.ID)
	if err != nil {
		slog.Error("查询状态页组件失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取状态页失败")
		return
	}

	ids := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.MonitorID)
	}

	uptimes, err := s.store.MonitorUptime24h(r.Context(), ids)
	if err != nil {
		slog.Warn("查询可用率失败", "err", err)
	}
	for index := range items {
		items[index].Uptime24h = uptimes[items[index].MonitorID]
	}

	incidents, err := s.store.ListIncidents(r.Context(), page.WorkspaceID, 10)
	if err != nil {
		slog.Warn("查询状态页事件失败", "err", err)
	}

	windows, err := s.store.ListMaintenanceWindows(r.Context(), page.WorkspaceID)
	if err != nil {
		slog.Warn("查询状态页维护失败", "err", err)
	}

	overall := "operational"
	for _, item := range items {
		if item.Status == "down" {
			overall = "down"
			break
		}
		if item.Status == "" {
			overall = "unknown"
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"page":        page,
		"overall":     overall,
		"items":       items,
		"incidents":   incidents,
		"maintenance": windows,
		"updated_at":  time.Now().UTC(),
	})
}

// handlePublicShare 返回公开统计看板数据。
func (s *Server) handlePublicShare(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")

	site, err := s.store.GetSiteByShareToken(r.Context(), token)
	if err != nil {
		writeError(w, http.StatusNotFound, "分享链接不存在")
		return
	}

	passwordHash, err := s.store.GetSiteSharePassword(r.Context(), site.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取分享信息失败")
		return
	}

	if passwordHash != "" && !auth.VerifyPassword(passwordHash, r.URL.Query().Get("password")) {
		writeError(w, http.StatusUnauthorized, "需要访问密码")
		return
	}

	rn, ok := parseRange(r, site.Timezone)
	if !ok {
		writeError(w, http.StatusBadRequest, "时间范围不合法")
		return
	}

	ctx := r.Context()

	overview, err := s.store.Overview(ctx, site.ID, rn.From, rn.To, store.StatsFilter{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取统计数据失败")
		return
	}

	series, err := s.store.Series(ctx, site.ID, rn.From, rn.To, rn.Interval, rn.Timezone, store.StatsFilter{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取统计数据失败")
		return
	}

	pages, err := s.store.TopPages(ctx, site.ID, rn.From, rn.To, store.StatsFilter{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取统计数据失败")
		return
	}

	referrers, err := s.store.Referrers(ctx, site.ID, rn.From, rn.To, store.StatsFilter{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取统计数据失败")
		return
	}

	channels, err := s.store.Channels(ctx, site.ID, rn.From, rn.To, store.StatsFilter{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取统计数据失败")
		return
	}

	devices, err := s.store.Breakdown(ctx, site.ID, rn.From, rn.To, "device", store.StatsFilter{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取统计数据失败")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"site": map[string]any{
			"name":     site.Name,
			"domain":   site.Domain,
			"timezone": site.Timezone,
		},
		"range":     rangePayload(rn),
		"overview":  overview,
		"series":    series,
		"pages":     pages,
		"referrers": referrers,
		"channels":  channels,
		"devices":   devices,
	})
}
