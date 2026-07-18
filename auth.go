package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	sessions   = map[string]*sessionData{}
	sessionsMu sync.RWMutex
)

type sessionData struct {
	UserID   int
	Username string
	Expiry   time.Time
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func createSession(w http.ResponseWriter, userID int, username string) string {
	token := generateToken()
	sessionsMu.Lock()
	sessions[token] = &sessionData{UserID: userID, Username: username, Expiry: time.Now().Add(7 * 24 * time.Hour)}
	sessionsMu.Unlock()
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   604800,
	})
	return token
}

func getSession(r *http.Request) *sessionData {
	c, err := r.Cookie("session")
	if err != nil {
		return nil
	}
	sessionsMu.RLock()
	s, ok := sessions[c.Value]
	sessionsMu.RUnlock()
	if !ok || time.Now().After(s.Expiry) {
		return nil
	}
	return s
}

func destroySession(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("session")
	if err == nil {
		sessionsMu.Lock()
		delete(sessions, c.Value)
		sessionsMu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: "session", Value: "", Path: "/", MaxAge: -1})
}

func authRequired(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess := getSession(r)
		if sess == nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		r.Header.Set("X-User-Id", strconv.Itoa(sess.UserID))
		r.Header.Set("X-Username", sess.Username)
		next(w, r)
	}
}

var funcMap = template.FuncMap{
	"activeClass": func(active, current string) string {
		if active == current {
			return "active"
		}
		return ""
	},
}

var pageTemplates = map[string]*template.Template{}
var loginTmpl *template.Template
var registerTmpl *template.Template

func loadTemplates() {
	pages := []string{
		"dashboard.html", "credentials.html", "setup-fallback.html",
		"create-hostname.html", "bulk-hostnames.html", "list-hostnames.html", "dns-manager.html",
	}
	for _, p := range pages {
		t, err := template.New("").Funcs(funcMap).ParseFiles("templates/base.html", "templates/"+p)
		if err != nil {
			log.Fatalf("Template parse error %s: %v", p, err)
		}
		pageTemplates[p] = t
	}

	loginTmpl = template.Must(template.New("").Funcs(funcMap).ParseFiles("templates/login.html"))
	registerTmpl = template.Must(template.New("").Funcs(funcMap).ParseFiles("templates/register.html"))

	log.Println("✅ Templates loaded")
}

func renderPage(w http.ResponseWriter, name string, data map[string]interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t, ok := pageTemplates[name]
	if !ok {
		http.Error(w, "Template not found: "+name, 500)
		return
	}
	if err := t.ExecuteTemplate(w, "base", data); err != nil {
		log.Printf("[TEMPLATE ERROR] %s: %v", name, err)
		http.Error(w, "Template error", 500)
	}
}

func jsonResp(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

func handleLoginPage(w http.ResponseWriter, r *http.Request) {
	if getSession(r) != nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	loginTmpl.ExecuteTemplate(w, "login.html", nil)
}

func handleRegisterPage(w http.ResponseWriter, r *http.Request) {
	if getSession(r) != nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	registerTmpl.ExecuteTemplate(w, "register.html", nil)
}

func handleLoginSubmit(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")
	if username == "" || password == "" {
		jsonResp(w, 400, map[string]interface{}{"success": false, "error": "Username dan password wajib diisi"})
		return
	}
	var id int
	var hash string
	err := db.QueryRow("SELECT id, password FROM users WHERE username = ?", username).Scan(&id, &hash)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		jsonResp(w, 401, map[string]interface{}{"success": false, "error": "Username atau password salah"})
		return
	}
	createSession(w, id, username)
	jsonResp(w, 200, map[string]interface{}{"success": true, "redirect": "/"})
}

func handleRegisterSubmit(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")
	confirm := r.FormValue("password_confirm")
	if username == "" || password == "" {
		jsonResp(w, 400, map[string]interface{}{"success": false, "error": "Semua field wajib diisi"})
		return
	}
	if len(username) < 3 {
		jsonResp(w, 400, map[string]interface{}{"success": false, "error": "Username minimal 3 karakter"})
		return
	}
	if len(password) < 6 {
		jsonResp(w, 400, map[string]interface{}{"success": false, "error": "Password minimal 6 karakter"})
		return
	}
	if password != confirm {
		jsonResp(w, 400, map[string]interface{}{"success": false, "error": "Password tidak cocok"})
		return
	}
	var exists int
	db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", username).Scan(&exists)
	if exists > 0 {
		jsonResp(w, 409, map[string]interface{}{"success": false, "error": "Username sudah digunakan"})
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	res, err := db.Exec("INSERT INTO users (username, password) VALUES (?, ?)", username, string(hash))
	if err != nil {
		jsonResp(w, 500, map[string]interface{}{"success": false, "error": "Gagal membuat akun"})
		return
	}
	uid, _ := res.LastInsertId()
	createSession(w, int(uid), username)
	jsonResp(w, 200, map[string]interface{}{"success": true, "redirect": "/"})
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	destroySession(w, r)
	http.Redirect(w, r, "/login", http.StatusFound)
}
