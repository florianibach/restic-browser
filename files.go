package main

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type FileEntry struct {
	Name    string
	RelPath string // relative to /repo, using forward slashes, no leading slash
	IsDir   bool
	Size    int64
	ModTime time.Time
	IsRepo  bool   // directory is restic repo root AND configured
	RepoID  string // e.g. "SRV002"
}

type FilesPageModel struct {
	Title      string
	RelPath    string // current folder relative to /repo, "" means root
	ParentRel  string // parent folder relative, "" means root, ""+showParent false
	ShowParent bool
	Entries    []FileEntry
	RepoBase   string // "/repositories"
	FilesBase  string // "/files"
	Error      string
}

var repoNameRe = regexp.MustCompile(`(?i)^srv\d{3}$`)

func (a *App) handleFiles(w http.ResponseWriter, r *http.Request) {
	const base = "/repo"

	rel := strings.TrimSpace(r.URL.Query().Get("path")) // e.g. "docs" or "docs/srv002"
	rel = strings.TrimPrefix(rel, "/")

	// Normalize + prevent traversal
	clean := path.Clean("/" + rel) // safe clean in URL path style
	if strings.Contains(clean, "..") {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	clean = strings.TrimPrefix(clean, "/") // back to relative

	abs := filepath.Join(base, filepath.FromSlash(clean))

	fi, err := os.Stat(abs)
	if err != nil {
		http.Error(w, fmt.Sprintf("path not found: %v", err), http.StatusNotFound)
		return
	}
	if !fi.IsDir() {
		// Optional: handle plain file download later. For now: 400
		http.Error(w, "path is not a directory", http.StatusBadRequest)
		return
	}

	// If this directory is a restic repo AND configured -> redirect to /repositories/{repo}
	if isResticRepoRoot(abs) {
		if repoID, ok := a.detectRepoIdFromUrlPath(abs); ok {
			http.Redirect(w, r, "/repositories/"+strings.ToLower(repoID), http.StatusFound)
			return
		}
		// restic repo exists but not configured: show listing + warning (still useful)
	}

	dirEntries, err := os.ReadDir(abs)
	if err != nil {
		http.Error(w, fmt.Sprintf("read dir failed: %v", err), 500)
		return
	}

	entries := make([]FileEntry, 0, len(dirEntries))
	for _, de := range dirEntries {
		name := de.Name()
		childRel := joinRel(clean, name) // relative (forward slashes)
		childAbs := filepath.Join(abs, name)

		isDir := de.IsDir()
		var size int64
		var mt time.Time

		if info, err := de.Info(); err == nil {
			mt = info.ModTime()
			if !isDir {
				size = info.Size()
			}
		}

		fe := FileEntry{
			Name:    name,
			RelPath: childRel,
			IsDir:   isDir,
			Size:    size,
			ModTime: mt,
		}

		if isDir && isResticRepoRoot(childAbs) {
			if repoID, ok := a.detectRepoIdFromUrlPath(childAbs); ok {
				fe.IsRepo = true
				fe.RepoID = repoID
			}
		}

		entries = append(entries, fe)
	}

	// Sort: directories first, then name
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})

	parentRel, showParent := parentRelPath(clean)

	model := FilesPageModel{
		Title:      "Files",
		RelPath:    clean,
		ParentRel:  parentRel,
		ShowParent: showParent,
		Entries:    entries,
		RepoBase:   "/repositories",
		FilesBase:  "/files",
	}

	if err := a.filesTpl.ExecuteTemplate(w, "files.html", model); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func joinRel(rel, name string) string {
	if rel == "" {
		return name
	}
	return rel + "/" + name
}

func parentRelPath(rel string) (string, bool) {
	// rel is "" for root
	if rel == "" {
		return "", false
	}
	trim := strings.Trim(rel, "/")
	if trim == "" {
		return "", false
	}
	parts := strings.Split(trim, "/")
	if len(parts) <= 1 {
		return "", true // parent is root
	}
	return strings.Join(parts[:len(parts)-1], "/"), true
}

func (a *App) detectRepoIdFromUrlPath(abs string) (string, bool) {
	name := filepath.Base(abs)
	id := strings.ToUpper(name)
	_, ok := GetRepo(id)
	return id, ok
}
