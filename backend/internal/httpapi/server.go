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
	cfg            config.Config
	store          *store.Store
	mail           *mailer.Mailer
	collectLimiter *rateLimiter
}

// New 构建路由。
func New(cfg config.Config, st *store.Store, mail *mailer.Mailer) http.Handler {
	s := &Server{cfg: cfg, store: st, mail: mail, collectLimiter: newRateLimiter()}

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
		r.Post("/collect", s.handleCollect)

		r.Get("/public/status/{slug}", s.handlePublicStatus)
		r.Get("/public/share/{token}", s.handlePublicShare)

		r.Group(func(r chi.Router) {
			r.Use(s.authRequired)

			r.Get("/auth/me", s.handleMe)

			r.Group(func(r chi.Router) {
				r.Use(s.csrfProtect)

				r.Get("/workspaces", s.handleListWorkspaces)
				r.Post("/workspaces", s.handleCreateWorkspace)

				r.Route("/workspaces/{wsID}", func(r chi.Router) {
					r.Delete("/", s.handleDeleteWorkspace)
					r.Get("/members", s.handleListMembers)
					r.Patch("/members/{userID}", s.handleUpdateMemberRole)
					r.Delete("/members/{userID}", s.handleRemoveMember)

					r.Get("/invites", s.handleListInvites)
					r.Post("/invites", s.handleCreateInvite)

					r.Get("/sites", s.handleListSites)
					r.Post("/sites", s.handleCreateSite)
					r.Get("/sites/{siteID}", s.handleGetSite)
					r.Patch("/sites/{siteID}", s.handleUpdateSite)
					r.Delete("/sites/{siteID}", s.handleDeleteSite)

					r.Get("/sites/{siteID}/stats/overview", s.handleStatsOverview)
					r.Get("/sites/{siteID}/stats/realtime", s.handleStatsRealtime)
					r.Get("/sites/{siteID}/stats/pages", s.handleStatsPages)
					r.Get("/sites/{siteID}/stats/sources", s.handleStatsSources)
					r.Get("/sites/{siteID}/stats/audience", s.handleStatsAudience)
					r.Get("/sites/{siteID}/stats/events", s.handleStatsEvents)
					r.Get("/sites/{siteID}/stats/events/{name}", s.handleStatsEventDetail)
					r.Post("/sites/{siteID}/share-password", s.handleSetSharePassword)

					r.Get("/monitors", s.handleListMonitors)
					r.Post("/monitors", s.handleCreateMonitor)
					r.Get("/monitors/{monitorID}", s.handleGetMonitor)
					r.Patch("/monitors/{monitorID}", s.handleUpdateMonitor)
					r.Delete("/monitors/{monitorID}", s.handleDeleteMonitor)
					r.Get("/monitors/{monitorID}/checks", s.handleMonitorChecks)
					r.Get("/monitors/{monitorID}/stats", s.handleMonitorStats)
					r.Post("/monitors/{monitorID}/mute", s.handleMuteMonitor)

					r.Get("/channels", s.handleListChannels)
					r.Post("/channels", s.handleCreateChannel)
					r.Patch("/channels/{channelID}", s.handleUpdateChannel)
					r.Delete("/channels/{channelID}", s.handleDeleteChannel)
					r.Post("/channels/{channelID}/test", s.handleTestChannel)

					r.Get("/incidents", s.handleListIncidents)
					r.Post("/incidents", s.handleCreateIncident)
					r.Patch("/incidents/{incidentID}/resolve", s.handleResolveIncident)

					r.Get("/maintenance-windows", s.handleListMaintenance)
					r.Post("/maintenance-windows", s.handleCreateMaintenance)

					r.Get("/status-pages", s.handleListStatusPages)
					r.Post("/status-pages", s.handleCreateStatusPage)
					r.Get("/status-pages/{pageID}", s.handleGetStatusPage)
					r.Patch("/status-pages/{pageID}", s.handleUpdateStatusPage)
					r.Delete("/status-pages/{pageID}", s.handleDeleteStatusPage)
					r.Put("/status-pages/{pageID}/items", s.handlePutStatusPageItems)
					r.Delete("/maintenance-windows/{windowID}", s.handleDeleteMaintenance)

					r.Get("/api-keys", s.handleListAPIKeys)
					r.Post("/api-keys", s.handleCreateAPIKey)
					r.Delete("/api-keys/{keyID}", s.handleDeleteAPIKey)
				})
			})
		})
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "接口不存在")
	})

	return r
}
