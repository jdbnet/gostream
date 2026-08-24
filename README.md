# GoStream

GoStream is a self-contained internet radio streaming server written in Go. It streams MP3 audio directly to Icecast with near-zero CPU overhead, using SQLite or MariaDB for metadata and local filesystem or S3-compatible storage for audio files.

## Features

- **Gapless Streaming**: Direct MP3 frame extraction and stitching without re-encoding.
- **ICY Metadata**: Injects track metadata into the stream for now-playing info.
- **Jingle Engine**: Automatically inserts jingles after a set number of tracks.
- **Web UI**: Embedded Vue 3 / Tailwind CSS interface for managing tracks, playlists, and config.
- **Auto-Normalisation**: Uses `ffmpeg` during upload to normalize all tracks to 128kbps CBR.
- **Flexible Storage**: Local filesystem or S3-compatible object storage.
- **Flexible Database**: SQLite (default) or MariaDB.

## Quick Start (Docker)

1. Clone the repository.
2. Run `docker compose up -d`.
3. The web UI will be available at `http://localhost:8080`.

The default stack uses SQLite and local storage (no separate database container). Published images are available at `ghcr.io/jdbnet/gostream:latest`.

### First-time Setup

1. Open the Web UI and navigate to the **Settings** page.
2. Provide your **Icecast Source** details.
3. Click **Save Configuration**.
4. Go to the **Playlists** tab, create a playlist, add some tracks, and click the checkmark to set it as Active.
5. The stream will begin automatically.

You can drop MP3 files into the `tracks/` folder under the data volume (`~/.local/share/gostream/media/tracks/` inside the container). They will be imported automatically on startup, or you can use **Scan Library** in Settings.

### MariaDB + S3 (optional)

For the previous MariaDB + S3 stack, use the override compose file:

```bash
docker compose -f docker-compose.yml -f docker-compose.mariadb.yml up -d
```

Then configure MariaDB and S3 credentials in the Settings page.

## Development

- Frontend: `cd frontend && npm install && npm run dev`
- Backend: `go build && ./gostream` (You must build the frontend to `frontend/dist` first so Go can embed it).
