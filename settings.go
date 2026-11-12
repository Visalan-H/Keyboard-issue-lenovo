package main

// The settings page: a small web app served on 127.0.0.1 and opened in
// the default browser. Every request must carry a random token that only
// this process knows, so other web pages and local users can't read or
// change the config through it.

import (
	"crypto/rand"
	"crypto/subtle"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"sync"
)

//go:embed web/settings.html
var settingsPage []byte

const maxConfigBytes = 64 << 10

// SettingsServer serves the settings page and its small JSON API.
type SettingsServer struct {
	configPath string
	apply      func(*Keymap) // called after a config is saved
	token      string

	mu   sync.Mutex
	host string // "127.0.0.1:port" once listening
	url  string

	// saveMu stops two saves from racing each other.
	saveMu sync.Mutex
}

// NewSettingsServer returns a server for the config at configPath.
// apply is called with the new keymap every time the config is saved.
func NewSettingsServer(configPath string, apply func(*Keymap)) *SettingsServer {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err) // crypto/rand never fails on supported platforms
	}
	return &SettingsServer{configPath: configPath, apply: apply, token: hex.EncodeToString(b)}
}

// URL starts the server the first time it is called and returns the
// address of the settings page, token included.
func (s *SettingsServer) URL() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.url != "" {
		return s.url, nil
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	s.host = ln.Addr().String()
	s.url = "http://" + s.host + "/?t=" + s.token
	go http.Serve(ln, s.Handler())
	return s.url, nil
}

// Handler returns the HTTP handler; exposed for tests.
func (s *SettingsServer) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.page)
	mux.HandleFunc("GET /api/state", s.getState)
	mux.HandleFunc("POST /api/config", s.saveConfig)
	return s.guard(mux)
}

// guard checks the Host header (against DNS rebinding) and the token.
func (s *SettingsServer) guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		host := s.host
		s.mu.Unlock()
		if host != "" && r.Host != host {
			http.Error(w, "wrong host", http.StatusForbidden)
			return
		}
		token := r.Header.Get("X-DeadKeys-Token")
		if r.URL.Path == "/" {
			token = r.URL.Query().Get("t")
		}
		if subtle.ConstantTimeCompare([]byte(token), []byte(s.token)) != 1 {
			http.Error(w, "This link has expired. Open Settings from the DeadKeys tray icon again.", http.StatusForbidden)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

func (s *SettingsServer) page(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy",
		"default-src 'none'; img-src data:; script-src 'unsafe-inline'; style-src 'unsafe-inline'; connect-src 'self'; frame-ancestors 'none'")
	w.Write(settingsPage)
}

type stateResponse struct {
	Config   Config    `json:"config"`
	Defaults Config    `json:"defaults"`
	Error    string    `json:"error,omitempty"` // config.json could not be loaded
	Keys     []KeyInfo `json:"keys"`
	Path     string    `json:"path"`
	Version  string    `json:"version"`
}

func (s *SettingsServer) getState(w http.ResponseWriter, r *http.Request) {
	resp := stateResponse{
		Defaults: DefaultConfig(),
		Keys:     AllKeys(),
		Path:     s.configPath,
		Version:  version,
	}
	cfg, err := LoadConfig(s.configPath)
	if err != nil {
		resp.Config = DefaultConfig()
		resp.Error = err.Error()
	} else {
		resp.Config = cfg
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *SettingsServer) saveConfig(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxConfigBytes))
	if err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, errors.New("config is too large"))
		return
	}
	cfg, err := ParseConfig(data)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	km, err := Compile(cfg)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	if err := SaveConfig(s.configPath, cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s.apply(km)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "config": cfg})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
