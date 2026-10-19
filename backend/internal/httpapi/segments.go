package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"remoon.net/yu/backend/internal/store"
)

type segmentRequest struct {
	Name    string            `json:"name"`
	Filters store.StatsFilter `json:"filters"`
}

func normalizeStatsFilter(f store.StatsFilter) store.StatsFilter {
	return store.StatsFilter{
		Path:         trimSpace(f.Path),
		ReferrerHost: trimSpace(f.ReferrerHost),
		Country:      trimSpace(f.Country),
		Device:       trimSpace(f.Device),
		Browser:      trimSpace(f.Browser),
		UTMSource:    trimSpace(f.UTMSource),
	}
}

func validStatsFilter(f store.StatsFilter) bool {
	for _, v := range []string{f.Path, f.ReferrerHost, f.Country, f.Device, f.Browser, f.UTMSource} {
		if len([]rune(v)) > 256 {
			return false
		}
	}
	return true
}

func (s *Server) handleListSegments(w http.ResponseWriter, r *http.Request) {
	site, ok := s.siteAccess(w, r, "member")
	if !ok {
		return
	}

	segments, err := s.store.ListSegments(r.Context(), site.ID)
	if err != nil {
		slog.Error("查询分段失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取分段失败")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"segments": segments})
}

func (s *Server) handleCreateSegment(w http.ResponseWriter, r *http.Request) {
	site, ok := s.siteAccess(w, r, "member")
	if !ok {
		return
	}

	var req segmentRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求内容不合法")
		return
	}

	req.Name = trimSpace(req.Name)
	req.Filters = normalizeStatsFilter(req.Filters)
	if req.Name == "" || len([]rune(req.Name)) > 80 || !validStatsFilter(req.Filters) {
		writeError(w, http.StatusBadRequest, "分段名称或筛选条件不合法")
		return
	}

	segment, err := s.store.CreateSegment(r.Context(), site.ID, store.SegmentInput{
		Name:    req.Name,
		Filters: req.Filters,
	})
	if err != nil {
		slog.Error("创建分段失败", "err", err)
		writeError(w, http.StatusInternalServerError, "创建分段失败")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"segment": segment})
}

func (s *Server) handleUpdateSegment(w http.ResponseWriter, r *http.Request) {
	site, ok := s.siteAccess(w, r, "member")
	if !ok {
		return
	}

	segmentID, err := uuid.Parse(chi.URLParam(r, "segmentID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "分段 ID 不合法")
		return
	}

	var req segmentRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求内容不合法")
		return
	}

	req.Name = trimSpace(req.Name)
	req.Filters = normalizeStatsFilter(req.Filters)
	if req.Name == "" || len([]rune(req.Name)) > 80 || !validStatsFilter(req.Filters) {
		writeError(w, http.StatusBadRequest, "分段名称或筛选条件不合法")
		return
	}

	segment, err := s.store.UpdateSegment(r.Context(), site.ID, segmentID, store.SegmentInput{
		Name:    req.Name,
		Filters: req.Filters,
	})
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "分段不存在")
		return
	}
	if err != nil {
		slog.Error("更新分段失败", "err", err)
		writeError(w, http.StatusInternalServerError, "更新分段失败")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"segment": segment})
}

func (s *Server) handleDeleteSegment(w http.ResponseWriter, r *http.Request) {
	site, ok := s.siteAccess(w, r, "member")
	if !ok {
		return
	}

	segmentID, err := uuid.Parse(chi.URLParam(r, "segmentID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "分段 ID 不合法")
		return
	}

	if err := s.store.DeleteSegment(r.Context(), site.ID, segmentID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "分段不存在")
			return
		}
		slog.Error("删除分段失败", "err", err)
		writeError(w, http.StatusInternalServerError, "删除分段失败")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
