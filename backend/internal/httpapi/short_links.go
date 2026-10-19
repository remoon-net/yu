package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"remoon.net/yu/backend/internal/analytics"
	"remoon.net/yu/backend/internal/store"
)

const shortLinkSlugAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

var shortLinkSlugPattern = workspaceSlugPattern

type shortLinkRequest struct {
	Slug      string     `json:"slug"`
	TargetURL string     `json:"target_url"`
	Title     string     `json:"title"`
	Enabled   *bool      `json:"enabled"`
	ExpiresAt *time.Time `json:"expires_at"`
}

func newShortLinkSlug() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "link-" + uuid.NewString()[:8]
	}

	out := make([]byte, len(buf))
	for i, b := range buf {
		out[i] = shortLinkSlugAlphabet[int(b)%len(shortLinkSlugAlphabet)]
	}
	return string(out)
}

func validateShortLinkTarget(raw string) (string, bool) {
	target := strings.TrimSpace(raw)
	parsed, err := url.Parse(target)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", false
	}
	return target, true
}

func (s *Server) handleListShortLinks(w http.ResponseWriter, r *http.Request) {
	site, ok := s.siteAccess(w, r, "member")
	if !ok {
		return
	}

	links, err := s.store.ListShortLinks(r.Context(), site.ID)
	if err != nil {
		slog.Error("查询短链失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取短链失败")
		return
	}

	stats, err := s.store.ShortLinkStats(r.Context(), site.ID)
	if err != nil {
		slog.Warn("查询短链统计失败", "err", err)
		stats = nil
	}

	writeJSON(w, http.StatusOK, map[string]any{"links": links, "stats": stats})
}

func (s *Server) handleCreateShortLink(w http.ResponseWriter, r *http.Request) {
	site, ok := s.siteAccess(w, r, "admin")
	if !ok {
		return
	}

	var req shortLinkRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	target, ok := validateShortLinkTarget(req.TargetURL)
	if !ok {
		writeError(w, http.StatusBadRequest, "目标地址必须是合法的 http(s) 链接")
		return
	}

	slug := strings.ToLower(strings.TrimSpace(req.Slug))
	if slug == "" {
		slug = newShortLinkSlug()
	} else if !shortLinkSlugPattern.MatchString(slug) {
		writeError(w, http.StatusBadRequest, "短链标识只能包含小写字母、数字与连字符，长度 2-40")
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	link, err := s.store.CreateShortLink(r.Context(), site.WorkspaceID, site.ID, store.ShortLinkInput{
		Slug:      slug,
		TargetURL: target,
		Title:     req.Title,
		Enabled:   enabled,
		ExpiresAt: req.ExpiresAt,
	})
	if err != nil {
		if errors.Is(err, store.ErrShortLinkSlugTaken) {
			writeError(w, http.StatusConflict, "短链标识已被占用")
			return
		}
		slog.Error("创建短链失败", "err", err)
		writeError(w, http.StatusInternalServerError, "创建短链失败")
		return
	}

	writeJSON(w, http.StatusCreated, link)
}

func (s *Server) handleUpdateShortLink(w http.ResponseWriter, r *http.Request) {
	site, ok := s.siteAccess(w, r, "admin")
	if !ok {
		return
	}

	linkID, err := uuid.Parse(chi.URLParam(r, "linkID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "短链不存在")
		return
	}

	var req shortLinkRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	target, ok := validateShortLinkTarget(req.TargetURL)
	if !ok {
		writeError(w, http.StatusBadRequest, "目标地址必须是合法的 http(s) 链接")
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	updated, err := s.store.UpdateShortLink(r.Context(), site.ID, linkID, store.ShortLinkInput{
		TargetURL: target,
		Title:     req.Title,
		Enabled:   enabled,
		ExpiresAt: req.ExpiresAt,
	})
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "短链不存在")
			return
		}
		slog.Error("更新短链失败", "err", err)
		writeError(w, http.StatusInternalServerError, "更新短链失败")
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleDeleteShortLink(w http.ResponseWriter, r *http.Request) {
	site, ok := s.siteAccess(w, r, "admin")
	if !ok {
		return
	}

	linkID, err := uuid.Parse(chi.URLParam(r, "linkID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "短链不存在")
		return
	}

	if err := s.store.DeleteShortLink(r.Context(), site.ID, linkID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "短链不存在")
			return
		}
		slog.Error("删除短链失败", "err", err)
		writeError(w, http.StatusInternalServerError, "删除短链失败")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleShortLinkStats 返回单条短链的点击趋势与分布。
func (s *Server) handleShortLinkStats(w http.ResponseWriter, r *http.Request) {
	site, ok := s.siteAccess(w, r, "member")
	if !ok {
		return
	}

	linkID, err := uuid.Parse(chi.URLParam(r, "linkID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "短链不存在")
		return
	}

	link, err := s.store.GetShortLink(r.Context(), site.ID, linkID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "短链不存在")
			return
		}
		slog.Error("查询短链失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取短链失败")
		return
	}

	rn, ok := parseRange(r, site.Timezone)
	if !ok {
		writeError(w, http.StatusBadRequest, "时间范围不合法")
		return
	}

	detail, err := s.store.ShortLinkDetailStats(
		r.Context(), site.ID, link.Slug, rn.From, rn.To, rn.Interval, rn.Timezone,
	)
	if err != nil {
		slog.Error("查询短链统计失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取短链统计失败")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"link": link,
		"range": map[string]any{
			"from":     rn.From,
			"to":       rn.To,
			"interval": rn.Interval,
			"timezone": rn.Timezone,
		},
		"stats": detail,
	})
}

// handleShortLinkRedirect 处理公开短链跳转，并异步记录点击。
func (s *Server) handleShortLinkRedirect(w http.ResponseWriter, r *http.Request) {
	slug := strings.ToLower(strings.TrimSpace(chi.URLParam(r, "slug")))
	if slug == "" {
		notFoundPage(w, "短链不存在")
		return
	}

	if !s.collectLimiter.allow("link|"+clientIP(r), 120, time.Minute) {
		w.WriteHeader(http.StatusTooManyRequests)
		return
	}

	link, _, err := s.store.GetShortLinkBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			notFoundPage(w, "短链不存在")
			return
		}
		slog.Error("查询短链失败", "err", err, "slug", slug)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !link.Enabled {
		gonePage(w, "该短链已停用")
		return
	}
	if link.ExpiresAt != nil && link.ExpiresAt.Before(time.Now()) {
		gonePage(w, "该短链已过期")
		return
	}

	target, err := url.Parse(link.TargetURL)
	if err != nil {
		slog.Error("短链目标地址无效", "err", err, "slug", slug)
		gonePage(w, "该短链配置有误")
		return
	}
	if r.URL.RawQuery != "" {
		if target.RawQuery == "" {
			target.RawQuery = r.URL.RawQuery
		} else {
			target.RawQuery += "&" + r.URL.RawQuery
		}
	}
	finalURL := target.String()

	s.recordHit(r, hitRecord{
		SiteID:    link.SiteID,
		EventType: "link_click",
		Name:      link.Slug,
		Title:     link.Title,
		URL:       finalURL,
		Path:      "/s/" + link.Slug,
	})

	http.Redirect(w, r, finalURL, http.StatusFound)
}

