package httpapi

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"remoon.net/yu/backend/internal/store"
)

// pixelGIF 是 1×1 的透明 GIF。
var pixelGIF = []byte{
	0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00,
	0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff, 0xff,
	0x21, 0xf9, 0x04, 0x01, 0x00, 0x00, 0x00, 0x00, 0x2c, 0x00,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02,
	0x44, 0x01, 0x00, 0x3b,
}

type pixelRequest struct {
	Slug      string     `json:"slug"`
	Name      string     `json:"name"`
	Enabled   *bool      `json:"enabled"`
	ExpiresAt *time.Time `json:"expires_at"`
}

func (s *Server) handleListPixels(w http.ResponseWriter, r *http.Request) {
	site, ok := s.siteAccess(w, r, "member")
	if !ok {
		return
	}

	pixels, err := s.store.ListPixels(r.Context(), site.ID)
	if err != nil {
		slog.Error("查询像素失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取像素失败")
		return
	}

	stats, err := s.store.PixelStats(r.Context(), site.ID)
	if err != nil {
		slog.Warn("查询像素统计失败", "err", err)
		stats = nil
	}

	writeJSON(w, http.StatusOK, map[string]any{"pixels": pixels, "stats": stats})
}

func (s *Server) handleCreatePixel(w http.ResponseWriter, r *http.Request) {
	site, ok := s.siteAccess(w, r, "admin")
	if !ok {
		return
	}

	var req pixelRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	slug := strings.ToLower(strings.TrimSpace(req.Slug))
	if slug == "" {
		slug = newShortLinkSlug()
	} else if !shortLinkSlugPattern.MatchString(slug) {
		writeError(w, http.StatusBadRequest, "像素标识只能包含小写字母、数字与连字符，长度 2-40")
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	pixel, err := s.store.CreatePixel(r.Context(), site.WorkspaceID, site.ID, store.PixelInput{
		Slug:      slug,
		Name:      req.Name,
		Enabled:   enabled,
		ExpiresAt: req.ExpiresAt,
	})
	if err != nil {
		if errors.Is(err, store.ErrPixelSlugTaken) {
			writeError(w, http.StatusConflict, "像素标识已被占用")
			return
		}
		slog.Error("创建像素失败", "err", err)
		writeError(w, http.StatusInternalServerError, "创建像素失败")
		return
	}

	writeJSON(w, http.StatusCreated, pixel)
}

func (s *Server) handleUpdatePixel(w http.ResponseWriter, r *http.Request) {
	site, ok := s.siteAccess(w, r, "admin")
	if !ok {
		return
	}

	pixelID, err := uuid.Parse(chi.URLParam(r, "pixelID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "像素不存在")
		return
	}

	var req pixelRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	updated, err := s.store.UpdatePixel(r.Context(), site.ID, pixelID, store.PixelInput{
		Name:      req.Name,
		Enabled:   enabled,
		ExpiresAt: req.ExpiresAt,
	})
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "像素不存在")
			return
		}
		slog.Error("更新像素失败", "err", err)
		writeError(w, http.StatusInternalServerError, "更新像素失败")
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleDeletePixel(w http.ResponseWriter, r *http.Request) {
	site, ok := s.siteAccess(w, r, "admin")
	if !ok {
		return
	}

	pixelID, err := uuid.Parse(chi.URLParam(r, "pixelID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "像素不存在")
		return
	}

	if err := s.store.DeletePixel(r.Context(), site.ID, pixelID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "像素不存在")
			return
		}
		slog.Error("删除像素失败", "err", err)
		writeError(w, http.StatusInternalServerError, "删除像素失败")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handlePixelStats 返回单个像素的曝光趋势与分布。
func (s *Server) handlePixelStats(w http.ResponseWriter, r *http.Request) {
	site, ok := s.siteAccess(w, r, "member")
	if !ok {
		return
	}

	pixelID, err := uuid.Parse(chi.URLParam(r, "pixelID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "像素不存在")
		return
	}

	pixel, err := s.store.GetPixel(r.Context(), site.ID, pixelID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "像素不存在")
			return
		}
		slog.Error("查询像素失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取像素失败")
		return
	}

	rn, ok := parseRange(r, site.Timezone)
	if !ok {
		writeError(w, http.StatusBadRequest, "时间范围不合法")
		return
	}

	detail, err := s.store.PixelDetailStats(
		r.Context(), site.ID, pixel.Slug, rn.From, rn.To, rn.Interval, rn.Timezone,
	)
	if err != nil {
		slog.Error("查询像素统计失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取像素统计失败")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"pixel": pixel,
		"range": map[string]any{
			"from":     rn.From,
			"to":       rn.To,
			"interval": rn.Interval,
			"timezone": rn.Timezone,
		},
		"stats": detail,
	})
}

// handlePixelImage 返回 1×1 透明 GIF，命中时异步记录一次曝光。
//
// 无论标识是否存在、是否停用，都返回图片：
// 像素挂在别人的页面里，返回 404 会出现破图，也会被用来枚举有效标识。
func (s *Server) handlePixelImage(w http.ResponseWriter, r *http.Request) {
	defer writePixelImage(w)

	slug := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(chi.URLParam(r, "file")), ".gif"))
	if slug == "" {
		return
	}

	if !s.collectLimiter.allow("pixel|"+clientIP(r), 120, time.Minute) {
		return
	}

	pixel, err := s.store.GetPixelBySlug(r.Context(), slug)
	if err != nil {
		if !errors.Is(err, store.ErrNotFound) {
			slog.Warn("查询像素失败", "err", err, "slug", slug)
		}
		return
	}

	if !pixel.Enabled {
		return
	}
	if pixel.ExpiresAt != nil && pixel.ExpiresAt.Before(time.Now()) {
		return
	}

	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}

	s.recordHit(r, hitRecord{
		SiteID:    pixel.SiteID,
		EventType: "pixel",
		Name:      pixel.Slug,
		Title:     pixel.Name,
		URL:       scheme + "://" + r.Host + r.URL.RequestURI(),
		Path:      "/t/" + pixel.Slug + ".gif",
	})
}

func writePixelImage(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "image/gif")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pixelGIF)
}
