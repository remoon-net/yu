package httpapi

import (
	"encoding/csv"
	"fmt"
	"log/slog"
	"net/http"

	"remoon.net/yu/backend/internal/store"
)

var exportHeader = []string{
	"created_at", "type", "name", "path", "title",
	"referrer", "referrer_host",
	"utm_source", "utm_medium", "utm_campaign", "utm_term", "utm_content", "channel",
	"browser", "browser_version", "os", "os_version", "device", "device_brand",
	"screen", "language", "country", "region", "city",
	"visitor_id", "session_id",
}

func (s *Server) handleStatsExport(w http.ResponseWriter, r *http.Request) {
	site, ok := s.siteAccess(w, r, "member")
	if !ok {
		return
	}

	rn, ok := parseRange(r, site.Timezone)
	if !ok {
		writeError(w, http.StatusBadRequest, "时间范围不合法")
		return
	}

	filename := fmt.Sprintf("yu-events-%s-%s.csv", rn.From.Format("20060102"), rn.To.Format("20060102"))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(w)
	writer.UseCRLF = true
	if err := writer.Write(exportHeader); err != nil {
		slog.Warn("写入 CSV 表头失败", "err", err)
		return
	}

	err := s.store.ExportEvents(r.Context(), site.ID, rn.From, rn.To, func(row store.EventExportRow) error {
		return writer.Write([]string{
			row.CreatedAt.UTC().Format("2006-01-02 15:04:05"),
			row.Type, row.Name, row.Path, row.Title,
			row.Referrer, row.ReferrerHost,
			row.UTMSource, row.UTMMedium, row.UTMCampaign, row.UTMTerm, row.UTMContent, row.Channel,
			row.Browser, row.BrowserVersion, row.OS, row.OSVersion, row.Device, row.DeviceBrand,
			row.Screen, row.Language, row.Country, row.Region, row.City,
			row.VisitorID, row.SessionID,
		})
	})

	writer.Flush()
	if err != nil {
		slog.Error("导出事件失败", "err", err)
		return
	}
	if err := writer.Error(); err != nil {
		slog.Error("写入 CSV 失败", "err", err)
	}
}
