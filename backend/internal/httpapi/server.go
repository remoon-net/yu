package httpapi

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"remoon.net/yu/backend/internal/config"
	"remoon.net/yu/backend/internal/geoip"
	"remoon.net/yu/backend/internal/mailer"
	"remoon.net/yu/backend/internal/store"
	"remoon.net/yu/backend/internal/version"
)

// Server 持有 HTTP 处理所需的依赖。
type Server struct {
	cfg            config.Config
	store          *store.Store
	mail           *mailer.Mailer
	geo            *geoip.Lookup
	collectLimiter *rateLimiter
	static         *staticFiles
}

// New 构建路由。
func New(cfg config.Config, st *store.Store, mail *mailer.Mailer, geo *geoip.Lookup) http.Handler {
	s := &Server{
		cfg:            cfg,
		store:          st,
		mail:           mail,
		geo:            geo,
		collectLimiter: newRateLimiter(),
		static:         newStaticFiles(cfg.StaticDir),
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(s.logRequests)
	r.Use(s.cors)
	r.Use(s.originGuard)

	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":    "ok",
			"version":   version.Version,
			"commit":    version.Commit,
			"buildTime": version.BuildTime,
			"static":    s.static != nil,
			"geoip":     s.geo != nil,
		})
	})

	// 公开短链跳转（反代或内置静态托管均可）
	r.Get("/s/{slug}", s.handleShortLinkRedirect)
	r.Get("/t/{file}", s.handlePixelImage)
	// 公开监控状态徽章
	r.Get("/monitor/{workspaceID}/{monitorID}/badge.svg", s.handleMonitorBadge)

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
					r.Get("/stats/overview", s.handleWorkspaceStatsOverview)

					r.Get("/members", s.handleListMembers)
					r.Patch("/members/{userID}", s.handleUpdateMemberRole)
					r.Delete("/members/{userID}", s.handleRemoveMember)

					r.Get("/invites", s.handleListInvites)
					r.Post("/invites", s.handleCreateInvite)
					r.Delete("/invites/{inviteID}", s.handleDeleteInvite)

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
					r.Get("/sites/{siteID}/stats/export", s.handleStatsExport)
					r.Get("/sites/{siteID}/stats/retention", s.handleStatsRetention)
					r.Post("/sites/{siteID}/stats/funnel", s.handleStatsFunnel)

					r.Get("/sites/{siteID}/goals", s.handleListGoals)
					r.Post("/sites/{siteID}/goals", s.handleCreateGoal)
					r.Get("/sites/{siteID}/goals/stats", s.handleGoalStats)
					r.Patch("/sites/{siteID}/goals/{goalID}", s.handleUpdateGoal)
					r.Delete("/sites/{siteID}/goals/{goalID}", s.handleDeleteGoal)

					r.Get("/sites/{siteID}/segments", s.handleListSegments)
					r.Post("/sites/{siteID}/segments", s.handleCreateSegment)
					r.Patch("/sites/{siteID}/segments/{segmentID}", s.handleUpdateSegment)
					r.Delete("/sites/{siteID}/segments/{segmentID}", s.handleDeleteSegment)
					r.Post("/sites/{siteID}/share-password", s.handleSetSharePassword)

					r.Get("/sites/{siteID}/short-links", s.handleListShortLinks)
					r.Post("/sites/{siteID}/short-links", s.handleCreateShortLink)
					r.Get("/sites/{siteID}/short-links/{linkID}/stats", s.handleShortLinkStats)
					r.Patch("/sites/{siteID}/short-links/{linkID}", s.handleUpdateShortLink)
					r.Delete("/sites/{siteID}/short-links/{linkID}", s.handleDeleteShortLink)

					r.Get("/sites/{siteID}/pixels", s.handleListPixels)
					r.Post("/sites/{siteID}/pixels", s.handleCreatePixel)
					r.Get("/sites/{siteID}/pixels/{pixelID}/stats", s.handlePixelStats)
					r.Patch("/sites/{siteID}/pixels/{pixelID}", s.handleUpdatePixel)
					r.Delete("/sites/{siteID}/pixels/{pixelID}", s.handleDeletePixel)

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

	r.NotFound(s.handleNotFound)

	return r
}
