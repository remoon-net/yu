package httpapi

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"remoon.net/yu/backend/internal/store"
)

type goalRequest struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	MatchValue string `json:"match_value"`
}

func validateGoalRequest(req goalRequest) (goalRequest, string, bool) {
	name := trimSpace(req.Name)
	if name == "" || len([]rune(name)) > 80 {
		return req, "目标名称需要 1-80 个字符", false
	}

	goalType := strings.ToLower(trimSpace(req.Type))
	if goalType != "event" && goalType != "page" {
		return req, "目标类型只能是 event 或 page", false
	}

	value := trimSpace(req.MatchValue)
	if value == "" || len([]rune(value)) > 512 {
		return req, "目标匹配值需要 1-512 个字符", false
	}

	return goalRequest{Name: name, Type: goalType, MatchValue: value}, "", true
}

func (s *Server) handleListGoals(w http.ResponseWriter, r *http.Request) {
	site, ok := s.siteAccess(w, r, "member")
	if !ok {
		return
	}

	goals, err := s.store.ListGoals(r.Context(), site.ID)
	if err != nil {
		slog.Error("查询目标失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取目标失败")
		return
	}

	writeJSON(w, http.StatusOK, goals)
}

func (s *Server) handleCreateGoal(w http.ResponseWriter, r *http.Request) {
	site, ok := s.siteAccess(w, r, "admin")
	if !ok {
		return
	}

	var req goalRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req, message, ok := validateGoalRequest(req)
	if !ok {
		writeError(w, http.StatusBadRequest, message)
		return
	}

	goal, err := s.store.CreateGoal(r.Context(), store.GoalInput{
		SiteID:     site.ID,
		Name:       req.Name,
		Type:       req.Type,
		MatchValue: req.MatchValue,
	})
	if err != nil {
		slog.Error("创建目标失败", "err", err)
		writeError(w, http.StatusInternalServerError, "创建目标失败")
		return
	}

	writeJSON(w, http.StatusCreated, goal)
}

func (s *Server) handleUpdateGoal(w http.ResponseWriter, r *http.Request) {
	site, ok := s.siteAccess(w, r, "admin")
	if !ok {
		return
	}

	goalID, err := uuid.Parse(chi.URLParam(r, "goalID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "目标不存在")
		return
	}

	var req goalRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req, message, ok := validateGoalRequest(req)
	if !ok {
		writeError(w, http.StatusBadRequest, message)
		return
	}

	goal, err := s.store.UpdateGoal(r.Context(), goalID, store.GoalInput{
		SiteID:     site.ID,
		Name:       req.Name,
		Type:       req.Type,
		MatchValue: req.MatchValue,
	})
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "目标不存在")
			return
		}
		slog.Error("更新目标失败", "err", err)
		writeError(w, http.StatusInternalServerError, "更新目标失败")
		return
	}

	writeJSON(w, http.StatusOK, goal)
}

func (s *Server) handleDeleteGoal(w http.ResponseWriter, r *http.Request) {
	site, ok := s.siteAccess(w, r, "admin")
	if !ok {
		return
	}

	goalID, err := uuid.Parse(chi.URLParam(r, "goalID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "目标不存在")
		return
	}

	if err := s.store.DeleteGoal(r.Context(), site.ID, goalID); err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "目标不存在")
			return
		}
		slog.Error("删除目标失败", "err", err)
		writeError(w, http.StatusInternalServerError, "删除目标失败")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleGoalStats(w http.ResponseWriter, r *http.Request) {
	site, ok := s.siteAccess(w, r, "member")
	if !ok {
		return
	}

	rn, ok := parseRange(r, site.Timezone)
	if !ok {
		writeError(w, http.StatusBadRequest, "时间范围不合法")
		return
	}

	stats, err := s.store.GoalStats(r.Context(), site.ID, rn.From, rn.To, statsFilter(r))
	if err != nil {
		slog.Error("查询目标统计失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取目标统计失败")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"goals": stats,
		"range": rangePayload(rn),
	})
}

type funnelRequest struct {
	Steps []store.FunnelStep `json:"steps"`
}

type funnelStepResult struct {
	Type           string  `json:"type"`
	Value          string  `json:"value"`
	Visitors       int64   `json:"visitors"`
	ConversionRate float64 `json:"conversion_rate"`
}

func (s *Server) handleStatsFunnel(w http.ResponseWriter, r *http.Request) {
	site, ok := s.siteAccess(w, r, "member")
	if !ok {
		return
	}

	rn, ok := parseRange(r, site.Timezone)
	if !ok {
		writeError(w, http.StatusBadRequest, "时间范围不合法")
		return
	}

	var req funnelRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if len(req.Steps) < 2 || len(req.Steps) > 5 {
		writeError(w, http.StatusBadRequest, "漏斗需要 2-5 个步骤")
		return
	}

	steps := make([]store.FunnelStep, 0, len(req.Steps))
	for _, step := range req.Steps {
		stepType := strings.ToLower(trimSpace(step.Type))
		stepValue := trimSpace(step.Value)
		if (stepType != "event" && stepType != "page") || stepValue == "" || len([]rune(stepValue)) > 512 {
			writeError(w, http.StatusBadRequest, "漏斗步骤不合法")
			return
		}
		steps = append(steps, store.FunnelStep{Type: stepType, Value: stepValue})
	}

	counts, err := s.store.FunnelVisitors(r.Context(), site.ID, rn.From, rn.To, steps, statsFilter(r))
	if err != nil {
		slog.Error("查询漏斗失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取漏斗数据失败")
		return
	}

	results := make([]funnelStepResult, 0, len(steps))
	base := int64(0)
	if len(counts) > 0 {
		base = counts[0]
	}
	for i, step := range steps {
		rate := float64(0)
		if base > 0 {
			rate = float64(counts[i]) / float64(base) * 100
		}
		results = append(results, funnelStepResult{
			Type:           step.Type,
			Value:          step.Value,
			Visitors:       counts[i],
			ConversionRate: rate,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"steps": results,
		"range": rangePayload(rn),
	})
}
