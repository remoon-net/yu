package httpapi

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// rangeInfo 描述统计查询的时间范围。
type rangeInfo struct {
	From     time.Time
	To       time.Time
	Interval string
	Timezone string
}

func parseRange(r *http.Request, timezone string) (rangeInfo, bool) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
		timezone = "UTC"
	}

	now := time.Now().In(loc)
	to := now
	from := now.Add(-24 * time.Hour)

	if raw := r.URL.Query().Get("to"); raw != "" {
		parsed, err := parseTime(raw, loc)
		if err != nil {
			return rangeInfo{}, false
		}
		to = parsed
	}

	if raw := r.URL.Query().Get("from"); raw != "" {
		parsed, err := parseTime(raw, loc)
		if err != nil {
			return rangeInfo{}, false
		}
		from = parsed
	}

	if raw := r.URL.Query().Get("range"); raw != "" {
		span, err := parseRangeSpan(raw)
		if err != nil {
			return rangeInfo{}, false
		}
		from = to.Add(-span)
	}

	if !from.Before(to) {
		return rangeInfo{}, false
	}

	return rangeInfo{
		From:     from,
		To:       to,
		Interval: pickInterval(to.Sub(from)),
		Timezone: timezone,
	}, true
}

func parseTime(raw string, loc *time.Location) (time.Time, error) {
	raw = strings.TrimSpace(raw)

	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.In(loc), nil
	}

	return time.ParseInLocation("2006-01-02 15:04:05", raw, loc)
}

func parseRangeSpan(raw string) (time.Duration, error) {
	raw = strings.TrimSpace(raw)

	if strings.HasSuffix(raw, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(raw, "d"))
		if err != nil || days <= 0 {
			return 0, strconv.ErrSyntax
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}

	span, err := time.ParseDuration(raw)
	if err != nil || span <= 0 {
		return 0, strconv.ErrSyntax
	}

	return span, nil
}

func pickInterval(span time.Duration) string {
	switch {
	case span <= 48*time.Hour:
		return "hour"
	case span <= 90*24*time.Hour:
		return "day"
	case span <= 730*24*time.Hour:
		return "week"
	default:
		return "month"
	}
}

func (s *Server) handleStatsOverview(w http.ResponseWriter, r *http.Request) {
	site, ok := s.siteAccess(w, r, "member")
	if !ok {
		return
	}

	rn, ok := parseRange(r, site.Timezone)
	if !ok {
		writeError(w, http.StatusBadRequest, "时间范围不合法")
		return
	}

	ctx := r.Context()

	overview, err := s.store.Overview(ctx, site.ID, rn.From, rn.To)
	if err != nil {
		slog.Error("查询概览失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取统计数据失败")
		return
	}

	series, err := s.store.Series(ctx, site.ID, rn.From, rn.To, rn.Interval, rn.Timezone)
	if err != nil {
		slog.Error("查询趋势失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取统计数据失败")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"overview": overview,
		"series":   series,
		"range": map[string]any{
			"from":     rn.From,
			"to":       rn.To,
			"interval": rn.Interval,
			"timezone": rn.Timezone,
		},
	})
}

