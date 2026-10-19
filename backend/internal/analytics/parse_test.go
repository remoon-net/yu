package analytics

import "testing"

func TestChannel(t *testing.T) {
	cases := []struct {
		name     string
		referrer string
		domain   string
		medium   string
		want     string
	}{
		{"utm email", "", "example.com", "email", "email"},
		{"utm newsletter", "", "", "newsletter", "email"},
		{"utm social", "", "", "social", "social"},
		{"utm cpc", "", "", "cpc", "paid"},
		{"utm display", "", "", "display", "paid"},
		{"utm other", "", "", "affiliate", "campaign"},
		{"direct", "", "example.com", "", "direct"},
		{"internal", "example.com", "example.com", "", "internal"},
		{"internal subdomain", "blog.example.com", "example.com", "", "internal"},
		{"search", "www.google.com", "example.com", "", "search"},
		{"social", "t.co", "example.com", "", "social"},
		{"referral", "news.ycombinator.com", "example.com", "", "referral"},
		{"referral without domain", "news.ycombinator.com", "", "", "referral"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Channel(tc.referrer, tc.domain, tc.medium); got != tc.want {
				t.Fatalf("Channel(%q, %q, %q) = %q, want %q", tc.referrer, tc.domain, tc.medium, got, tc.want)
			}
		})
	}
}

func TestReferrerHost(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"", ""},
		{"https://Example.COM/post/1", "example.com"},
		{"http://news.ycombinator.com/item?id=1", "news.ycombinator.com"},
		{"not a url", ""},
	}

	for _, tc := range cases {
		if got := ReferrerHost(tc.raw); got != tc.want {
			t.Fatalf("ReferrerHost(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}
}

func TestUTM(t *testing.T) {
	source, medium, campaign, term, content := UTM("https://shop.example.net/a?utm_source=weibo&utm_medium=social&utm_campaign=autumn&utm_term=phone&utm_content=banner")
	if source != "weibo" || medium != "social" || campaign != "autumn" || term != "phone" || content != "banner" {
		t.Fatalf("UTM() = (%q, %q, %q, %q, %q)", source, medium, campaign, term, content)
	}

	source, medium, campaign, term, content = UTM("not a url")
	if source != "" || medium != "" || campaign != "" || term != "" || content != "" {
		t.Fatalf("UTM(invalid) = (%q, %q, %q, %q, %q), want empty", source, medium, campaign, term, content)
	}
}

func TestParseUserAgent(t *testing.T) {
	cases := []struct {
		name        string
		ua          string
		wantDevice  string
		wantBrowser string
		wantBrand   string
	}{
		{
			name:        "chrome desktop",
			ua:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			wantDevice:  "desktop",
			wantBrowser: "Chrome",
			wantBrand:   "",
		},
		{
			name:        "iphone safari",
			ua:          "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
			wantDevice:  "mobile",
			wantBrowser: "Safari",
			wantBrand:   "Apple",
		},
		{
			name:        "samsung android",
			ua:          "Mozilla/5.0 (Linux; Android 13; SM-G991B Build/TP1A.220624.014) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Mobile Safari/537.36",
			wantDevice:  "mobile",
			wantBrowser: "Chrome",
			wantBrand:   "Samsung",
		},
		{
			name:        "redmi android",
			ua:          "Mozilla/5.0 (Linux; Android 10; Redmi Note 8 Pro) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Mobile Safari/537.36",
			wantDevice:  "mobile",
			wantBrowser: "Chrome",
			wantBrand:   "Xiaomi",
		},
		{
			name:        "huawei android",
			ua:          "Mozilla/5.0 (Linux; Android 10; ELS-NX9 Build/HUAWEIELS-N29) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/99.0.4844.88 Mobile Safari/537.36",
			wantDevice:  "mobile",
			wantBrowser: "Chrome",
			wantBrand:   "Huawei",
		},
		{
			name:        "oppo android",
			ua:          "Mozilla/5.0 (Linux; Android 12; CPH2451) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Mobile Safari/537.36",
			wantDevice:  "mobile",
			wantBrowser: "Chrome",
			wantBrand:   "OPPO",
		},
		{
			name:        "vivo android",
			ua:          "Mozilla/5.0 (Linux; Android 11; V2111) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Mobile Safari/537.36",
			wantDevice:  "mobile",
			wantBrowser: "Chrome",
			wantBrand:   "vivo",
		},
		{
			name:        "unknown android",
			ua:          "Mozilla/5.0 (Linux; Android 9; Generic Phone) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Mobile Safari/537.36",
			wantDevice:  "mobile",
			wantBrowser: "Chrome",
			wantBrand:   "其他",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			info := ParseUserAgent(tc.ua)
			if info.Device != tc.wantDevice {
				t.Fatalf("Device = %q, want %q", info.Device, tc.wantDevice)
			}
			if info.Browser != tc.wantBrowser {
				t.Fatalf("Browser = %q, want %q", info.Browser, tc.wantBrowser)
			}
			if info.DeviceBrand != tc.wantBrand {
				t.Fatalf("DeviceBrand = %q, want %q", info.DeviceBrand, tc.wantBrand)
			}
		})
	}
}

func TestParseUserAgentEmpty(t *testing.T) {
	info := ParseUserAgent("")
	if info.Browser != "unknown" || info.OS != "unknown" || info.Device != "desktop" || info.DeviceBrand != "" {
		t.Fatalf("ParseUserAgent(\"\") = %+v", info)
	}
}

func TestPage(t *testing.T) {
	host, path := Page("https://example.com/path?x=1")
	if host != "example.com" || path != "/path" {
		t.Fatalf("Page() = (%q, %q)", host, path)
	}

	if _, path := Page("https://example.com"); path != "/" {
		t.Fatalf("Page(no path) = %q, want /", path)
	}
}
