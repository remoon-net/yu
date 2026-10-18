package httpapi

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"remoon.net/yu/backend/internal/config"
	"remoon.net/yu/backend/internal/mailer"
	"remoon.net/yu/backend/internal/store"
)

// Server 持有 HTTP 处理所需的依赖。
type Server struct {
	cfg   config.Config
	store *store.Store
	mail  *mailer.Mailer
}

// New 构建路由。
func New(cfg config.Config, st *store.Store, mail *mailer.Mailer) http.Handler {
	s := &Server{cfg: cfg, store: st, mail: mail}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(s.logRequests)
	r.Use(s.cors)
	r.Use(s.originGuard)

	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/api", func(r chi.Router) {
		r.Post("/auth/login", s.handleLogin)
		r.Post("/auth/logout", s.handleLogout)
		r.Post("/auth/invite/accept", s.handleAcceptInvite)

		r.Group(func(r chi.Router) {
			r.Use(s.authRequired)

			r.Get("/auth/me", s.handleMe)

			r.Group(func(r chi.Router) {
				r.Use(s.csrfProtect)

				r.Get("/workspaces", s.handleListWorkspaces)
				r.Post("/workspaces", s.handleCreateWorkspace)

				r.Route("/workspaces/{wsID}", func(r chi.Router) {
					r.Get("/members", s.handleListMembers)
					r.Patch("/members/{userID}", s.handleUpdateMemberRole)
					r.Delete("/members/{userID}", s.handleRemoveMember)

					r.Get("/invites", s.handleListInvites)
					r.Post("/invites", s.handleCreateInvite)
				})
			})
		})
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "接口不存在")
	})

	return r
}
