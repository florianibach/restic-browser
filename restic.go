package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
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
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
	Size int64  `json:"size,omitempty"`
	Mode int    `json:"mode,omitempty"`  // <-- war string
	Mtime string `json:"mtime,omitempty"`
}

type resticLsEvent struct {
	MessageType string `json:"message_type"`

	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
	Size int64  `json:"size,omitempty"`
	Mode int    `json:"mode,omitempty"`  // <-- war string
	Mtime string `json:"mtime,omitempty"`
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

func resticEnv() []string {
	env := os.Environ()

	if repo := os.Getenv("RESTIC_REPOSITORY"); repo != "" {
		env = append(env, "RESTIC_REPOSITORY="+repo)
	}
	if pf := os.Getenv("RESTIC_PASSWORD_FILE"); pf != "" {
		env = append(env, "RESTIC_PASSWORD_FILE="+pf)
	}
	if pw := os.Getenv("RESTIC_PASSWORD"); pw != "" {
		env = append(env, "RESTIC_PASSWORD="+pw)
	}
	if cache := os.Getenv("RESTIC_CACHE_DIR"); cache != "" {
		env = append(env, "RESTIC_CACHE_DIR="+cache)
	}

	return env
}

func resticArgs(args ...string) []string {
	if resticNoLockEnabled() {
		// Setzt no-lock global vor den eigentlichen Command
		return append([]string{"--no-lock"}, args...)
	}
	return args
}

// -------------------- Process runner --------------------

func runRestic(ctx context.Context, args ...string) ([]byte, []byte, error) {
	//cmd := exec.CommandContext(ctx, "restic", resticArgs(args...)...)
	finalArgs := resticArgs(args...)
	fmt.Fprintf(os.Stderr, "RESTIC CMD: restic %s\n", strings.Join(finalArgs, " "))

	cmd := exec.CommandContext(ctx, "restic", finalArgs...)
	cmd.Env = resticEnv()

	var out bytes.Buffer
	var errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb

	err := cmd.Run()
	return out.Bytes(), errb.Bytes(), err
}

// -------------------- API --------------------

func ResticSnapshots(ctx context.Context) ([]Snapshot, error) {
	out, errb, err := runRestic(ctx, "snapshots", "--json")
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, string(errb))
	}

	var snaps []Snapshot
	if e := json.Unmarshal(out, &snaps); e != nil {
		return nil, fmt.Errorf("parse json: %w", e)
	}

	for i := range snaps {
		if snaps[i].ShortID == "" && len(snaps[i].ID) >= 8 {
			snaps[i].ShortID = snaps[i].ID[:8]
		}
	}
	return snaps, nil
}

func ResticList(ctx context.Context, snapshotID, p string) ([]LsEntry, error) {
	out, errb, err := runRestic(ctx, "ls", snapshotID, p, "--json")
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
			Name: ev.Name,
			Path: ev.Path,
			Type: ev.Type,
			Size: ev.Size,
			Mode: ev.Mode,
			Mtime: ev.Mtime,
		})
	}

	return entries, nil
}

func ResticDumpToWriter(ctx context.Context, snapshotID, p string, w io.Writer) error {
	cmd := exec.CommandContext(ctx, "restic", resticArgs("dump", snapshotID, p)...)
	cmd.Env = resticEnv()
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
