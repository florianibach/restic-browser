package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Snapshot struct {
	ID       string    `json:"id"`
	Time     time.Time `json:"time"`
	Hostname string    `json:"hostname"`
	Username string    `json:"username"`
	Paths    []string  `json:"paths"`
	Tags     []string  `json:"tags"`
	ShortID  string    `json:"short_id"`
}

type LsEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Type  string `json:"type"`
	Size  int64  `json:"size,omitempty"`
	Mode  int    `json:"mode,omitempty"` // <-- war string
	Mtime string `json:"mtime,omitempty"`
}

type resticLsEvent struct {
	MessageType string `json:"message_type"`

	Name  string `json:"name"`
	Path  string `json:"path"`
	Type  string `json:"type"`
	Size  int64  `json:"size,omitempty"`
	Mode  int    `json:"mode,omitempty"` // <-- war string
	Mtime string `json:"mtime,omitempty"`
}

func isResticRepoRoot(dir string) bool {
	// Minimal robust: config + typische Ordner
	if _, err := os.Stat(filepath.Join(dir, "config")); err != nil {
		return false
	}
	for _, d := range []string{"data", "index", "keys"} {
		if fi, err := os.Stat(filepath.Join(dir, d)); err != nil || !fi.IsDir() {
			return false
		}
	}
	return true
}

// -------------------- Config --------------------

func resticNoLockEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("RESTIC_NO_LOCK")))
	// Default ON für deinen aktuellen Plan
	if v == "" {
		return true
	}
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func resticEnvForRepo(repo RepoConfig) []string {
	env := os.Environ()

	env = append(env,
		"RESTIC_REPOSITORY="+repo.Path,
		"RESTIC_PASSWORD="+repo.Password,
	)

	if cache := os.Getenv("RESTIC_CACHE_DIR"); cache != "" {
		env = append(env, "RESTIC_CACHE_DIR="+cache)
	}

	return env
}

func resticArgsForRepo(repo RepoConfig, args ...string) []string {
	if repo.NoLock {
		return append([]string{"--no-lock"}, args...)
	}
	return args
}

// -------------------- Process runner --------------------

func runRestic(ctx context.Context, repo RepoConfig, args ...string) ([]byte, []byte, error) {
	finalArgs := resticArgsForRepo(repo, args...)
	cmd := exec.CommandContext(ctx, "restic", finalArgs...)
	cmd.Env = resticEnvForRepo(repo)

	var out bytes.Buffer
	var errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb

	err := cmd.Run()
	return out.Bytes(), errb.Bytes(), err
}

// -------------------- API --------------------

func ResticSnapshots(ctx context.Context, repo RepoConfig) ([]Snapshot, error) {
	out, errb, err := runRestic(ctx, repo, "snapshots", "--json")
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, string(errb))
	}

	var snaps []Snapshot
	if e := json.Unmarshal(out, &snaps); e != nil {
		return nil, fmt.Errorf("parse json: %w", e)
	}

	sort.Slice(snaps, func(i, j int) bool {
		return snaps[i].Time.After(snaps[j].Time)
	})

	for i := range snaps {
		if snaps[i].ShortID == "" && len(snaps[i].ID) >= 8 {
			snaps[i].ShortID = snaps[i].ID[:8]
		}
	}
	return snaps, nil
}

func ResticList(ctx context.Context, repo RepoConfig, snapshotID, p string) ([]LsEntry, error) {
	out, errb, err := runRestic(ctx, repo, "ls", snapshotID, p, "--json")
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, string(errb))
	}

	dec := json.NewDecoder(bytes.NewReader(out))

	var entries []LsEntry
	for dec.More() {
		var ev resticLsEvent
		if err := dec.Decode(&ev); err != nil {
			return nil, fmt.Errorf("parse ndjson: %w", err)
		}

		if ev.MessageType != "node" {
			continue
		}

		entries = append(entries, LsEntry{
			Name:  ev.Name,
			Path:  ev.Path,
			Type:  ev.Type,
			Size:  ev.Size,
			Mode:  ev.Mode,
			Mtime: ev.Mtime,
		})
	}

	return entries, nil
}

func ResticDumpToWriter(ctx context.Context, repo RepoConfig, snapshotID, p string, w io.Writer) error {
	cmd := exec.CommandContext(ctx, "restic", resticArgsForRepo(repo, "dump", snapshotID, p)...)
	cmd.Env = resticEnvForRepo(repo)
	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	_, copyErr := io.Copy(w, stdout)
	waitErr := cmd.Wait()

	if copyErr != nil {
		return copyErr
	}
	if waitErr != nil {
		return waitErr
	}
	return nil
}
