package api

import (
	"encoding/json"
	"html"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/mrered/presto/internal/preview"
	"github.com/mrered/presto/internal/skill"
	"github.com/mrered/presto/internal/template"
	"github.com/mrered/presto/internal/typst"
)

type Server struct {
	mux            *http.ServeMux
	manager        *template.Manager
	compiler       *typst.Compiler
	registry       *template.RegistryCache
	capabilities   ReleaseCapabilities
	availableFonts map[string]bool // cached from typst fonts at startup
	skillManager   *skill.SkillManager
	previewService *preview.Service
}

type ReleaseCapabilities struct {
	ReleaseChannel       string `json:"releaseChannel"`
	OnlineRegistry       bool   `json:"onlineRegistry"`
	OnlineTemplateStore  bool   `json:"onlineTemplateStore"`
	OnlineSkillStore     bool   `json:"onlineSkillStore"`
	TemplateAutoUpdate   bool   `json:"templateAutoUpdate"`
	FirstLaunchBootstrap bool   `json:"firstLaunchBootstrap"`
	AppUpdateCheck       bool   `json:"appUpdateCheck"`
	ExternalBrowserLinks bool   `json:"externalBrowserLinks"`
	LocalTemplateImport  bool   `json:"localTemplateImport"`
	PackagedRuntimes     bool   `json:"packagedRuntimes"`
}

func defaultReleaseCapabilities() ReleaseCapabilities {
	return ReleaseCapabilities{
		ReleaseChannel:       "slim",
		OnlineRegistry:       true,
		OnlineTemplateStore:  true,
		OnlineSkillStore:     true,
		TemplateAutoUpdate:   true,
		FirstLaunchBootstrap: true,
		AppUpdateCheck:       true,
		ExternalBrowserLinks: true,
		LocalTemplateImport:  true,
		PackagedRuntimes:     false,
	}
}

func normalizeReleaseCapabilities(capabilities ReleaseCapabilities) ReleaseCapabilities {
	if capabilities.ReleaseChannel == "" {
		return defaultReleaseCapabilities()
	}
	return capabilities
}

// ServerOptions configures the API server.
type ServerOptions struct {
	TemplatesDir string
	StaticDir    string
	TypstBin     string
	APIKey       string   // empty = desktop mode (no auth required)
	InjectAPIKey bool     // inject API key into static HTML only for trusted local deployments
	FontPaths    []string // additional font directories for typst
	Registry     *template.RegistryCache
	Capabilities ReleaseCapabilities
}

func NewServer(opts ServerOptions) http.Handler {
	// SEC-02: Use a restricted compiler root instead of "/"
	compilerRoot, err := os.MkdirTemp("", "presto-root-*")
	if err != nil {
		log.Printf("[presto] failed to create compiler root dir: %v, using os temp dir", err)
		compilerRoot = os.TempDir()
	}
	compiler := typst.NewCompilerWithRoot(compilerRoot)
	compiler.BinPath = opts.TypstBin
	compiler.FontPaths = opts.FontPaths
	availableFonts := compiler.ListFonts()
	compiler.AvailableFonts = availableFonts

	s := &Server{
		mux:            http.NewServeMux(),
		manager:        template.NewManager(opts.TemplatesDir),
		compiler:       compiler,
		registry:       opts.Registry,
		capabilities:   normalizeReleaseCapabilities(opts.Capabilities),
		availableFonts: availableFonts,
		skillManager:   skill.NewManager(),
		previewService: preview.NewService(),
	}

	log.Printf("[presto] starting server, templates=%s static=%s typst=%s root=%s",
		opts.TemplatesDir, opts.StaticDir, opts.TypstBin, compilerRoot)

	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("POST /api/convert", s.handleConvert)
	s.mux.HandleFunc("POST /api/compile", s.handleCompile)
	s.mux.HandleFunc("POST /api/compile-svg", s.handleCompileSVG)
	s.mux.HandleFunc("POST /api/convert-and-compile", s.handleConvertAndCompile)
	s.mux.HandleFunc("POST /api/preview/update", s.handlePreviewUpdate)
	s.mux.HandleFunc("GET /api/preview/events", s.handlePreviewEvents)
	s.mux.HandleFunc("POST /api/batch", s.handleBatch)
	s.mux.HandleFunc("GET /api/templates", s.handleListTemplates)
	s.mux.HandleFunc("GET /api/templates/discover", s.handleDiscoverTemplates)
	s.mux.HandleFunc("POST /api/templates/{id}/install", s.handleInstallTemplate)
	s.mux.HandleFunc("PATCH /api/templates/{id}", s.handleRenameTemplate)
	s.mux.HandleFunc("DELETE /api/templates/{id}", s.handleDeleteTemplate)
	s.mux.HandleFunc("GET /api/templates/{id}/manifest", s.handleGetManifest)
	s.mux.HandleFunc("GET /api/templates/{id}/example", s.handleGetExample)
	s.mux.HandleFunc("POST /api/templates/{id}/info", s.handleGetOutputInfo)
	s.mux.HandleFunc("POST /api/templates/import", s.handleImportTemplate)
	s.mux.HandleFunc("POST /api/batch/import-zip", s.handleBatchImportZip)

	// Skill management routes
	s.mux.HandleFunc("GET /api/skills", s.handleListSkills)

	if opts.StaticDir != "" {
		// SEC-27: Filter hidden files from static file server
		fs := http.FileServer(http.Dir(opts.StaticDir))
		var static http.Handler
		if opts.APIKey != "" && opts.InjectAPIKey {
			static = apiKeyInjectionHandler(opts.StaticDir, opts.APIKey, fs)
		} else {
			static = fs
		}
		s.mux.Handle("/", dotfileFilterHandler(static))
	}

	// Middleware chain: logging → CORS → securityHeaders → auth → rateLimit → handler
	rl := newRateLimiter(10, 30) // 10 req/s, burst 30 (SEC-19)
	var handler http.Handler = s.mux
	handler = rateLimitMiddleware(rl)(handler)
	handler = authMiddleware(opts.APIKey)(handler)
	handler = securityHeadersMiddleware(handler) // SEC-36
	handler = corsMiddleware(handler)
	handler = loggingMiddleware(handler)
	return handler
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

// writeJSONError writes a safe JSON error response (SEC-15, SEC-16).
// Only generic messages are sent to the client; details should be logged server-side.
func writeJSONError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// apiKeyInjectionHandler intercepts HTML responses and injects the API key
// as a <meta> tag so the embedded frontend can authenticate API requests.
func apiKeyInjectionHandler(staticDir, apiKey string, fallback http.Handler) http.Handler {
	metaTag := `<meta name="api-key" content="` + html.EscapeString(apiKey) + `">`
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if p != "/" && !strings.HasSuffix(p, ".html") {
			fallback.ServeHTTP(w, r)
			return
		}
		if p == "/" {
			p = "/index.html"
		}
		filePath := filepath.Join(staticDir, filepath.Clean(p))
		data, err := os.ReadFile(filePath)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		modified := strings.Replace(string(data), "</head>", metaTag+"\n</head>", 1)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(modified))
	})
}