// hitRecord 描述一次待记录的服务端命中（短链点击或像素曝光）。
type hitRecord struct {
	SiteID    uuid.UUID
	EventType string
	Name      string
	Title     string
	URL       string
	Path      string
}

// recordHit 把一次命中异步写入 events，不阻塞响应。
func (s *Server) recordHit(r *http.Request, hit hitRecord) {
	ip := clientIP(r)
	client := analytics.ParseUserAgent(r.UserAgent())

	sum := sha256.Sum256([]byte(ip + "|" + r.UserAgent()))
	visitorID := hex.EncodeToString(sum[:16])

	now := time.Now().UTC()
	referrer := r.Referer()
	referrerHost := analytics.ReferrerHost(referrer)
	utmSource, utmMedium, utmCampaign, utmTerm, utmContent := analytics.UTM(hit.URL)
	location := s.geo.Location(ip)

	in := store.EventInput{
		SiteID:       hit.SiteID,
		Type:         hit.EventType,
		Name:         hit.Name,
		VisitorID:    visitorID,
		SessionID:    visitorID + "-" + now.Format("20060102"),
		URL:          truncate(hit.URL, 2048),
		Path:         truncate(hit.Path, 1024),
		Title:        truncate(hit.Title, 512),
		Referrer:     truncate(referrer, 2048),
		ReferrerHost: referrerHost,
		UTMSource:    truncate(utmSource, 200),
		UTMMedium:    truncate(utmMedium, 200),
		UTMCampaign:  truncate(utmCampaign, 200),
		UTMTerm:      truncate(utmTerm, 200),
		UTMContent:   truncate(utmContent, 200),
		// 短链/像素都是跨站命中，不能拿绑定站点当 internal 基准：
		// 从自己的博客/官网点出去也应该按来源归类（站外推广场景）。
		Channel:        analytics.Channel(referrerHost, "", utmMedium),
		Browser:        client.Browser,
		BrowserVersion: client.BrowserVersion,
		OS:             client.OS,
		OSVersion:      client.OSVersion,
		Device:         client.Device,
		DeviceBrand:    client.DeviceBrand,
		Language:       truncate(r.Header.Get("Accept-Language"), 32),
		Country:        location.Country,
		Region:         location.Region,
		City:           location.City,
		CreatedAt:      now,
	}

	// 不等待落库，写入失败只记日志。
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.store.InsertEvent(ctx, in); err != nil {
			slog.Warn("记录命中事件失败", "err", err, "type", hit.EventType, "name", hit.Name)
		}
	}()
}

func notFoundPage(w http.ResponseWriter, message string) {
	writeTextPage(w, http.StatusNotFound, message)
}

func gonePage(w http.ResponseWriter, message string) {
	writeTextPage(w, http.StatusGone, message)
}

func writeTextPage(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`<!doctype html><html lang="zh-CN"><head><meta charset="utf-8">` +
		`<meta name="viewport" content="width=device-width, initial-scale=1">` +
		`<title>` + message + `</title>` +
		`<style>body{margin:0;min-height:100vh;display:grid;place-items:center;` +
		`font-family:system-ui,-apple-system,"Segoe UI",sans-serif;background:#f8fafc;color:#0f172a}` +
		`div{padding:32px 40px;border-radius:16px;background:#fff;box-shadow:0 20px 50px -30px rgba(15,23,42,.5);text-align:center}` +
		`p{margin:0 0 6px;font-size:15px;color:#475569}</style></head>` +
		`<body><div><p>` + message + `</p></div></body></html>`))
}
