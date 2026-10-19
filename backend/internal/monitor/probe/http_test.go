package probe

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestParseHTTPConfigDefaults(t *testing.T) {
	cfg, err := parseHTTPConfig(json.RawMessage(`{"url":"https://example.com"}`))
	if err != nil {
		t.Fatalf("parseHTTPConfig 返回错误: %v", err)
	}

	if cfg.Method != http.MethodGet {
		t.Fatalf("默认 method = %q, want GET", cfg.Method)
	}
	if len(cfg.ExpectedStatus) != 2 || cfg.ExpectedStatus[0] != 200 || cfg.ExpectedStatus[1] != 299 {
		t.Fatalf("默认 expected_status = %v, want [200 299]", cfg.ExpectedStatus)
	}
	if cfg.MaxRedirects != 5 {
		t.Fatalf("默认 max_redirects = %d, want 5", cfg.MaxRedirects)
	}
}

func TestParseHTTPConfigNormalizesHeaders(t *testing.T) {
	raw := json.RawMessage(`{
		"url": "https://example.com",
		"method": " post ",
		"headers": {" X-Token ": " secret ", "Host": "api.example.com"},
		"body": "{\"ping\": true}",
		"expected_status": [201, 204]
	}`)

	cfg, err := parseHTTPConfig(raw)
	if err != nil {
		t.Fatalf("parseHTTPConfig 返回错误: %v", err)
	}

	if cfg.Method != http.MethodPost {
		t.Fatalf("method = %q, want POST", cfg.Method)
	}
	if got := cfg.Headers["X-Token"]; got != "secret" {
		t.Fatalf("X-Token = %q, want secret", got)
	}
	if got := cfg.Headers["Host"]; got != "api.example.com" {
		t.Fatalf("Host = %q, want api.example.com", got)
	}
	if cfg.Body != `{"ping": true}` {
		t.Fatalf("body = %q", cfg.Body)
	}
}

func TestParseHTTPConfigValidation(t *testing.T) {
	cases := []struct {
		name string
		raw  string
	}{
		{"空 URL", `{"url": ""}`},
		{"非 http 协议", `{"url": "ftp://example.com"}`},
		{"method 含空格", `{"url": "https://example.com", "method": "GE T"}`},
		{"状态码过多", `{"url": "https://example.com", "expected_status": [200, 201, 202]}`},
		{"状态码越界", `{"url": "https://example.com", "expected_status": [99]}`},
		{"状态码区间倒置", `{"url": "https://example.com", "expected_status": [299, 200]}`},
		{"Header 名称为空", `{"url": "https://example.com", "headers": {" ": "x"}}`},
		{"Header 名称含空格", `{"url": "https://example.com", "headers": {"X Token": "x"}}`},
		{"Header 含换行", `{"url": "https://example.com", "headers": {"X-Token": "a\nb"}}`},
		{"请求体过大", `{"url": "https://example.com", "body": "` + string(make([]byte, 64<<10+1)) + `"}`},
		{"耗时为负", `{"url": "https://example.com", "max_latency_ms": -1}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseHTTPConfig(json.RawMessage(tc.raw)); err == nil {
				t.Fatal("期望返回错误，实际通过校验")
			}
		})
	}
}

func TestRunHTTPSendsHeadersAndBody(t *testing.T) {
	var (
		gotMethod      string
		gotToken       string
		gotContentType string
		gotBody        string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotToken = r.Header.Get("X-Token")
		gotContentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":"ok"}`)
	}))
	defer server.Close()

	raw := fmt.Sprintf(`{
		"url": %q,
		"method": "post",
		"headers": {"X-Token": "secret"},
		"body": "{\"ping\":true}",
		"expected_status": [200],
		"json_path": "$.status",
		"json_equals": "ok"
	}`, server.URL)

	result := runHTTP(context.Background(), json.RawMessage(raw), 5*time.Second)

	if result.Status != "up" {
		t.Fatalf("status = %q, error = %q", result.Status, result.Error)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("服务端收到 method = %q, want POST", gotMethod)
	}
	if gotToken != "secret" {
		t.Fatalf("服务端收到 X-Token = %q, want secret", gotToken)
	}
	if gotContentType != "application/json" {
		t.Fatalf("服务端收到 Content-Type = %q, want application/json", gotContentType)
	}
	if gotBody != `{"ping":true}` {
		t.Fatalf("服务端收到 body = %q", gotBody)
	}
	if result.StatusCode != http.StatusOK {
		t.Fatalf("status_code = %d, want 200", result.StatusCode)
	}
}
