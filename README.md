# GoStream

GoStream is a self-contained internet radio streaming server written in Go. It streams MP3 audio directly to Icecast with near-zero CPU overhead, using SQLite or MariaDB for metadata and local filesystem or S3-compatible storage for audio files.

## Features

- **Gapless Streaming**: Direct MP3 frame extraction and stitching without re-encoding.
- **ICY Metadata**: Injects track metadata into the stream for now-playing info.
- **Library**: Search, filter, and browse by artist, album, and genre. The track list is virtualised so large libraries stay usable.
- **Playlists**: Manual playlists with drag-and-drop order, plus smart playlists that follow rules (artist, album, genre, year, duration, recently added).
- **Station playback**: Each playlist can play in order, shuffle, or weighted rotation (avoid the same artist for a few tracks, and avoid repeating a track for a set number of hours).
- **Jingle Engine**: Automatically inserts jingles after a set number of tracks.
- **Web UI**: Embedded Vue 3 / Tailwind CSS interface.
- **Auto-Normalisation**: Uses `ffmpeg` during upload to normalize writable tracks to 128kbps CBR.
- **Flexible Storage**: Local filesystem (including a read-only music folder), or S3-compatible object storage.
- **Flexible Database**: SQLite (default) or MariaDB. Existing databases are migrated on startup.

## Quick Start (Docker)

1. Clone the repository.
2. Run `docker compose up -d`.
3. The web UI will be available at `http://localhost:8080`.

The default stack uses SQLite and local storage (no separate database container). Published images are available at `ghcr.io/jdbnet/gostream:latest`.

### First-time Setup

1. Open the Web UI and navigate to **Settings**.
2. Provide your **Icecast Source** details.
3. Click **Save Configuration**.
4. Open **Playlists**, create a playlist, add tracks, and set it active.
5. The stream will begin automatically.

You can drop MP3 files into the tracks folder under the data volume (`~/.local/share/gostream/media/tracks/` on the host, or the mounted media path in Docker). They are imported on startup when `storage.auto_scan_on_startup` is true, or from **Scan Library** in Settings.

### Read-only library

Point `storage.tracks_path` at a mounted folder, such as an SMB share of MP3 files. GoStream scans that folder, stores tags in the database, and streams the files in place. Keep `storage.artwork_path` on a writable disk so cover art can be saved. Uploads stay disabled while the tracks folder is read-only.

If the library mixes bitrates or sample rates, set `storage.transcode_readonly_cache` to `true`. GoStream then writes a cache of files that do not match `icecast.bitrate` and `icecast.sample_rate`. This is off by default. `storage.transcode_cache_path` overrides the cache directory (otherwise `~/.local/share/gostream/media/cache`).

A scan only removes missing tracks after it can read the whole folder. If the mount is unavailable, the library is left as it is.

### MariaDB + S3 (optional)

```bash
docker compose -f docker-compose.yaml -f docker-compose.mariadb.yaml up -d
```

Then configure MariaDB and S3 credentials in Settings.

## Configuration

The config file is `~/.config/gostream/config.yaml`. New fields are optional.

- `database.driver`: `sqlite` or `mariadb`.
- `storage.type`: `local` or `s3`.
- `storage.tracks_path`, `storage.jingles_path`, `storage.artwork_path`.
- `storage.auto_scan_on_startup`: scan the local library when the process starts.
- `storage.transcode_readonly_cache` and `storage.transcode_cache_path`: optional cache for read-only files.
- `icecast.bitrate` and `icecast.sample_rate`: stream format. Writable uploads are still normalised to 128 kbps. The read-only cache uses these values.
- Playlist `playback_mode`: `order`, `shuffle` (default, including playlists created before this setting), or `weighted`.
- Weighted rotation uses `artist_separation` (default 5 tracks) and `track_repeat_hours` (default 4).

**Refresh metadata** in Settings re-reads tags for tracks that are already in the database. The next scan does the same for files whose size or modification time changed.

## Development

- Frontend: `cd frontend && npm install && npm run dev`
- Backend: `go build && ./gostream` (build the frontend to `frontend/dist` first so Go can embed it).
