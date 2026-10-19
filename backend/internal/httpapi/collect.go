package httpapi

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"remoon.net/yu/backend/internal/analytics"
	"remoon.net/yu/backend/internal/auth"
	"remoon.net/yu/backend/internal/store"
)

type collectPayload struct {
	Token     string         `json:"t"`
	Type      string         `json:"type"`
	URL       string         `json:"url"`
	Referrer  string         `json:"ref"`
	Title     string         `json:"title"`
	Name      string         `json:"name"`
	Props     map[string]any `json:"props"`
	VisitorID string         `json:"vid"`
	SessionID string         `json:"sid"`
	Screen    string         `json:"screen"`
	Language  string         `json:"lang"`
}

func (s *Server) handleCollect(w http.ResponseWriter, r *http.Request) {
	// 采集接口允许任意站点跨域上报
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "no-store")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var payload collectPayload
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10)).Decode(&payload); err != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	payload.Token = strings.TrimSpace(payload.Token)
	if payload.Token == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if !s.collectLimiter.allow(payload.Token+"|"+clientIP(r), 120, time.Minute) {
		w.WriteHeader(http.StatusTooManyRequests)
		return
	}

	ctx := r.Context()

	site, err := s.store.GetSiteByToken(ctx, payload.Token)
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	eventType := strings.TrimSpace(payload.Type)
	if eventType == "" {
		eventType = "pageview"
	}
	switch eventType {
	case "pageview", "custom", "ping":
	default:
		w.WriteHeader(http.StatusNoContent)
		return
	}

	host, path := analytics.Page(payload.URL)
	if path == "" && eventType == "pageview" {
		if ref, err := url.Parse(r.Referer()); err == nil {
			path = ref.Path
		}
	}

	utmSource, utmMedium, utmCampaign, utmTerm, utmContent := analytics.UTM(payload.URL)
	referrerHost := analytics.ReferrerHost(payload.Referrer)
	channel := analytics.Channel(referrerHost, site.Domain, utmMedium)
	client := analytics.ParseUserAgent(r.UserAgent())

	ip := clientIP(r)
	if newTrafficFilter(site.FilterBots, site.ExcludeCIDRs).shouldSkip(ip, client.Device == "bot") {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	location := s.geo.Location(ip)

	props := encodeProps(payload.Props)

	visitorID := sanitizeID(payload.VisitorID)
	if visitorID == "" {
		visitorID, _ = auth.NewToken()
	}
	sessionID := sanitizeID(payload.SessionID)
	if sessionID == "" {
		sessionID = visitorID
	}

	input := store.EventInput{
		SiteID:         site.ID,
		Type:           eventType,
		Name:           truncate(payload.Name, 128),
		VisitorID:      visitorID,
		SessionID:      sessionID,
		URL:            truncate(payload.URL, 2048),
		Path:           truncate(path, 1024),
		Title:          truncate(payload.Title, 512),
		Referrer:       truncate(payload.Referrer, 2048),
		ReferrerHost:   referrerHost,
		UTMSource:      truncate(utmSource, 200),
		UTMMedium:      truncate(utmMedium, 200),
		UTMCampaign:    truncate(utmCampaign, 200),
		UTMTerm:        truncate(utmTerm, 200),
		UTMContent:     truncate(utmContent, 200),
		Channel:        channel,
		Browser:        client.Browser,
		BrowserVersion: client.BrowserVersion,
		OS:             client.OS,
		OSVersion:      client.OSVersion,
		Device:         client.Device,
		DeviceBrand:    client.DeviceBrand,
		Screen:         truncate(payload.Screen, 32),
		Language:       truncate(payload.Language, 32),
		Country:        location.Country,
		Region:         location.Region,
		City:           location.City,
		Props:          props,
		CreatedAt:      time.Now().UTC(),
	}

	// host 仅用于日志排查，统计中不使用
	_ = host

	if err := s.store.InsertEvent(ctx, input); err != nil {
		slog.Warn("写入统计事件失败", "err", err, "site", site.ID)
	}

	w.WriteHeader(http.StatusNoContent)
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func sanitizeID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" || len(id) > 64 {
		return ""
	}
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '-', r == '_', r == '.':
		default:
			return ""
		}
	}
	return id
}

func truncate(value string, max int) string {
	value = strings.TrimSpace(value)
	if utf8.RuneCountInString(value) <= max {
		return value
	}
	return string([]rune(value)[:max])
}

func encodeProps(props map[string]any) string {
	if len(props) == 0 {
		return ""
	}

	raw, err := json.Marshal(props)
	if err != nil || len(raw) > 4096 {
		return ""
	}

	return string(raw)
}
