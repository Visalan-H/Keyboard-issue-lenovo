package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func newTestSettings(t *testing.T) (*SettingsServer, *httptest.Server, *[]*Keymap) {
	t.Helper()
	var applied []*Keymap
	path := filepath.Join(t.TempDir(), "config.json")
	s := NewSettingsServer(path, func(km *Keymap) { applied = append(applied, km) })
	srv := httptest.NewServer(s.Handler())
	t.Cleanup(srv.Close)
	return s, srv, &applied
}

func request(t *testing.T, method, url, token, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		req.Header.Set("X-DeadKeys-Token", token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { res.Body.Close() })
	return res
}

func TestSettingsNeedsToken(t *testing.T) {
	s, srv, _ := newTestSettings(t)
	for _, c := range []struct{ method, path, token string }{
		{"GET", "/", ""},
		{"GET", "/?t=" + s.token, ""}, // the old query form is gone
		{"GET", "/s/", ""},
		{"GET", "/s/wrong", ""},
		{"GET", "/s/" + s.token + "/extra", ""},
		{"GET", "/api/state", ""},
		{"GET", "/api/state", "wrong"},
		{"POST", "/api/config", ""},
		{"GET", "/api/state?t=" + s.token, ""}, // the API only takes the header
	} {
		res := request(t, c.method, srv.URL+c.path, c.token, "{}")
		if res.StatusCode != http.StatusForbidden && res.StatusCode != http.StatusNotFound {
			t.Errorf("%s %s: status %d, want 403 or 404", c.method, c.path, res.StatusCode)
		}
	}
	res := request(t, "GET", srv.URL+"/s/"+s.token, "", "")
	if res.StatusCode != http.StatusOK || !strings.Contains(res.Header.Get("Content-Type"), "text/html") {
		t.Errorf("page: status %d, type %q", res.StatusCode, res.Header.Get("Content-Type"))
	}
}

func TestSettingsChecksHost(t *testing.T) {
	s, srv, _ := newTestSettings(t)
	s.mu.Lock()
	s.host = "127.0.0.1:1" // pretend we listen elsewhere
	s.mu.Unlock()
	res := request(t, "GET", srv.URL+"/api/state", s.token, "")
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("status %d, want 403 for wrong Host", res.StatusCode)
	}
}

func TestSettingsStateAndSave(t *testing.T) {
	s, srv, applied := newTestSettings(t)

	var state stateResponse
	res := request(t, "GET", srv.URL+"/api/state", s.token, "")
	if err := json.NewDecoder(res.Body).Decode(&state); err != nil {
		t.Fatal(err)
	}
	if state.Error != "" || len(state.Config.Layer) != 6 || len(state.Keys) < 100 {
		t.Fatalf("unexpected state: error %q, %d layer rules, %d keys", state.Error, len(state.Config.Layer), len(state.Keys))
	}

	cfg := state.Config
	cfg.Remap = append(cfg.Remap, Rule{"Insert", "S"})
	body, _ := json.Marshal(cfg)
	res = request(t, "POST", srv.URL+"/api/config", s.token, string(body))
	if res.StatusCode != http.StatusOK {
		t.Fatalf("save: status %d", res.StatusCode)
	}
	if len(*applied) != 1 || (*applied)[0].Remap[vkInsert] != 'S' {
		t.Fatalf("new keymap not applied: %+v", *applied)
	}
	onDisk, err := LoadConfig(s.configPath)
	if err != nil || len(onDisk.Remap) != 1 {
		t.Fatalf("not saved to disk: %+v, %v", onDisk, err)
	}
}

func TestSettingsRejectsBadConfig(t *testing.T) {
	s, srv, applied := newTestSettings(t)
	res := request(t, "POST", srv.URL+"/api/config", s.token, `{"layer": [{"from": "Q", "to": "Nope"}]}`)
	var body map[string]string
	json.NewDecoder(res.Body).Decode(&body)
	if res.StatusCode != http.StatusBadRequest || !strings.Contains(body["error"], "unknown key") {
		t.Errorf("status %d, body %v", res.StatusCode, body)
	}
	if len(*applied) != 0 {
		t.Error("bad config must not be applied")
	}

	res = request(t, "POST", srv.URL+"/api/config", s.token, strings.Repeat(" ", maxConfigBytes+1))
	if res.StatusCode != http.StatusRequestEntityTooLarge {
		t.Errorf("huge body: status %d", res.StatusCode)
	}
}

func TestSettingsReportsBrokenConfigFile(t *testing.T) {
	s, srv, _ := newTestSettings(t)
	if err := writeFile(s.configPath, `{"layer": [{"from": "Q"`); err != nil {
		t.Fatal(err)
	}
	var state stateResponse
	res := request(t, "GET", srv.URL+"/api/state", s.token, "")
	json.NewDecoder(res.Body).Decode(&state)
	if state.Error == "" || len(state.Config.Layer) != 6 {
		t.Errorf("want an error and the defaults, got %q and %d rules", state.Error, len(state.Config.Layer))
	}
}

func TestSettingsURLStartsOnce(t *testing.T) {
	s := NewSettingsServer(filepath.Join(t.TempDir(), "config.json"), func(*Keymap) {})
	a, err := s.URL()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := s.URL()
	// No "?" in the URL: explorer.exe silently drops URLs with a query
	// string, so the token goes in the path.
	if a != b || !strings.HasPrefix(a, "http://127.0.0.1:") || !strings.HasSuffix(a, "/s/"+s.token) || strings.Contains(a, "?") {
		t.Errorf("URLs %q, %q", a, b)
	}
	res, err := http.Get(a)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Errorf("live server: status %d", res.StatusCode)
	}
}
