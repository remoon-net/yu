package store

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// EventExportRow 是 CSV 导出的一行事件。
type EventExportRow struct {
	CreatedAt      time.Time
	Type           string
	Name           string
	Path           string
	Title          string
	Referrer       string
	ReferrerHost   string
	UTMSource      string
	UTMMedium      string
	UTMCampaign    string
	UTMTerm        string
	UTMContent     string
	Channel        string
	Browser        string
	BrowserVersion string
	OS             string
	OSVersion      string
	Device         string
	DeviceBrand    string
	Screen         string
	Language       string
	Country        string
	Region         string
	City           string
	VisitorID      string
	SessionID      string
}

// ExportEvents 按时间顺序流式读取区间内的事件。
func (s *Store) ExportEvents(ctx context.Context, siteID uuid.UUID, from, to time.Time, fn func(EventExportRow) error) error {
	rows, err := s.Pool.Query(ctx, `
		select
			created_at, type, coalesce(name, ''), coalesce(path, ''), coalesce(title, ''),
			coalesce(referrer, ''), coalesce(referrer_host, ''),
			coalesce(utm_source, ''), coalesce(utm_medium, ''), coalesce(utm_campaign, ''),
			coalesce(utm_term, ''), coalesce(utm_content, ''), coalesce(channel, ''),
			coalesce(browser, ''), coalesce(browser_version, ''),
			coalesce(os, ''), coalesce(os_version, ''),
			coalesce(device, ''), coalesce(device_brand, ''),
			coalesce(screen, ''), coalesce(language, ''),
			coalesce(country, ''), coalesce(region, ''), coalesce(city, ''),
			visitor_id, session_id
		from events
		where site_id = $1 and created_at >= $2 and created_at < $3
		order by created_at`,
		siteID, from, to,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var row EventExportRow
		if err := rows.Scan(
			&row.CreatedAt, &row.Type, &row.Name, &row.Path, &row.Title,
			&row.Referrer, &row.ReferrerHost,
			&row.UTMSource, &row.UTMMedium, &row.UTMCampaign,
			&row.UTMTerm, &row.UTMContent, &row.Channel,
			&row.Browser, &row.BrowserVersion,
			&row.OS, &row.OSVersion,
			&row.Device, &row.DeviceBrand,
			&row.Screen, &row.Language,
			&row.Country, &row.Region, &row.City,
			&row.VisitorID, &row.SessionID,
		); err != nil {
			return err
		}

		if err := fn(row); err != nil {
			return err
		}
	}

	return rows.Err()
}
