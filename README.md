# Restic Browser

[![GitHub Repo](https://img.shields.io/badge/GitHub-Repository-blue?logo=github)](https://github.com/florianibach/restic-browser)

A lightweight web UI to **browse, download and inspect restic backups** – built with Go and Docker.

The Restic Browser lets you:
- list snapshots
- browse snapshot contents like a file browser
- download single files
- download folders as ZIP archives

Designed as a **read-only viewer** for restic repositories, ideal for homelabs, servers and backup monitoring setups.

---

## Features

- List restic snapshots
- Browse snapshot contents
- Download individual files
- Download folders as ZIP (streamed, no temp restore)
- Docker-ready (multi-arch: amd64 & arm64)
    - Works perfectly on Raspberry Pi
- ⚡ Fast, lightweight Go backend
- Ideal for homelab & Backrest setups

---

## Screenshots

coming soon...

---

## Quick Start (Docker)

```bash
docker run -d \
  -p 8080:8080 \
  -v /path/to/restic-repo:/repo \
  -v restic-cache:/cache \
  -e RESTIC_REPOSITORY=/repo \
  -e RESTIC_PASSWORD=yourpassword \
  -e RESTIC_NO_LOCK=true \
  --name restic-browser \
  florianibach/restic-browser:latest
````

Then open:
 [http://localhost:8080](http://localhost:8080)

---

## Docker Compose Example

```yaml
services:
  restic-browser:
    image: floibach/restic-browser:latest
    ports:
      - "8080:8080"
    environment:
      RESTIC_REPOSITORY: /repo
      RESTIC_PASSWORD_FILE: /run/secrets/restic_password
      RESTIC_CACHE_DIR: /cache
      RESTIC_NO_LOCK: "true"
    volumes:
      - /path/to/restic-repo:/repo
      - restic-cache:/cache
    secrets:
      - restic_password

secrets:
  restic_password:
    file: ./secrets/restic_password.txt

volumes:
  restic-cache:
```

---

## Configuration

### Environment Variables

| Variable               | Description                                            |
| ---------------------- | ------------------------------------------------------ |
| `RESTIC_REPOSITORY`    | Path to the restic repository (inside container)       |
| `RESTIC_PASSWORD`      | Restic repository password                             |
| `RESTIC_PASSWORD_FILE` | Alternative to `RESTIC_PASSWORD`                       |
| `RESTIC_CACHE_DIR`     | Restic cache directory                                 |
| `BASIC_AUTH_USER`      | Optional basic auth username                           |
| `BASIC_AUTH_PASS`      | Optional basic auth password                           |

---

## Read-only vs Locks

By default, Restic Browser run with `--no-lock` enabled.

This is useful when:

* backups are running in parallel (e.g. via Backrest)
* the repository is mounted read-only

⚠️ **Note:**
Running without locks is safe for browsing and downloading, but should only be used when you understand the implications.

---

## Security Notes

* The UI does **not modify** the repository
* No restore, prune or unlock operations are exposed
* Use Basic Auth or a reverse proxy when exposing publicly
* Prefer mounting the repository **read-only** if possible

---

## Architecture

* Go (`net/http`, no heavy framework)
* Restic CLI as backend
* NDJSON parsing for `restic ls`
* Streaming downloads (no temp files)
* ZIP generation on-the-fly

---

## Development

Run locally (requires restic installed):

```bash
export RESTIC_REPOSITORY=/path/to/repo
export RESTIC_PASSWORD=yourpassword
go run .
```

Or use the provided Dev Container setup.

---

## Roadmap / Ideas

* Sort folders before files
* Show snapshot root paths directly
* Search inside snapshots
* UI toggle for lock / no-lock mode
* Restore to temp folder (optional)

---

## License

MIT

---

## Disclaimer

This project is **not affiliated with or endorsed by the restic project**.
Restic is an independent open-source backup solution.

---

Built for homelabs ❤️
