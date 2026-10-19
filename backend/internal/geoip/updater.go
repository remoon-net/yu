package geoip

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	neturl "net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/oschwald/maxminddb-golang"
)

// DefaultUpdateInterval 是 GeoIP 数据库的默认刷新间隔。
const DefaultUpdateInterval = 7 * 24 * time.Hour

// Updater 负责下载并热更新 MMDB 数据库。
type Updater struct {
	lookup   *Lookup
	path     string
	url      string
	interval time.Duration
	client   *http.Client
}

// NewUpdater 创建更新器。url 为空时使用 DB-IP Country Lite 的当月公开地址。
func NewUpdater(lookup *Lookup, path, url string, interval time.Duration) *Updater {
	return &Updater{
		lookup:   lookup,
		path:     strings.TrimSpace(path),
		url:      strings.TrimSpace(url),
		interval: interval,
		client:   &http.Client{Timeout: 10 * time.Minute},
	}
}

// Start 立即尝试更新一次，之后按 interval 定时更新；interval <= 0 时只更新一次。
func (u *Updater) Start(ctx context.Context) {
	if u == nil || u.path == "" {
		return
	}

	go func() {
		u.updateOnce(ctx)
		if u.interval <= 0 {
			return
		}

		ticker := time.NewTicker(u.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				u.updateOnce(ctx)
			}
		}
	}()
}

func (u *Updater) updateOnce(ctx context.Context) {
	urls := u.sourceURLs(time.Now().UTC())

	var lastErr error
	for i, url := range urls {
		if err := u.download(ctx, url); err != nil {
			lastErr = err
			if i+1 < len(urls) {
				slog.Warn("GeoIP 下载失败，尝试上一月地址", "err", err, "url", url)
				continue
			}
			break
		}
		return
	}

	slog.Warn("更新 GeoIP 数据库失败", "err", lastErr, "path", u.path)
}

func (u *Updater) sourceURLs(now time.Time) []string {
	if u.url != "" {
		return []string{u.url}
	}
	return []string{
		dbipCountryURL(now),
		dbipCountryURL(now.AddDate(0, -1, 0)),
	}
}

func dbipCountryURL(now time.Time) string {
	return "https://download.db-ip.com/free/dbip-country-lite-" + now.UTC().Format("2006-01") + ".mmdb.gz"
}

func (u *Updater) download(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "yu-geoip-updater")
	req.Header.Set("Accept-Encoding", "identity")

	resp, err := u.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	dir := filepath.Dir(u.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmpPath := u.path + ".tmp"
	file, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	writeErr := writeMMDB(file, resp.Body, url)
	closeErr := file.Close()
	if writeErr != nil {
		_ = os.Remove(tmpPath)
		return writeErr
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return closeErr
	}

	reader, err := maxminddb.Open(tmpPath)
	if err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("校验 MMDB 失败: %w", err)
	}
	_ = reader.Close()

	if err := os.Rename(tmpPath, u.path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	if err := u.lookup.Load(u.path); err != nil {
		return err
	}

	info, _ := os.Stat(u.path)
	size := int64(0)
	if info != nil {
		size = info.Size()
	}
	slog.Info("GeoIP 数据库已更新", "path", u.path, "url", url, "size", size)
	return nil
}

func writeMMDB(dst io.Writer, src io.Reader, rawURL string) error {
	parsed, _ := neturl.Parse(rawURL)
	path := strings.ToLower(parsed.Path)

	switch {
	case strings.Contains(strings.ToLower(rawURL), "suffix=tar.gz") || strings.HasSuffix(path, ".tar.gz"):
		gz, err := gzip.NewReader(src)
		if err != nil {
			return err
		}
		defer gz.Close()

		tr := tar.NewReader(gz)
		for {
			header, err := tr.Next()
			if errors.Is(err, io.EOF) {
				return errors.New("压缩包中没有 .mmdb 文件")
			}
			if err != nil {
				return err
			}
			if header.Typeflag != tar.TypeReg || !strings.HasSuffix(strings.ToLower(header.Name), ".mmdb") {
				continue
			}
			_, err = io.Copy(dst, tr)
			return err
		}
	case strings.HasSuffix(path, ".gz"):
		gz, err := gzip.NewReader(src)
		if err != nil {
			return err
		}
		defer gz.Close()
		_, err = io.Copy(dst, gz)
		return err
	default:
		_, err := io.Copy(dst, src)
		return err
	}
}
