package main

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type ConfigPageModel struct {
	Title  string
	ID     string
	Path   string
	NoLock bool
	Error  string
}

func (a *App) handleConfigGet(w http.ResponseWriter, r *http.Request) {
	id := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("id")))
	p := a.ensureRepoPrefix(strings.TrimSpace(r.URL.Query().Get("path")))

	model := ConfigPageModel{
		Title:  "Configure Repository",
		ID:     id,
		Path:   p,
		NoLock: true,
	}

	// Falls schon vorhanden -> vorfüllen (außer Passwort)
	if id != "" {
		if repo, ok, err := a.store.GetRepo(r.Context(), id); err == nil && ok {
			model.Path = repo.Path
			model.NoLock = repo.NoLock
		}
	}

	_ = a.configTpl.ExecuteTemplate(w, "config.html", model)
}

func (a *App) ensureRepoPrefix(p string) string {
	p = strings.TrimSpace(p)
	if p != "" && !strings.HasPrefix(p, "/") {
		p = "/repo/" + strings.TrimPrefix(p, "/")
	}

	return p
}

func (a *App) handleConfigPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	id := strings.ToUpper(strings.TrimSpace(r.FormValue("id")))
	p := a.ensureRepoPrefix(strings.TrimSpace(r.FormValue("path")))
	pw := r.FormValue("password")
	noLock := r.FormValue("no_lock") == "on"

	// Validierung (minimal, Step 4 härten wir)
	if id == "" || p == "" || pw == "" {
		model := ConfigPageModel{
			Title:  "Configure Repository",
			ID:     id,
			Path:   p,
			NoLock: noLock,
			Error:  "Please fill ID, Path and Password.",
		}
		_ = a.configTpl.ExecuteTemplate(w, "config.html", model)
		return
	}

	// Security: Path muss unter /repo liegen
	base := "/repo"
	clean := filepath.Clean(p)
	if clean != base && !strings.HasPrefix(clean, base+string(os.PathSeparator)) {
		model := ConfigPageModel{
			Title:  "Configure Repository",
			ID:     id,
			Path:   p,
			NoLock: noLock,
			Error:  "Path must be inside /repo.",
		}
		_ = a.configTpl.ExecuteTemplate(w, "config.html", model)
		return
	}

	// Speichern
	if err := a.store.Upsert(r.Context(), RepoConfig{
		ID:       id,
		Path:     clean,
		Password: pw,
		NoLock:   noLock,
	}); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	http.Redirect(w, r, "/repositories/"+strings.ToLower(id), http.StatusFound)
}

func qs(values map[string]string) string {
	v := url.Values{}
	for k, val := range values {
		if val != "" {
			v.Set(k, val)
		}
	}
	return v.Encode()
}
