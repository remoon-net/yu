package httpapi

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"remoon.net/yu/backend/internal/store"
)

type siteResponse struct {
	*store.Site
	ScriptURL      string `json:"script_url"`
	Endpoint       string `json:"endpoint"`
	ShareURL       string `json:"share_url"`
	InstallSnippet string `json:"install_snippet"`
}

func (s *Server) siteResponse(site *store.Site) siteResponse {
	base := s.cfg.PublicBaseURL

	return siteResponse{
		Site:           site,
		ScriptURL:      base + "/collect.js",
		Endpoint:       base + "/api/collect",
		ShareURL:       base + "/share/" + site.ShareToken + "/",
		InstallSnippet: `<script defer src="` + base + `/collect.js" data-token="` + site.Token + `"></script>`,
	}
}

// siteAccess 校验工作区权限并读取路径中的站点。
func (s *Server) siteAccess(w http.ResponseWriter, r *http.Request, minRole string) (*store.Site, bool) {
	ws, _, ok := s.workspaceAccess(w, r, minRole)
	if !ok {
		return nil, false
	}

	siteID, err := uuid.Parse(chi.URLParam(r, "siteID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "站点不存在")
		return nil, false
	}

	site, err := s.store.GetSite(r.Context(), ws.ID, siteID)
	if err != nil {
		writeError(w, http.StatusNotFound, "站点不存在")
		return nil, false
	}

	return site, true
}

func (s *Server) handleListSites(w http.ResponseWriter, r *http.Request) {
	ws, _, ok := s.workspaceAccess(w, r, "member")
	if !ok {
		return
	}

	sites, err := s.store.ListSites(r.Context(), ws.ID)
	if err != nil {
		slog.Error("查询站点失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取站点失败")
		return
	}

	out := make([]siteResponse, 0, len(sites))
	for i := range sites {
		out = append(out, s.siteResponse(&sites[i]))
	}

	writeJSON(w, http.StatusOK, out)
}

type siteRequest struct {
	Name     string `json:"name"`
	Domain   string `json:"domain"`
	Timezone string `json:"timezone"`
}

func (s *Server) handleCreateSite(w http.ResponseWriter, r *http.Request) {
	ws, _, ok := s.workspaceAccess(w, r, "admin")
	if !ok {
		return
	}

	var req siteRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	name := trimSpace(req.Name)
	if name == "" || len([]rune(name)) > 60 {
		writeError(w, http.StatusBadRequest, "站点名称需要 1-60 个字符")
		return
	}

	timezone, ok := normalizeTimezone(req.Timezone)
	if !ok {
		writeError(w, http.StatusBadRequest, "时区不合法")
		return
	}

	site, err := s.store.CreateSite(r.Context(), ws.ID, name, req.Domain, timezone)
	if err != nil {
		slog.Error("创建站点失败", "err", err)
		writeError(w, http.StatusInternalServerError, "创建站点失败")
		return
	}

	writeJSON(w, http.StatusCreated, s.siteResponse(site))
}

func (s *Server) handleGetSite(w http.ResponseWriter, r *http.Request) {
	site, ok := s.siteAccess(w, r, "member")
	if !ok {
		return
	}

	writeJSON(w, http.StatusOK, s.siteResponse(site))
}

func (s *Server) handleUpdateSite(w http.ResponseWriter, r *http.Request) {
	site, ok := s.siteAccess(w, r, "admin")
	if !ok {
		return
	}

	var req siteRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	name := trimSpace(req.Name)
	if name == "" || len([]rune(name)) > 60 {
		writeError(w, http.StatusBadRequest, "站点名称需要 1-60 个字符")
		return
	}

	timezone, ok := normalizeTimezone(req.Timezone)
	if !ok {
		writeError(w, http.StatusBadRequest, "时区不合法")
		return
	}

	updated, err := s.store.UpdateSite(r.Context(), site.WorkspaceID, site.ID, name, req.Domain, timezone)
	if err != nil {
		slog.Error("更新站点失败", "err", err)
		writeError(w, http.StatusInternalServerError, "更新站点失败")
		return
	}

	writeJSON(w, http.StatusOK, s.siteResponse(updated))
}

func (s *Server) handleDeleteSite(w http.ResponseWriter, r *http.Request) {
	site, ok := s.siteAccess(w, r, "admin")
	if !ok {
		return
	}

	if err := s.store.DeleteSite(r.Context(), site.WorkspaceID, site.ID); err != nil {
		slog.Error("删除站点失败", "err", err)
		writeError(w, http.StatusInternalServerError, "删除站点失败")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func normalizeTimezone(tz string) (string, bool) {
	tz = strings.TrimSpace(tz)
	if tz == "" {
		return "UTC", true
	}
	if _, err := time.LoadLocation(tz); err != nil {
		return "", false
	}
	return tz, true
}
