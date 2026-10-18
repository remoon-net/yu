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
	source, medium, campaign := UTM("https://shop.example.net/a?utm_source=weibo&utm_medium=social&utm_campaign=autumn")
	if source != "weibo" || medium != "social" || campaign != "autumn" {
		t.Fatalf("UTM() = (%q, %q, %q)", source, medium, campaign)
	}

	source, medium, campaign = UTM("not a url")
	if source != "" || medium != "" || campaign != "" {
		t.Fatalf("UTM(invalid) = (%q, %q, %q), want empty", source, medium, campaign)
	}
}

func TestParseUserAgent(t *testing.T) {
	cases := []struct {
		name        string
		ua          string
		wantDevice  string
		wantBrowser string
	}{
		{
			name:        "chrome desktop",
			ua:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			wantDevice:  "desktop",
			wantBrowser: "Chrome",
		},
		{
			name:        "iphone safari",
			ua:          "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
			wantDevice:  "mobile",
			wantBrowser: "Safari",
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
		})
	}
}

func TestParseUserAgentEmpty(t *testing.T) {
	info := ParseUserAgent("")
	if info.Browser != "unknown" || info.OS != "unknown" || info.Device != "desktop" {
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
