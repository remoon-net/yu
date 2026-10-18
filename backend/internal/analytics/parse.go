package analytics

import (
	"net/url"
	"strings"

	"github.com/mileusna/useragent"
)

// ClientInfo 是从 User-Agent 解析出的客户端信息。
type ClientInfo struct {
	Browser string
	OS      string
	Device  string
}

// ParseUserAgent 解析浏览器、操作系统与设备类型。
func ParseUserAgent(raw string) ClientInfo {
	ua := useragent.Parse(raw)

	browser := ua.Name
	if browser == "" {
		browser = "unknown"
	}

	osName := ua.OS
	if osName == "" {
		osName = "unknown"
	}

	device := "desktop"
	switch {
	case ua.Bot:
		device = "bot"
	case ua.Mobile:
		device = "mobile"
	case ua.Tablet:
		device = "tablet"
	}

	return ClientInfo{Browser: browser, OS: osName, Device: device}
}

// Page 从完整 URL 中提取主机名与路径。
func Page(rawURL string) (host, path string) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", ""
	}

	path = u.Path
	if path == "" {
		path = "/"
	}

	return strings.ToLower(u.Hostname()), path
}

// ReferrerHost 提取来源主机名。
func ReferrerHost(rawURL string) string {
	if rawURL == "" {
		return ""
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	return strings.ToLower(u.Hostname())
}

// UTM 提取三个常用 UTM 参数。
func UTM(rawURL string) (source, medium, campaign string) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", "", ""
	}

	q := u.Query()
	return strings.TrimSpace(q.Get("utm_source")),
		strings.TrimSpace(q.Get("utm_medium")),
		strings.TrimSpace(q.Get("utm_campaign"))
}

// Channel 根据 UTM 与来源推导渠道。
func Channel(referrerHost, siteDomain, utmMedium string) string {
	switch strings.ToLower(strings.TrimSpace(utmMedium)) {
	case "email", "newsletter":
		return "email"
	case "social":
		return "social"
	case "cpc", "ppc", "paid", "paidsearch", "display":
		return "paid"
	case "":
		// 继续按来源判断
	default:
		return "campaign"
	}

	host := strings.ToLower(referrerHost)
	domain := strings.ToLower(siteDomain)

	switch {
	case host == "":
		return "direct"
	case domain != "" && (host == domain || strings.HasSuffix(host, "."+domain)):
		return "internal"
	case isSearchHost(host):
		return "search"
	case isSocialHost(host):
		return "social"
	default:
		return "referral"
	}
}

func matchesAny(host string, suffixes []string) bool {
	for _, suffix := range suffixes {
		if host == suffix || strings.HasSuffix(host, "."+suffix) {
			return true
		}
	}
	return false
}

func isSearchHost(host string) bool {
	return matchesAny(host, []string{
		"google.com", "google.com.hk", "bing.com", "baidu.com", "sogou.com",
		"so.com", "yahoo.com", "duckduckgo.com", "yandex.com", "yandex.ru",
		"ecosia.org", "brave.com", "startpage.com", "qwant.com",
	})
}

func isSocialHost(host string) bool {
	return matchesAny(host, []string{
		"facebook.com", "instagram.com", "twitter.com", "x.com", "t.co",
		"linkedin.com", "reddit.com", "pinterest.com", "youtube.com",
		"tiktok.com", "weibo.com", "zhihu.com", "douyin.com",
		"t.me", "telegram.org", "discord.com", "discord.gg", "mastodon.social",
	})
}
