package main

import (
	"archive/zip"
	"context"
	"fmt"
	"log"
	"net/http"
	"path"
	"strings"
)

func (a *App) handleDownloadZip(w http.ResponseWriter, r *http.Request) {
	snap := r.URL.Query().Get("snap")
	p := r.URL.Query().Get("path")
	if snap == "" {
		http.Error(w, "missing snap", 400)
		return
	}
	p = normalizeDirPath(p)

	filename := "folder.zip"
	if p != "/" {
		filename = strings.Trim(path.Base(strings.Trim(p, "/")), " ")
		if filename == "" {
			filename = "folder"
		}
		filename += ".zip"
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	zw := zip.NewWriter(w)
	defer zw.Close()

	base := strings.TrimSuffix(p, "/") // z.B. "/userdata"
	if base == "" {
		base = "/"
	}

	if err := zipDirFromRestic(r.Context(), zw, snap, p, base); err != nil {
		// Wenn schon gestreamt wird: nur loggen
		log.Printf("zip download failed: %v", err)
		return
	}
}

// p muss ein Ordnerpfad mit trailing "/" sein.
// base ist der "root" der ZIP relativen Pfade.
func zipDirFromRestic(ctx context.Context, zw *zip.Writer, snap, p, base string) error {
	entries, err := ResticList(ctx, snap, strings.TrimSuffix(p, "/"))
	if err != nil {
		return err
	}

	for _, e := range entries {
		// ResticList gibt bei ls <dir> auch den dir-node selbst mit zurück (je nach Version).
		// Wir überspringen den Knoten, der genau dem angefragten Pfad entspricht.
		if normalizeDirPath(e.Path) == normalizeDirPath(p) {
			continue
		}

		if e.Type == "dir" {
			// Rekursion: dir path als ".../"
			if err := zipDirFromRestic(ctx, zw, snap, normalizeDirPath(e.Path), base); err != nil {
				return err
			}
			continue
		}

		// Datei
		rel := e.Path
		if base != "/" && strings.HasPrefix(rel, base+"/") {
			rel = strings.TrimPrefix(rel, base+"/")
		} else if base == "/" && strings.HasPrefix(rel, "/") {
			rel = strings.TrimPrefix(rel, "/")
		}

		fh := &zip.FileHeader{
			Name:   rel,
			Method: zip.Deflate,
		}
		// Optional: mtime, permissions etc. könntest du aus e.Mtime / e.Mode setzen

		zf, err := zw.CreateHeader(fh)
		if err != nil {
			return err
		}

		// restic dump direkt in den ZIP entry streamen
		if err := ResticDumpToWriter(ctx, snap, e.Path, zf); err != nil {
			return err
		}
	}

	return nil
}
