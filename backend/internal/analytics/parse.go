package analytics

import (
	"net/url"
	"strings"

	"github.com/mileusna/useragent"
)

// ClientInfo 是从 User-Agent 解析出的客户端信息。
type ClientInfo struct {
	Browser        string
	BrowserVersion string
	OS             string
	OSVersion      string
	Device         string
	DeviceBrand    string
}

// ParseUserAgent 解析浏览器、操作系统、设备类型与设备品牌。
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

	return ClientInfo{
		Browser:        browser,
		BrowserVersion: strings.TrimSpace(ua.Version),
		OS:             osName,
		OSVersion:      strings.TrimSpace(ua.OSVersion),
		Device:         device,
		DeviceBrand:    detectDeviceBrand(raw, ua.Device, osName, device),
	}
}

// detectDeviceBrand 从 UA 与设备型号推断品牌，无法识别时归入「其他」。
// 仅移动端/平板返回品牌，桌面端返回空字符串。
func detectDeviceBrand(raw, model, osName, device string) string {
	if device != "mobile" && device != "tablet" {
		return ""
	}

	text := strings.ToLower(raw)
	model = strings.ToLower(strings.TrimSpace(model))

	if osName == useragent.IOS || containsAny(text, "iphone", "ipad", "ipod") {
		return "Apple"
	}

	has := func(values ...string) bool {
		return containsAny(text, values...) || containsAny(model, values...)
	}
	hasPrefix := func(prefixes ...string) bool {
		for _, prefix := range prefixes {
			if strings.HasPrefix(model, prefix) {
				return true
			}
		}
		return false
	}

	switch {
	case has("honor") || hasPrefix("any-", "bvl-", "crt-", "elp-", "fne-", "pgt-", "rky-", "rea-", "tna-", "tft-"):
		return "Honor"
	case has("huawei") || hasPrefix(
		"els-", "ana-", "vog-", "lya-", "ele-", "noh-", "nxt-", "pra-", "duk-",
		"bla-", "lio-", "lon-", "mha-", "tas-", "wlz-", "jny-", "sea-", "vtr-",
		"eva-", "vie-", "fio-", "hry-", "jef-", "bne-", "ada-", "mna-", "nla-",
		"tah-", "pot-", "alb-",
	):
		return "Huawei"
	case has("samsung") || hasPrefix("sm-", "gt-", "sch-", "sgh-", "sph-", "shv-", "sc-"):
		return "Samsung"
	case has("xiaomi", "redmi", "poco") || hasPrefix("redmi", "poco", "mi ", "mi-", "m20", "m21", "m22", "m23", "m24"):
		return "Xiaomi"
	case has("oppo") || hasPrefix("cph", "pbem", "pbet", "pdam", "pfzm", "phz"):
		return "OPPO"
	case has("vivo") || hasPrefix("v1", "v2"):
		return "vivo"
	case has("oneplus", "one plus") || hasPrefix("in2", "le2", "ne2", "kb2", "hd19", "gm19", "gm20", "gm21", "ac2"):
		return "OnePlus"
	case has("realme") || hasPrefix("rmx"):
		return "realme"
	case has("motorola", "moto ") || hasPrefix("xt"):
		return "Motorola"
	case has("pixel"):
		return "Google"
	case has("sony", "xperia") || hasPrefix("xq-"):
		return "Sony"
	case has("lge", "lg-") || hasPrefix("lm-"):
		return "LG"
	case has("nokia") || hasPrefix("ta-"):
		return "Nokia"
	case has("htc") || hasPrefix("htc"):
		return "HTC"
	case has("meizu") || hasPrefix("mz-"):
		return "Meizu"
	case has("nubia", "redmagic"):
		return "Nubia"
	case has("zte"):
		return "ZTE"
	case has("lenovo"):
		return "Lenovo"
	case has("asus", "zenfone", "rog phone"):
		return "ASUS"
	case has("blackberry", "bb10"):
		return "BlackBerry"
	case has("sharp"):
		return "Sharp"
	case has("tcl"):
		return "TCL"
	case has("infinix"):
		return "Infinix"
	case has("tecno", "camon", "spark"):
		return "Tecno"
	case has("itel"):
		return "itel"
	case has("wiko"):
		return "Wiko"
	case has("ulefone"):
		return "Ulefone"
	case has("doogee"):
		return "Doogee"
	case has("cubot"):
		return "Cubot"
	case has("oukitel"):
		return "Oukitel"
	case has("gionee"):
		return "Gionee"
	case has("smartisan"):
		return "Smartisan"
	case has("nothing phone", "a063", "a065"):
		return "Nothing"
	case has("fairphone"):
		return "Fairphone"
	case has("kindle", "silk"):
		return "Amazon"
	case has("lumia"):
		return "Microsoft"
	case osName == useragent.Harmony:
		return "Huawei"
	default:
		return "其他"
	}
}

func containsAny(text string, values ...string) bool {
	for _, value := range values {
		if strings.Contains(text, value) {
			return true
		}
	}
	return false
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

// UTM 提取常用的 UTM 参数。
func UTM(rawURL string) (source, medium, campaign, term, content string) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", "", "", "", ""
	}

	q := u.Query()
	return strings.TrimSpace(q.Get("utm_source")),
		strings.TrimSpace(q.Get("utm_medium")),
		strings.TrimSpace(q.Get("utm_campaign")),
		strings.TrimSpace(q.Get("utm_term")),
		strings.TrimSpace(q.Get("utm_content"))
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
