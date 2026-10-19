package httpapi

import (
	"fmt"
	"html"
	"math"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const (
	badgeGreen  = "#2ea44f"
	badgeRed    = "#d73a49"
	badgeYellow = "#dbab09"
	badgeGrey   = "#8b949e"
)

// handleMonitorBadge 输出监控状态的 SVG 徽章，可嵌入 README 或站点。
// 路径与 tianji 保持一致：/monitor/{workspaceID}/{monitorID}/badge.svg
func (s *Server) handleMonitorBadge(w http.ResponseWriter, r *http.Request) {
	spec := badgeSpec{label: "yu", message: "unknown", color: badgeGrey}

	workspaceID, err := uuid.Parse(chi.URLParam(r, "workspaceID"))
	if err == nil {
		monitorID, parseErr := uuid.Parse(chi.URLParam(r, "monitorID"))
		if parseErr == nil {
			monitor, getErr := s.store.GetMonitor(r.Context(), workspaceID, monitorID)
			if getErr == nil {
				spec.label = monitor.Name
				spec.message, spec.color = badgeStatus(monitor.LastStatus)

				if !monitor.Enabled {
					spec.message, spec.color = "disabled", badgeGrey
				} else if r.URL.Query().Get("showDetail") != "" && monitor.LastLatencyMS != nil && *monitor.LastLatencyMS > 0 {
					spec.message = fmt.Sprintf("%s %dms", spec.message, *monitor.LastLatencyMS)
				}
			}
		}
	}

	w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, max-age=0, no-store, s-maxage=0, proxy-revalidate")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(renderBadge(spec)))
}

type badgeSpec struct {
	label   string
	message string
	color   string
}

func badgeStatus(status string) (string, string) {
	switch status {
	case "up":
		return "up", badgeGreen
	case "down":
		return "down", badgeRed
	case "pending", "recovering":
		return status, badgeYellow
	default:
		return "unknown", badgeGrey
	}
}

func renderBadge(spec badgeSpec) string {
	label := html.EscapeString(spec.label)
	message := html.EscapeString(spec.message)

	labelWidth := badgeTextWidth(spec.label) + 10
	messageWidth := badgeTextWidth(spec.message) + 10
	totalWidth := labelWidth + messageWidth

	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="20" role="img" aria-label="%s: %s">
  <title>%s: %s</title>
  <linearGradient id="s" x2="0" y2="100%%">
    <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
    <stop offset="1" stop-opacity=".1"/>
  </linearGradient>
  <clipPath id="r"><rect width="%d" height="20" rx="3" fill="#fff"/></clipPath>
  <g clip-path="url(#r)">
    <rect width="%d" height="20" fill="#555"/>
    <rect x="%d" width="%d" height="20" fill="%s"/>
    <rect width="%d" height="20" fill="url(#s)"/>
  </g>
  <g fill="#fff" text-anchor="middle" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" font-size="110">
    <text x="%d" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="%d">%s</text>
    <text x="%d" y="140" transform="scale(.1)" textLength="%d">%s</text>
    <text x="%d" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="%d">%s</text>
    <text x="%d" y="140" transform="scale(.1)" textLength="%d">%s</text>
  </g>
</svg>`,
		totalWidth, label, message, label, message,
		totalWidth,
		labelWidth,
		labelWidth, messageWidth, spec.color,
		totalWidth,
		labelWidth*5, (labelWidth-10)*10, label,
		labelWidth*5, (labelWidth-10)*10, label,
		labelWidth*10+messageWidth*5, (messageWidth-10)*10, message,
		labelWidth*10+messageWidth*5, (messageWidth-10)*10, message,
	)
}

// badgeTextWidth 粗略估算 Verdana 11px 下的文本宽度，中日韩字符按全角计算。
func badgeTextWidth(text string) int {
	width := 0.0
	for _, r := range text {
		if r >= 0x2e80 {
			width += 11
		} else {
			width += 6.5
		}
	}
	return int(math.Ceil(width))
}
