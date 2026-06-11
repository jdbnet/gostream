# GoStream

GoStream is a self-contained internet radio streaming server written in Go. It streams MP3 audio directly to Icecast with near-zero CPU overhead, pulling track data from MariaDB and audio files from an S3-compatible backend.

## Features

- **Gapless Streaming**: Direct MP3 frame extraction and stitching without re-encoding.
- **ICY Metadata**: Injects track metadata into the stream for now-playing info.
- **Jingle Engine**: Automatically inserts jingles after a set number of tracks.
- **Web UI**: Embedded Vue 3 / Tailwind CSS interface for managing tracks, playlists, and config.
- **Auto-Normalisation**: Uses `ffmpeg` during upload to normalize all tracks to 128kbps CBR.

## Quick Start (Docker)

1. Clone the repository.
2. Edit `docker-compose.yml` if you want to change default database passwords.
3. Run `docker-compose up -d --build`.
4. The web UI will be available at `http://localhost:8080`.

### First-time Setup

1. Open the Web UI and navigate to the **Settings** page.
2. Update the **Database** password if you changed it in docker-compose (`gostream_pass` by default).
3. Provide your **S3 Storage** credentials (Endpoint, Bucket, Region, Access Key, Secret Key).
4. Provide your **Icecast Source** details.
5. Click **Save Configuration**.
6. Restart the Docker container: `docker-compose restart gostream`
7. Go to the **Playlists** tab, create a playlist, add some tracks, and click the checkmark to set it as Active.
8. The stream will begin automatically.

## Development

- Frontend: `cd frontend && npm install && npm run dev`
- Backend: `go build && ./gostream` (You must build the frontend to `frontend/dist` first so Go can embed it).
