package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	initDB()
	defer db.Close()

	loadTemplates()

	mux := http.NewServeMux()

	// Static files - no Cloudflare caching (always revalidate)
	staticDir := "./static"
	fileServer := http.FileServer(http.Dir(staticDir))

	serveStatic := func(w http.ResponseWriter, r *http.Request, contentType string) {
		fpath := filepath.Join(staticDir, r.URL.Path)
		if _, err := os.Stat(fpath); os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}
		if contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		// Tell Cloudflare to always revalidate with origin
		w.Header().Set("Cache-Control", "public, max-age=0, must-revalidate")
		w.Header().Set("CF-Cache-Status", "DYNAMIC")
		fileServer.ServeHTTP(w, r)
	}

	mux.HandleFunc("/assets/", func(w http.ResponseWriter, r *http.Request) {
		serveStatic(w, r, "")
	})
	mux.HandleFunc("/js/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".map") {
			serveStatic(w, r, "application/json")
		} else {
			serveStatic(w, r, "application/javascript")
		}
	})
	mux.HandleFunc("/css/", func(w http.ResponseWriter, r *http.Request) {
		serveStatic(w, r, "text/css")
	})

	// Public routes
	mux.HandleFunc("GET /login", handleLoginPage)
	mux.HandleFunc("POST /login", handleLoginSubmit)
	mux.HandleFunc("GET /register", handleRegisterPage)
	mux.HandleFunc("POST /register", handleRegisterSubmit)
	mux.HandleFunc("GET /logout", handleLogout)

	// Protected page routes
	mux.HandleFunc("GET /{$}", authRequired(handleDashboard))
	mux.HandleFunc("GET /credentials", authRequired(handleCredentialsPage))
	mux.HandleFunc("GET /setup-fallback", authRequired(handleSetupFallbackPage))
	mux.HandleFunc("GET /create-hostname", authRequired(handleCreateHostnamePage))
	mux.HandleFunc("GET /bulk-hostnames", authRequired(handleBulkHostnamesPage))
	mux.HandleFunc("GET /list-hostnames", authRequired(handleListHostnamesPage))
	mux.HandleFunc("GET /dns-manager", authRequired(handleDnsManagerPage))

	// API routes (POST)
	mux.HandleFunc("POST /check-config", authRequired(handleCheckConfig))
	mux.HandleFunc("POST /check-status-global", authRequired(handleCheckStatusGlobal))
	mux.HandleFunc("POST /setup-fallback", authRequired(handleSetupFallback))
	mux.HandleFunc("POST /create-hostname", authRequired(handleCreateHostnameAPI))
	mux.HandleFunc("POST /check-status", authRequired(handleCheckStatus))
	mux.HandleFunc("POST /list-hostnames", authRequired(handleListHostnames))
	mux.HandleFunc("POST /delete-hostname", authRequired(handleDeleteHostname))
	mux.HandleFunc("POST /delete-dns-record", authRequired(handleDeleteDnsRecord))
	mux.HandleFunc("POST /create-challenge", authRequired(handleCreateChallenge))
	mux.HandleFunc("POST /dns-records", authRequired(handleGetDnsRecords))
	mux.HandleFunc("POST /dns-update", authRequired(handleUpdateDnsRecord))
	mux.HandleFunc("POST /dns-create", authRequired(handleCreateDnsRecord))
	mux.HandleFunc("POST /user-info", authRequired(handleUserInfo))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	log.Printf("🚀 Wildcard-Go starting on :%s", port)
	log.Fatal(http.ListenAndServe("127.0.0.1:"+port, securityHeaders(mux)))
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}
