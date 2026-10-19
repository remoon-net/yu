package httpapi

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"remoon.net/yu/backend/internal/store"
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

func statsFilter(r *http.Request) store.StatsFilter {
	q := r.URL.Query()
	return store.StatsFilter{
		Path:         trimSpace(q.Get("path")),
		ReferrerHost: trimSpace(q.Get("referrer_host")),
		Country:      trimSpace(q.Get("country")),
		Device:       trimSpace(q.Get("device")),
		Browser:      trimSpace(q.Get("browser")),
		UTMSource:    trimSpace(q.Get("utm_source")),
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
	filter := statsFilter(r)

	overview, err := s.store.Overview(ctx, site.ID, rn.From, rn.To, filter)
	if err != nil {
		slog.Error("查询概览失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取统计数据失败")
		return
	}

	// 上期：与当前区间等长、紧邻在前的一段，用于环比。
	span := rn.To.Sub(rn.From)
	previous, err := s.store.Overview(ctx, site.ID, rn.From.Add(-span), rn.From, filter)
	if err != nil {
		slog.Warn("查询上期概览失败", "err", err)
		previous = store.StatsOverview{}
	}

	series, err := s.store.Series(ctx, site.ID, rn.From, rn.To, rn.Interval, rn.Timezone, filter)
	if err != nil {
		slog.Error("查询趋势失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取统计数据失败")
		return
	}

	buckets, err := s.store.DurationBuckets(ctx, site.ID, rn.From, rn.To, filter)
	if err != nil {
		slog.Error("查询会话时长分布失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取统计数据失败")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"overview":         overview,
		"previous":         previous,
		"series":           series,
		"duration_buckets": buckets,
		"range": map[string]any{
			"from":     rn.From,
			"to":       rn.To,
			"interval": rn.Interval,
			"timezone": rn.Timezone,
		},
	})
}

func (s *Server) handleWorkspaceStatsOverview(w http.ResponseWriter, r *http.Request) {
	ws, _, ok := s.workspaceAccess(w, r, "member")
	if !ok {
		return
	}

	timezone := strings.TrimSpace(r.URL.Query().Get("timezone"))
	if timezone == "" {
		timezone = "UTC"
	}

	rn, ok := parseRange(r, timezone)
	if !ok {
		writeError(w, http.StatusBadRequest, "时间范围不合法")
		return
	}

	ctx := r.Context()

	overview, err := s.store.WorkspaceOverview(ctx, ws.ID, rn.From, rn.To)
	if err != nil {
		slog.Error("查询工作区概览失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取统计数据失败")
		return
	}

	span := rn.To.Sub(rn.From)
	previous, err := s.store.WorkspaceOverview(ctx, ws.ID, rn.From.Add(-span), rn.From)
	if err != nil {
		slog.Warn("查询工作区上期概览失败", "err", err)
		previous = store.StatsOverview{}
	}

	series, err := s.store.WorkspaceSeries(ctx, ws.ID, rn.From, rn.To, rn.Interval, rn.Timezone)
	if err != nil {
		slog.Error("查询工作区趋势失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取统计数据失败")
		return
	}

	sites, err := s.store.WorkspaceSiteStats(ctx, ws.ID, rn.From, rn.To)
	if err != nil {
		slog.Error("查询站点概览失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取统计数据失败")
		return
	}

	buckets, err := s.store.WorkspaceDurationBuckets(ctx, ws.ID, rn.From, rn.To)
	if err != nil {
		slog.Error("查询会话时长分布失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取统计数据失败")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"overview":         overview,
		"previous":         previous,
		"series":           series,
		"sites":            sites,
		"duration_buckets": buckets,
		"range": map[string]any{
			"from":     rn.From,
			"to":       rn.To,
			"interval": rn.Interval,
			"timezone": rn.Timezone,
		},
	})
}

func (s *Server) handleStatsRetention(w http.ResponseWriter, r *http.Request) {
	site, ok := s.siteAccess(w, r, "member")
	if !ok {
		return
	}

	rn, ok := parseRange(r, site.Timezone)
	if !ok {
		writeError(w, http.StatusBadRequest, "时间范围不合法")
		return
	}

	interval := strings.ToLower(trimSpace(r.URL.Query().Get("interval")))
	if interval != "week" {
		interval = "day"
	}

	result, err := s.store.Retention(r.Context(), site.ID, rn.From, rn.To, interval, rn.Timezone, statsFilter(r))
	if err != nil {
		slog.Error("查询留存失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取留存数据失败")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"interval": result.Interval,
		"periods":  result.Periods,
		"cohorts":  result.Cohorts,
		"range":    rangePayload(rn),
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
	filter := statsFilter(r)

	pages, err := s.store.TopPages(ctx, site.ID, rn.From, rn.To, filter)
	if err != nil {
		slog.Error("查询页面统计失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取统计数据失败")
		return
	}

	entry, err := s.store.SessionPages(ctx, site.ID, rn.From, rn.To, true, filter)
	if err != nil {
		slog.Error("查询入口页失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取统计数据失败")
		return
	}

	exit, err := s.store.SessionPages(ctx, site.ID, rn.From, rn.To, false, filter)
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
	filter := statsFilter(r)

	referrers, err := s.store.Referrers(ctx, site.ID, rn.From, rn.To, filter)
	if err != nil {
		slog.Error("查询来源失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取统计数据失败")
		return
	}

	referrerURLs, err := s.store.ReferrerURLs(ctx, site.ID, rn.From, rn.To, filter)
	if err != nil {
		slog.Error("查询来源页面失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取统计数据失败")
		return
	}

	channels, err := s.store.Channels(ctx, site.ID, rn.From, rn.To, filter)
	if err != nil {
		slog.Error("查询渠道失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取统计数据失败")
		return
	}

	utm := make(map[string]any, 5)
	for _, field := range []string{"utm_source", "utm_medium", "utm_campaign", "utm_term", "utm_content"} {
		items, err := s.store.UTMStats(ctx, site.ID, rn.From, rn.To, field, filter)
		if err != nil {
			slog.Error("查询 UTM 失败", "err", err, "field", field)
			writeError(w, http.StatusInternalServerError, "读取统计数据失败")
			return
		}
		utm[field] = items
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"referrers":     referrers,
		"referrer_urls": referrerURLs,
		"channels":      channels,
		"utm_source":    utm["utm_source"],
		"utm_medium":    utm["utm_medium"],
		"utm_campaign":  utm["utm_campaign"],
		"utm_term":      utm["utm_term"],
		"utm_content":   utm["utm_content"],
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
	filter := statsFilter(r)
	out := make(map[string]any, 11)

	for _, field := range []string{"browser", "browser_version", "os", "os_version", "device", "device_brand", "screen", "language", "country", "region", "city"} {
		items, err := s.store.Breakdown(ctx, site.ID, rn.From, rn.To, field, filter)
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

	events, err := s.store.CustomEvents(r.Context(), site.ID, rn.From, rn.To, statsFilter(r))
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
	filter := statsFilter(r)

	series, err := s.store.EventSeries(ctx, site.ID, rn.From, rn.To, name, rn.Interval, rn.Timezone, filter)
	if err != nil {
		slog.Error("查询事件趋势失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取统计数据失败")
		return
	}

	props, err := s.store.EventProps(ctx, site.ID, rn.From, rn.To, name, filter)
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
