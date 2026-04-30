package api

import (
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/supaback/supaback/internal/appstate"
	"github.com/supaback/supaback/internal/config"
	"github.com/supaback/supaback/internal/scheduler"
	"github.com/supaback/supaback/internal/store"
)

func NewServer(
	cfg *config.Config,
	state *appstate.State,
	s *store.Store,
	sched *scheduler.Scheduler,
) *http.Server {
	h := &Handler{
		state:     state,
		store:     s,
		scheduler: sched,
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	}))

	r.Route("/api", func(r chi.Router) {
		r.Get("/health", h.Health)
		r.Get("/settings", h.GetSettings)
		r.Put("/settings", h.UpdateSettings)
		r.Get("/jobs", h.ListJobs)
		r.Post("/jobs", h.TriggerJob)
		r.Get("/schedules", h.ListSchedules)
		r.Post("/schedules", h.CreateSchedule)
		r.Delete("/schedules/{id}", h.DeleteSchedule)
		r.Patch("/schedules/{id}/toggle", h.ToggleSchedule)
		r.Get("/backups", h.ListBackups)
		r.Get("/backups/download", h.DownloadBackup)
		r.Get("/backups/{date}/download", h.DownloadBackupDate)
	})

	staticDir := cfg.Server.StaticDir
	if staticDir != "" {
		if _, err := os.Stat(staticDir); err == nil {
			// Serve built frontend with SPA fallback
			fs := http.FileServer(http.Dir(staticDir))
			r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
				if _, err := os.Stat(staticDir + req.URL.Path); os.IsNotExist(err) {
					http.ServeFile(w, req, staticDir+"/index.html")
					return
				}
				fs.ServeHTTP(w, req)
			})
			return &http.Server{
				Addr:    fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
				Handler: r,
			}
		}
	}

	// Fallback: frontend not built yet
	r.Get("/*", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, placeholderHTML)
	})

	return &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler: r,
	}
}

const placeholderHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="UTF-8"/>
  <meta name="viewport" content="width=device-width,initial-scale=1"/>
  <title>SupaBack</title>
  <style>
    *{box-sizing:border-box;margin:0;padding:0}
    body{font-family:system-ui,sans-serif;background:#f8fafc;color:#0f172a;display:flex;align-items:center;justify-content:center;min-height:100vh}
    .card{background:#fff;border:1px solid #e2e8f0;border-radius:12px;padding:40px;max-width:480px;width:100%;box-shadow:0 1px 3px rgba(0,0,0,.06)}
    .logo{display:flex;align-items:center;gap:10px;margin-bottom:24px}
    .icon{width:36px;height:36px;background:#4f46e5;border-radius:8px;display:flex;align-items:center;justify-content:center}
    h1{font-size:1.25rem;font-weight:600;margin-bottom:8px}
    p{color:#64748b;font-size:.9rem;line-height:1.6;margin-bottom:16px}
    code{display:block;background:#f1f5f9;border:1px solid #e2e8f0;border-radius:6px;padding:10px 14px;font-family:monospace;font-size:.85rem;margin:6px 0}
    .badge{display:inline-flex;align-items:center;gap:6px;padding:4px 10px;border-radius:20px;font-size:.8rem;font-weight:500}
    .ok{background:#d1fae5;color:#065f46}
    a{color:#4f46e5;text-decoration:none;font-weight:500}
    a:hover{text-decoration:underline}
  </style>
</head>
<body>
  <div class="card">
    <div class="logo">
      <div class="icon">
        <svg width="18" height="18" fill="none" viewBox="0 0 24 24" stroke="white" stroke-width="2.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M20.25 6.375c0 2.278-3.694 4.125-8.25 4.125S3.75 8.653 3.75 6.375m16.5 0c0-2.278-3.694-4.125-8.25-4.125S3.75 4.097 3.75 6.375m16.5 0v11.25c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125V6.375"/>
        </svg>
      </div>
      <strong>SupaBack</strong>
      <span class="badge ok">● API running</span>
    </div>
    <h1>Frontend not built yet</h1>
    <p>The Go API is running. To get the full UI, build the frontend:</p>
    <code>cd apps/web &amp;&amp; npm install &amp;&amp; npm run build</code>
    <p style="margin-top:16px">Or for development (hot-reload):</p>
    <code>cd apps/web &amp;&amp; npm run dev</code>
    <p style="margin-top:16px;font-size:.8rem;color:#94a3b8">
      Then open <a href="http://localhost:5173">localhost:5173</a> (dev) or restart the server after build.
      API is available at <a href="/api/health">/api/health</a>.
    </p>
  </div>
</body>
</html>`
