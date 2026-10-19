package httpapi

import (
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// staticFiles 托管前端构建产物，并为 SPA 路由提供 index.html 回退。
type staticFiles struct {
	root  string
	index string
}

// newStaticFiles 在目录可用且包含 index.html 时启用内置静态资源。
func newStaticFiles(dir string) *staticFiles {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil
	}

	abs, err := filepath.Abs(dir)
	if err != nil {
		slog.Warn("静态资源目录无效，将仅提供 API", "dir", dir, "err", err)
		return nil
	}

	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		slog.Info("未找到静态资源目录，将仅提供 API", "dir", abs)
		return nil
	}

	entries, err := os.ReadDir(abs)
	if err != nil {
		slog.Warn("无法读取静态资源目录，将仅提供 API", "dir", abs, "err", err)
		return nil
	}
	if len(entries) == 0 {
		slog.Info("静态资源目录为空，将仅提供 API", "dir", abs)
		return nil
	}

	index := filepath.Join(abs, "index.html")
	if info, err := os.Stat(index); err != nil || info.IsDir() {
		slog.Warn("静态资源目录缺少 index.html，将仅提供 API", "dir", abs)
		return nil
	}

	slog.Info("已启用内置静态资源", "dir", abs)
	return &staticFiles{root: abs, index: index}
}

// serve 尝试返回静态文件；文件不存在且看起来像前端路由时回退到 index.html。
func (s *staticFiles) serve(w http.ResponseWriter, r *http.Request) bool {
	clean := path.Clean("/" + r.URL.Path)
	file := filepath.Join(s.root, filepath.FromSlash(clean))
	if file != s.root && !strings.HasPrefix(file, s.root+string(filepath.Separator)) {
		return false
	}

	if info, err := os.Stat(file); err == nil {
		if info.IsDir() {
			index := filepath.Join(file, "index.html")
			if info, err := os.Stat(index); err == nil && !info.IsDir() {
				s.setCache(w, clean+"/")
				http.ServeFile(w, r, index)
				return true
			}
		} else {
			s.setCache(w, clean)
			http.ServeFile(w, r, file)
			return true
		}
	}

	// 带扩展名的请求视为资源文件，缺失时直接 404，避免把图片等请求回退成 HTML。
	if path.Ext(clean) != "" {
		return false
	}

	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFile(w, r, s.index)
	return true
}

func (s *staticFiles) setCache(w http.ResponseWriter, p string) {
	switch {
	case strings.HasPrefix(p, "/_app/immutable/"):
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	case p == "/collect.js":
		w.Header().Set("Cache-Control", "public, max-age=300")
	default:
		w.Header().Set("Cache-Control", "no-cache")
	}
}

// handleNotFound 优先交给静态资源处理；API、短链与像素路径保持 JSON 404。
func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	if s.static != nil && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
		p := r.URL.Path
		if p != "/api" && !strings.HasPrefix(p, "/api/") &&
			!strings.HasPrefix(p, "/s/") && !strings.HasPrefix(p, "/t/") {
			if s.static.serve(w, r) {
				return
			}
		}
	}

	writeError(w, http.StatusNotFound, "接口不存在")
}
