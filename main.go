package main

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

//go:embed templates/*
var templateFS embed.FS

type App struct {
	indexTpl  *template.Template
	browseTpl *template.Template
}

func main() {
	// Basic env sanity (minimal)
	if os.Getenv("RESTIC_REPOSITORY") == "" {
		log.Println("WARN: RESTIC_REPOSITORY not set (container should set it, or your shell).")
	}
	if os.Getenv("RESTIC_PASSWORD_FILE") == "" && os.Getenv("RESTIC_PASSWORD") == "" {
		log.Println("WARN: Neither RESTIC_PASSWORD_FILE nor RESTIC_PASSWORD set. restic will likely fail.")
	}

	funcs := template.FuncMap{
		"basename": path.Base,
	}

	indexTpl := template.Must(template.New("").
		Funcs(funcs).
		ParseFS(templateFS, "templates/layout.html", "templates/repository.html"))

	browseTpl := template.Must(template.New("").
		Funcs(funcs).
		ParseFS(templateFS, "templates/layout.html", "templates/browse.html"))

	app := &App{indexTpl: indexTpl, browseTpl: browseTpl}

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/files", http.StatusFound)
	})

	mux.HandleFunc("/files", app.handleFiles)
	mux.HandleFunc("/repositories/{repo}", app.handleSnapshots)
	mux.HandleFunc("/repositories/{repo}/browse", app.handleBrowse)
	mux.HandleFunc("/repositories/{repo}/download", app.handleDownload)
	mux.HandleFunc("/repositories/{repo}/download-zip", app.handleDownloadZip)

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })

	handler := withBasicAuth(mux)

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Println("Listening on http://0.0.0.0:8080")
	log.Fatal(srv.ListenAndServe())
}

func withBasicAuth(next http.Handler) http.Handler {
	user := os.Getenv("BASIC_AUTH_USER")
	pass := os.Getenv("BASIC_AUTH_PASS")
	if user == "" && pass == "" {
		return next // auth disabled by default
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != user || p != pass {
			w.Header().Set("WWW-Authenticate", `Basic realm="restic-browser"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) handleFiles(w http.ResponseWriter, r *http.Request) {
	base := "/repo"

	rel := r.URL.Query().Get("path") // z.B. "docs/srv002"
	rel = strings.TrimPrefix(rel, "/")

	// Prevent traversal
	clean := path.Clean("/" + rel) // -> always absolute-like
	if strings.Contains(clean, "..") {
		http.Error(w, "invalid path", 400)
		return
	}
	clean = strings.TrimPrefix(clean, "/") // back to relative

	abs := filepath.Join(base, filepath.FromSlash(clean))

	// Read directory
	entries, err := os.ReadDir(abs)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// If this folder itself is a restic repo root -> redirect to /repositories/<repo>
	// (In Step 1: repo is ONLY "srv002", so we detect by folder name or mapping)
	if isResticRepoRoot(abs) {
		repoID, ok := a.detectRepoIdFromUrlPath(abs) // e.g. "SRV002" if folder name matches

		if ok {
			http.Redirect(w, r, "/repositories/"+strings.ToLower(repoID), http.StatusFound)
			return
		}
	}

	fmt.Println(entries)

	// Otherwise render file browser table
	// (Provide "ParentPath" for '..' link)
}

func (a *App) detectRepoIdFromUrlPath(abs string) (string, bool) {
	name := filepath.Base(abs)
	id := strings.ToUpper(name)
	_, ok := GetRepo(id)
	return id, ok
}

func (a *App) handleSnapshots(w http.ResponseWriter, r *http.Request) {
	snaps, err := ResticSnapshots(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("restic snapshots failed: %v", err), 500)
		return
	}

	repoID := strings.ToUpper(r.PathValue("repo")) // falls du "srv002" zulässt
	repo, ok := GetRepo(repoID)
	if !ok {
		http.NotFound(w, r)
		return
	}

	data := map[string]any{
		"Body":      "index_body",
		"Snapshots": snaps,
		"Repo":      repo,
	}
	if err := a.indexTpl.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func (a *App) handleBrowse(w http.ResponseWriter, r *http.Request) {
	snap := r.URL.Query().Get("snap")
	p := r.URL.Query().Get("path")

	p = normalizeDirPath(p)
	parent := parentPath(p)

	if snap == "" {
		http.Error(w, "missing snap", 400)
		return
	}
	if p == "" {
		p = "/"
	}

	entries, err := ResticList(r.Context(), snap, p)
	if err != nil {
		http.Error(w, fmt.Sprintf("restic ls failed: %v", err), 500)
		return
	}

	crumbs := buildBreadcrumbs(p)

	data := map[string]any{
		"Title":      "Browse",
		"Body":       "browse_body",
		"Snap":       snap,
		"Path":       p,
		"ParentPath": parent,
		"Crumbs":     crumbs,
		"Entries":    entries,
	}
	if err := a.browseTpl.ExecuteTemplate(w, "browse.html", data); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func buildBreadcrumbs(p string) []map[string]string {
	// p is absolute restic path, e.g. "/" or "/etc/nginx/"
	if p == "" || p[0] != '/' {
		p = "/" + p
	}
	if p == "/" {
		return []map[string]string{{"Name": "/", "Path": "/"}}
	}
	parts := strings.Split(strings.Trim(p, "/"), "/")
	out := []map[string]string{{"Name": "/", "Path": "/"}}
	cur := ""
	for _, part := range parts {
		cur += "/" + part
		out = append(out, map[string]string{"Name": part, "Path": cur + "/"})
	}
	return out
}

func normalizeDirPath(p string) string {
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	// browse behandelt Ordner als ".../"
	if p != "/" && !strings.HasSuffix(p, "/") {
		p += "/"
	}
	return p
}

func parentPath(p string) string {
	p = normalizeDirPath(p)
	if p == "/" {
		return ""
	}
	trim := strings.Trim(p, "/") // "a/b"
	if trim == "" {
		return ""
	}
	parts := strings.Split(trim, "/")
	if len(parts) <= 1 {
		return "/"
	}
	return "/" + strings.Join(parts[:len(parts)-1], "/") + "/"
}

func (a *App) handleDownload(w http.ResponseWriter, r *http.Request) {
	snap := r.URL.Query().Get("snap")
	p := r.URL.Query().Get("path")
	if snap == "" || p == "" {
		http.Error(w, "missing snap or path", 400)
		return
	}

	// Force attachment filename (best-effort)
	filename := path.Base(strings.TrimSuffix(p, "/"))
	if filename == "" || filename == "/" || filename == "." {
		filename = "download.bin"
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	// content-type unknown; browser will sniff or treat as octet-stream
	w.Header().Set("Content-Type", "application/octet-stream")

	if err := ResticDumpToWriter(r.Context(), snap, p, w); err != nil {
		// If headers already started (streaming), can't reliably http.Error.
		log.Printf("download failed snap=%s path=%s err=%v", snap, p, err)
		return
	}
}
