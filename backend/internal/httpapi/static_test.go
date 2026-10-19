package httpapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"remoon.net/yu/backend/internal/config"
)

func TestStaticFiles(t *testing.T) {
	dir := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("index.html", "<html>app</html>")
	write("_app/immutable/app.js", "console.log(1)")
	write("collect.js", "/* collect */")

	h := New(config.Config{StaticDir: dir}, nil, nil, nil)

	tests := []struct {
		name        string
		method      string
		path        string
		wantStatus  int
		wantBody    string
		wantNoStore bool
		wantCache   string
	}{
		{name: "index", method: http.MethodGet, path: "/", wantStatus: http.StatusOK, wantBody: "<html>app</html>"},
		{name: "spa fallback", method: http.MethodGet, path: "/dashboard", wantStatus: http.StatusOK, wantBody: "<html>app</html>"},
		{name: "immutable asset", method: http.MethodGet, path: "/_app/immutable/app.js", wantStatus: http.StatusOK, wantBody: "console.log(1)", wantCache: "public, max-age=31536000, immutable"},
		{name: "collect script", method: http.MethodGet, path: "/collect.js", wantStatus: http.StatusOK, wantBody: "/* collect */", wantCache: "public, max-age=300"},
		{name: "missing asset", method: http.MethodGet, path: "/missing.js", wantStatus: http.StatusNotFound, wantBody: "接口不存在"},
		{name: "unknown api", method: http.MethodGet, path: "/api/missing", wantStatus: http.StatusNotFound, wantBody: "接口不存在"},
		{name: "post route", method: http.MethodPost, path: "/dashboard", wantStatus: http.StatusNotFound, wantBody: "接口不存在"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Fatalf("body = %q, want contains %q", rec.Body.String(), tt.wantBody)
			}
			if tt.wantCache != "" {
				if got := rec.Header().Get("Cache-Control"); got != tt.wantCache {
					t.Fatalf("Cache-Control = %q, want %q", got, tt.wantCache)
				}
			}
		})
	}
}

func TestStaticFilesDisabledWhenEmpty(t *testing.T) {
	h := New(config.Config{StaticDir: t.TempDir()}, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
