package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gostream/internal/db"
	"gostream/internal/upload"
)

type TrackImportOptions struct {
	Title      string
	Artist     string
	StorageKey string
	PlaylistID int
}

func ProcessTrack(database *db.DB, backend Backend, srcPath string, opts TrackImportOptions) (*db.Track, error) {
	if !backend.Writable() {
		return nil, fmt.Errorf("storage is not writable")
	}

	tmpDir, err := os.MkdirTemp("", "gostream_import")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	outPath := filepath.Join(tmpDir, "output.mp3")

	title := opts.Title
	artist := opts.Artist
	var trackDuration int
	metaTitle, metaArtist, duration, metaErr := upload.ExtractMetadata(srcPath)
	if metaErr == nil {
		if title == "" && metaTitle != "" {
			title = metaTitle
		}
		if artist == "" && metaArtist != "" {
			artist = metaArtist
		}
		trackDuration = duration
	}

	if title == "" {
		title = filepath.Base(srcPath)
		ext := filepath.Ext(title)
		if ext != "" {
			title = title[:len(title)-len(ext)]
		}
	}

	if existingTrack, err := database.GetTrackByTitleAndArtist(title, artist); err == nil && existingTrack != nil {
		if opts.PlaylistID > 0 {
			tracks, _ := database.GetPlaylistTracks(opts.PlaylistID)
			_ = database.AddTrackToPlaylist(opts.PlaylistID, existingTrack.ID, len(tracks))
		}
		return existingTrack, nil
	}

	if err := upload.NormalizeAudio(srcPath, outPath); err != nil {
		return nil, fmt.Errorf("ffmpeg failed: %w", err)
	}

	stat, err := os.Stat(outPath)
	if err != nil {
		return nil, err
	}

	storageKey := opts.StorageKey
	if storageKey == "" {
		storageKey = fmt.Sprintf("tracks/%d_%s", time.Now().UnixNano(), filepath.Base(outPath))
	}

	outFile, err := os.Open(outPath)
	if err != nil {
		return nil, err
	}
	defer outFile.Close()

	if err := backend.Upload(storageKey, outFile, "audio/mpeg"); err != nil {
		return nil, err
	}

	artworkPath := filepath.Join(tmpDir, "artwork.jpg")
	var artworkKey string
	if err := upload.ExtractArtwork(srcPath, artworkPath); err == nil {
		if artStat, err := os.Stat(artworkPath); err == nil && artStat.Size() > 0 {
			if artFile, err := os.Open(artworkPath); err == nil {
				key := fmt.Sprintf("artworks/%d_%s.jpg", time.Now().UnixNano(), filepath.Base(outPath))
				if err := backend.Upload(key, artFile, "image/jpeg"); err == nil {
					artworkKey = key
				}
				artFile.Close()
			}
		}
	}

	t := &db.Track{
		Title:           title,
		Artist:          artist,
		DurationSeconds: trackDuration,
		FileSizeBytes:   stat.Size(),
		S3Key:           storageKey,
		ArtworkS3Key:    artworkKey,
	}
	if err := database.InsertTrack(t); err != nil {
		return nil, err
	}

	if opts.PlaylistID > 0 {
		tracks, _ := database.GetPlaylistTracks(opts.PlaylistID)
		_ = database.AddTrackToPlaylist(opts.PlaylistID, t.ID, len(tracks))
	}

	return t, nil
}

func ProcessJingle(database *db.DB, backend Backend, srcPath, storageKey, name string) (*db.Jingle, error) {
	if !backend.Writable() {
		return nil, fmt.Errorf("storage is not writable")
	}

	tmpDir, err := os.MkdirTemp("", "gostream_jingle_import")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	outPath := filepath.Join(tmpDir, "output.mp3")

	if name == "" {
		name = filepath.Base(srcPath)
		ext := filepath.Ext(name)
		if ext != "" {
			name = name[:len(name)-len(ext)]
		}
	}

	if err := upload.NormalizeAudio(srcPath, outPath); err != nil {
		return nil, fmt.Errorf("ffmpeg failed: %w", err)
	}

	if storageKey == "" {
		storageKey = fmt.Sprintf("jingles/%d_%s", time.Now().UnixNano(), filepath.Base(outPath))
	}

	outFile, err := os.Open(outPath)
	if err != nil {
		return nil, err
	}
	defer outFile.Close()

	if err := backend.Upload(storageKey, outFile, "audio/mpeg"); err != nil {
		return nil, err
	}

	j := &db.Jingle{
		Name:  name,
		S3Key: storageKey,
	}
	if err := database.InsertJingle(j); err != nil {
		return nil, err
	}

	return j, nil
}