func (s *Server) handleStatsRealtime(w http.ResponseWriter, r *http.Request) {
	site, ok := s.siteAccess(w, r, "member")
	if !ok {
		return
	}

	stats, err := s.store.Realtime(r.Context(), site.ID)
	if err != nil {
		slog.Error("查询实时数据失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取实时数据失败")
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

func (s *Server) handleStatsPages(w http.ResponseWriter, r *http.Request) {
	site, ok := s.siteAccess(w, r, "member")
	if !ok {
		return
	}

	rn, ok := parseRange(r, site.Timezone)
	if !ok {
		writeError(w, http.StatusBadRequest, "时间范围不合法")
		return
	}

	ctx := r.Context()

	pages, err := s.store.TopPages(ctx, site.ID, rn.From, rn.To)
	if err != nil {
		slog.Error("查询页面统计失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取统计数据失败")
		return
	}

	entry, err := s.store.SessionPages(ctx, site.ID, rn.From, rn.To, true)
	if err != nil {
		slog.Error("查询入口页失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取统计数据失败")
		return
	}

	exit, err := s.store.SessionPages(ctx, site.ID, rn.From, rn.To, false)
	if err != nil {
		slog.Error("查询退出页失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取统计数据失败")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"pages": pages,
		"entry": entry,
		"exit":  exit,
	})
}

func (s *Server) handleStatsSources(w http.ResponseWriter, r *http.Request) {
	site, ok := s.siteAccess(w, r, "member")
	if !ok {
		return
	}

	rn, ok := parseRange(r, site.Timezone)
	if !ok {
		writeError(w, http.StatusBadRequest, "时间范围不合法")
		return
	}

	ctx := r.Context()

	referrers, err := s.store.Referrers(ctx, site.ID, rn.From, rn.To)
	if err != nil {
		slog.Error("查询来源失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取统计数据失败")
		return
	}

	channels, err := s.store.Channels(ctx, site.ID, rn.From, rn.To)
	if err != nil {
		slog.Error("查询渠道失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取统计数据失败")
		return
	}

	utm := make(map[string]any, 3)
	for _, field := range []string{"utm_source", "utm_medium", "utm_campaign"} {
		items, err := s.store.UTMStats(ctx, site.ID, rn.From, rn.To, field)
		if err != nil {
			slog.Error("查询 UTM 失败", "err", err, "field", field)
			writeError(w, http.StatusInternalServerError, "读取统计数据失败")
			return
		}
		utm[field] = items
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"referrers":    referrers,
		"channels":     channels,
		"utm_source":   utm["utm_source"],
		"utm_medium":   utm["utm_medium"],
		"utm_campaign": utm["utm_campaign"],
	})
}

func (s *Server) handleStatsAudience(w http.ResponseWriter, r *http.Request) {
	site, ok := s.siteAccess(w, r, "member")
	if !ok {
		return
	}

	rn, ok := parseRange(r, site.Timezone)
	if !ok {
		writeError(w, http.StatusBadRequest, "时间范围不合法")
		return
	}

	ctx := r.Context()
	out := make(map[string]any, 6)

	for _, field := range []string{"browser", "os", "device", "screen", "language", "country"} {
		items, err := s.store.Breakdown(ctx, site.ID, rn.From, rn.To, field)
		if err != nil {
			slog.Error("查询访客画像失败", "err", err, "field", field)
			writeError(w, http.StatusInternalServerError, "读取统计数据失败")
			return
		}
		out[field] = items
	}

	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleStatsEvents(w http.ResponseWriter, r *http.Request) {
	site, ok := s.siteAccess(w, r, "member")
	if !ok {
		return
	}

	rn, ok := parseRange(r, site.Timezone)
	if !ok {
		writeError(w, http.StatusBadRequest, "时间范围不合法")
		return
	}

	events, err := s.store.CustomEvents(r.Context(), site.ID, rn.From, rn.To)
	if err != nil {
		slog.Error("查询自定义事件失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取统计数据失败")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"events": events})
}

func (s *Server) handleStatsEventDetail(w http.ResponseWriter, r *http.Request) {
	site, ok := s.siteAccess(w, r, "member")
	if !ok {
		return
	}

	rn, ok := parseRange(r, site.Timezone)
	if !ok {
		writeError(w, http.StatusBadRequest, "时间范围不合法")
		return
	}

	name := chi.URLParam(r, "name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "事件名不能为空")
		return
	}

	ctx := r.Context()

	series, err := s.store.EventSeries(ctx, site.ID, rn.From, rn.To, name, rn.Interval, rn.Timezone)
	if err != nil {
		slog.Error("查询事件趋势失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取统计数据失败")
		return
	}

	props, err := s.store.EventProps(ctx, site.ID, rn.From, rn.To, name)
	if err != nil {
		slog.Error("查询事件属性失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取统计数据失败")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"name":   name,
		"series": series,
		"props":  props,
	})
}
