# Restic Browser

[![GitHub Repo](https://img.shields.io/badge/GitHub-Repository-blue?logo=github)](https://github.com/florianibach/restic-browser)

[![DockerHub Repo](https://img.shields.io/badge/Docker_Hub-Repository-blue?logo=docker)](https://hub.docker.com/r/floibach/restic-browser)

A lightweight web UI to **browse, download and inspect restic backups** – built with Go and Docker.

Restic Browser provides:
- a **file browser** for your `/repo` mount
- automatic detection of **restic repositories**
- a **repository configuration page** backed by SQLite (password + no-lock)
- snapshot listing, browsing and downloads (files + folder ZIP)

> This project is not affiliated with restic.

---

This project is built and maintained in my free time.  
If it helps you or saves you some time, you can support my work on [![BuyMeACoffee](https://raw.githubusercontent.com/pachadotdev/buymeacoffee-badges/main/bmc-black.svg)](https://buymeacoffee.com/floibach)

Thank you for your support!

## Features

- File Browser under `/files` (browse `/repo`)
- Detect restic repositories automatically (`config`, `data/`, `index/`, `keys/`)
- Configure repositories via UI (`/config`) and store settings in SQLite
- List snapshots (newest first)
- Browse snapshot contents
- Download individual files
- Download folders as ZIP (streamed)
- Docker-ready (multi-arch: amd64 & arm64)
  - Works perfectly on Raspberry Pi

---

## Quick Start (Docker)

### 1) Prepare folders

- **Repositories root** (your restic repos somewhere inside this):
  - Mounted to: `/repo`
- **Persistent app data** (SQLite DB):
  - Mounted to: `/data`

Example on host:
```bash
mkdir -p ./data
# your restic repos exist somewhere, e.g. /home/${USER}$/backup/repos
````

### 2) Run container

```bash
docker run -d \
  -p 8080:8080 \
  -v /home/${USER}$/backup/repos:/repo:ro \
  -v ./data:/data \
  --name restic-browser \
  floibach/restic-browser:latest
```

Open:

* File browser: [http://localhost:8080/](http://localhost:8080/)
* Health check: [http://localhost:8080/health](http://localhost:8080/health)

---

## Docker Compose Example

```yaml
services:
  restic-browser:
    image: floibach/restic-browser:latest
    ports:
      - "8080:8080"
    environment:
      # Optional:
      # CONFIG_DB_PATH: /data/config.db 
      # RESTIC_CACHE_DIR: /cache
    volumes:
      - /home/florian/backrest/repos:/repo:ro
      - ./data:/data # Path inside container must be same as configured CONFIG_DB_PATH
      # Optional restic cache:
      # - restic-cache:/cache

# volumes:
#   restic-cache:
```

---

## How it works

1. Go to `/files` and browse your mounted `/repo` folder.
2. When a folder is detected as a **restic repository**, you can:

   * open it directly if already configured
   * or configure it via `/config` (password + no-lock)
3. After configuration, Restic Browser can run restic commands for that repo:

   * `restic snapshots --json`
   * `restic ls ... --json`
   * `restic dump ...`
4. Browse snapshot files and download:

   * single file downloads
   * folder ZIP downloads

---

## Configuration

### Environment Variables

| Variable           | Description                            | Default           |
| ------------------ | -------------------------------------- | ----------------- |
| `CONFIG_DB_PATH`   | SQLite file path used for repo configs | `/data/config.db` |
| `RESTIC_CACHE_DIR` | Optional restic cache directory        | (empty)           |

### Volumes

| Container Path | Purpose                                                      |
| -------------- | ------------------------------------------------------------ |
| `/repo`        | Your repositories root (contains one or many restic repos)   |
| `/data`        | Persistent data (SQLite DB)                                  |
| `/cache`       | Optional restic cache (if you set `RESTIC_CACHE_DIR=/cache`) |

> Recommended: mount `/repo` read-only (`:ro`) for safety.

---

## Security Notes

* Repository passwords are currently stored in SQLite (plain text).

  * Consider restricting access to the DB volume.
  * Future improvement could add encryption / OS secret integration.
* Use a reverse proxy + authentication if exposing this service publicly.
* Mount repositories read-only if possible.

---

## Development

Run locally (requires `restic` installed):

```bash
go run .
```

Or use the provided Dev Container.

---

## Roadmap / Ideas

* Config list page `/configs` (edit existing repos)
* Better sorting (folders first, sizes formatted)
* Search within snapshot contents
* Optional basic auth / auth middleware
* Encrypt stored passwords

---

## License

MIT
