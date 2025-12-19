package main

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

//go:embed templates/*
var templateFS embed.FS

type App struct {
	indexTpl  *template.Template
	browseTpl *template.Template
	filesTpl  *template.Template
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
		"lower":    strings.ToLower,
	}

	indexTpl := template.Must(template.New("").
		Funcs(funcs).
		ParseFS(templateFS, "templates/layout.html", "templates/snapshot.html"))
	browseTpl := template.Must(template.New("").
		Funcs(funcs).
		ParseFS(templateFS, "templates/layout.html", "templates/browse.html"))
	filesTpl := template.Must(template.New("").
		Funcs(funcs).
		ParseFS(templateFS, "templates/layout.html", "templates/files.html"))

	app := &App{indexTpl: indexTpl, browseTpl: browseTpl, filesTpl: filesTpl}

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

func (a *App) handleSnapshots(w http.ResponseWriter, r *http.Request) {
	repoID := strings.ToUpper(r.PathValue("repo"))
	repo, ok := GetRepo(repoID)
	if !ok {
		http.NotFound(w, r)
		return
	}

	snaps, err := ResticSnapshots(r.Context(), repo)
	if err != nil {
		http.Error(w, fmt.Sprintf("restic snapshots failed: %v", err), 500)
		return
	}

	data := map[string]any{
		"Body":      "index_body",
		"Snapshots": snaps,
		"Repo":      repo,
	}
	if err := a.indexTpl.ExecuteTemplate(w, "snapshot.html", data); err != nil {
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

	repoID := strings.ToUpper(r.PathValue("repo"))
	repo, ok := GetRepo(repoID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	entries, err := ResticList(r.Context(), repo, snap, p)
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

	repoID := strings.ToUpper(r.PathValue("repo"))
	repo, ok := GetRepo(repoID)
	if !ok {
		http.NotFound(w, r)
		return
	}

	if err := ResticDumpToWriter(r.Context(), repo, snap, p, w); err != nil {
		// If headers already started (streaming), can't reliably http.Error.
		log.Printf("download failed snap=%s path=%s err=%v", snap, p, err)
		return
	}
}
