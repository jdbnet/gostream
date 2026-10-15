package storage

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"gostream/internal/config"
	"gostream/internal/db"
	"gostream/internal/upload"
)

var (
	probeFile     = upload.ExtractMetadata
	normalizeFile = upload.NormalizeAudio
	transcodeFile = upload.TranscodeToStream
)

type ScanProgress struct {
	Running    bool     `json:"running"`
	Phase      string   `json:"phase"`
	Seen       int      `json:"seen"`
	Total      int      `json:"total"`
	Imported   int      `json:"imported"`
	Updated    int      `json:"updated"`
	Removed    int      `json:"removed"`
	Skipped    int      `json:"skipped"`
	Errors     []string `json:"errors"`
	RemovedIDs []int    `json:"removed_ids,omitempty"`
}

type ScanResult struct {
	Imported   int      `json:"imported"`
	Updated    int      `json:"updated"`
	Removed    int      `json:"removed"`
	Skipped    int      `json:"skipped"`
	Errors     []string `json:"errors"`
	RemovedIDs []int    `json:"removed_ids,omitempty"`
}

func (r *ScanResult) Progress(phase string, seen, total int, running bool) ScanProgress {
	if r == nil {
		return ScanProgress{Phase: phase, Seen: seen, Total: total, Running: running}
	}
	return ScanProgress{
		Running: running, Phase: phase, Seen: seen, Total: total,
		Imported: r.Imported, Updated: r.Updated, Removed: r.Removed, Skipped: r.Skipped,
		Errors: r.Errors, RemovedIDs: r.RemovedIDs,
	}
}

type mediaFile struct {
	Path    string
	Key     string
	Size    int64
	ModTime time.Time
}

func ScanLibrary(database *db.DB, backend Backend, cfg *config.Config, progress func(ScanProgress)) (*ScanResult, error) {
	local, ok := backend.(*LocalBackend)
	if !ok {
		return nil, fmt.Errorf("scan is only supported for local storage")
	}
	result := &ScanResult{}
	report := func(phase string, seen, total int, running bool) {
		if progress != nil {
			progress(result.Progress(phase, seen, total, running))
		}
	}

	knownRows, err := database.ListKnownTracks()
	if err != nil {
		return nil, err
	}
	known := make(map[string]db.KnownTrack, len(knownRows))
	for _, row := range knownRows {
		known[row.S3Key] = row
	}
	jingleKeys, err := database.ListJingleKeys()
	if err != nil {
		return nil, err
	}

	report("walking", 0, 0, true)
	trackFiles, trackWalkErr := collectMP3(local.TracksPath(), "tracks")
	jingleFiles, jingleWalkErr := collectMP3(local.JinglesPath(), "jingles")
	if trackWalkErr != nil {
		result.Errors = appendError(result.Errors, trackWalkErr.Error())
	}
	if jingleWalkErr != nil {
		result.Errors = appendError(result.Errors, jingleWalkErr.Error())
	}

	seenTracks := map[string]struct{}{}
	jobs := make([]mediaFile, 0, len(trackFiles))
	for _, file := range trackFiles {
		seenTracks[file.Key] = struct{}{}
		row, exists := known[file.Key]
		if exists && !fileChanged(row, file) {
			result.Skipped++
			continue
		}
		jobs = append(jobs, file)
	}

	report("reading", 0, len(jobs), true)
	prepared := prepareFiles(local, cfg, jobs, known, report)
	report("writing", len(jobs), len(jobs), true)
	if err := writePrepared(database, prepared, result); err != nil {
		return result, err
	}

	seenJingles := map[string]struct{}{}
	for _, file := range jingleFiles {
		seenJingles[file.Key] = struct{}{}
		if _, exists := jingleKeys[file.Key]; exists {
			result.Skipped++
			continue
		}
		if local.JinglesWritable() {
			if _, err := ProcessJingle(database, backend, file.Path, file.Key, ""); err != nil {
				result.Errors = appendError(result.Errors, fmt.Sprintf("%s: %v", file.Key, err))
				continue
			}
		} else if _, err := RegisterJingleFromFile(database, file.Path, file.Key, ""); err != nil {
			result.Errors = appendError(result.Errors, fmt.Sprintf("%s: %v", file.Key, err))
			continue
		}
		result.Imported++
	}

	report("pruning", len(jobs), len(jobs), true)
	if trackWalkErr == nil && local.TracksPath() != "" {
		var removeIDs []int
		for key, row := range known {
			if !strings.HasPrefix(key, "tracks/") {
				continue
			}
			if _, ok := seenTracks[key]; ok {
				continue
			}
			removeIDs = append(removeIDs, row.ID)
		}
		if err := database.DeleteTracks(removeIDs); err != nil {
			return result, err
		}
		result.Removed += len(removeIDs)
		result.RemovedIDs = append(result.RemovedIDs, removeIDs...)
	}
	if jingleWalkErr == nil && local.JinglesPath() != "" {
		var removeJingles []int
		for key, id := range jingleKeys {
			if _, ok := seenJingles[key]; ok {
				continue
			}
			removeJingles = append(removeJingles, id)
		}
		if err := database.DeleteJingles(removeJingles); err != nil {
			return result, err
		}
		result.Removed += len(removeJingles)
	}

	report("done", len(jobs), len(jobs), false)
	return result, nil
}

func RefreshMetadata(database *db.DB, backend Backend, cfg *config.Config, progress func(ScanProgress)) (*ScanResult, error) {
	local, ok := backend.(*LocalBackend)
	if !ok {
		return nil, fmt.Errorf("refresh is only supported for local storage")
	}
	rows, err := database.ListKnownTracks()
	if err != nil {
		return nil, err
	}
	known := make(map[string]db.KnownTrack, len(rows))
	jobs := make([]mediaFile, 0, len(rows))
	for _, row := range rows {
		if !strings.HasPrefix(row.S3Key, "tracks/") {
			continue
		}
		known[row.S3Key] = row
		path, err := local.resolveKey(row.S3Key)
		if err != nil {
			continue
		}
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			continue
		}
		jobs = append(jobs, mediaFile{Path: path, Key: row.S3Key, Size: info.Size(), ModTime: info.ModTime()})
	}
	result := &ScanResult{}
	report := func(phase string, seen, total int, running bool) {
		if progress != nil {
			progress(result.Progress(phase, seen, total, running))
		}
	}
	report("reading", 0, len(jobs), true)
	prepared := prepareFiles(local, cfg, jobs, known, report)
	for i := range prepared {
		prepared[i].forceUpdate = true
	}
	if err := writePrepared(database, prepared, result); err != nil {
		return result, err
	}
	report("done", len(jobs), len(jobs), false)
	return result, nil
}

type preparedTrack struct {
	track       *db.Track
	existingID  int
	forceUpdate bool
	errText     string
	skip        bool
}

func prepareFiles(local *LocalBackend, cfg *config.Config, jobs []mediaFile, known map[string]db.KnownTrack, report func(string, int, int, bool)) []preparedTrack {
	if len(jobs) == 0 {
		return nil
	}
	workers := runtime.NumCPU()
	if workers > 4 {
		workers = 4
	}
	if workers < 1 {
		workers = 1
	}
	in := make(chan mediaFile)
	out := make(chan preparedTrack, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range in {
				out <- prepareOne(local, cfg, job, known[job.Key])
			}
		}()
	}
	go func() {
		for _, job := range jobs {
			in <- job
		}
		close(in)
		wg.Wait()
		close(out)
	}()
	prepared := make([]preparedTrack, 0, len(jobs))
	for item := range out {
		prepared = append(prepared, item)
		report("reading", len(prepared), len(jobs), true)
	}
	return prepared
}

func prepareOne(local *LocalBackend, cfg *config.Config, job mediaFile, existing db.KnownTrack) preparedTrack {
	meta, err := probeFile(job.Path)
	if err != nil && existing.ID > 0 {
		return preparedTrack{errText: fmt.Sprintf("%s: %v", job.Key, err)}
	}
	if err != nil {
		meta = upload.Metadata{}
	}
	title := meta.Title
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(job.Path), filepath.Ext(job.Path))
	}
	path := job.Path
	size := job.Size
	mtime := job.ModTime
	if local.TracksWritable() {
		normalized, stat, normErr := normalizeInPlace(local, job)
		if normErr != nil {
			return preparedTrack{errText: fmt.Sprintf("%s: %v", job.Key, normErr)}
		}
		path = normalized
		size = stat.Size()
		mtime = stat.ModTime()
	} else {
		maybeTranscode(local, cfg, job.Path, job.Key, meta)
	}
	artwork := existing.ArtworkS3Key
	if artwork == "" {
		artwork = extractArtworkKey(local, path, job.Path)
	}
	track := trackFromMeta(title, meta.Artist, meta, fileStat(size, mtime), job.Key, artwork)
	if existing.ID > 0 {
		track.ID = existing.ID
		return preparedTrack{track: track, existingID: existing.ID}
	}
	return preparedTrack{track: track}
}

func fileStat(size int64, mtime time.Time) stubStat {
	return stubStat{size: size, mtime: mtime}
}

type stubStat struct {
	size  int64
	mtime time.Time
}

func (s stubStat) Name() string       { return "" }
func (s stubStat) Size() int64        { return s.size }
func (s stubStat) Mode() fs.FileMode  { return 0 }
func (s stubStat) ModTime() time.Time { return s.mtime }
func (s stubStat) IsDir() bool        { return false }
func (s stubStat) Sys() any           { return nil }

func normalizeInPlace(local *LocalBackend, job mediaFile) (string, os.FileInfo, error) {
	tmp, err := os.CreateTemp("", "gostream-norm-*.mp3")
	if err != nil {
		return "", nil, err
	}
	tmpName := tmp.Name()
	tmp.Close()
	defer os.Remove(tmpName)
	if err := normalizeFile(job.Path, tmpName); err != nil {
		return "", nil, err
	}
	in, err := os.Open(tmpName)
	if err != nil {
		return "", nil, err
	}
	defer in.Close()
	if err := local.Upload(job.Key, in, "audio/mpeg"); err != nil {
		return "", nil, err
	}
	full, err := local.resolveKey(job.Key)
	if err != nil {
		return "", nil, err
	}
	stat, err := os.Stat(full)
	return full, stat, err
}

func maybeTranscode(local *LocalBackend, cfg *config.Config, src, key string, meta upload.Metadata) {
	if cfg == nil || !cfg.Storage.TranscodeReadonlyCache || local.cachePath == "" {
		return
	}
	if !needsTranscode(meta, cfg) {
		return
	}
	dest := local.cacheDestination(key)
	if dest == "" {
		return
	}
	if info, err := os.Stat(dest); err == nil && info.Size() > 0 {
		srcInfo, srcErr := os.Stat(src)
		if srcErr == nil && !srcInfo.ModTime().After(info.ModTime()) {
			return
		}
	}
	_ = transcodeFile(src, dest, cfg.Icecast.Bitrate, cfg.Icecast.SampleRate)
}

func needsTranscode(meta upload.Metadata, cfg *config.Config) bool {
	if meta.Bitrate > 0 && cfg.Icecast.Bitrate > 0 && meta.Bitrate != cfg.Icecast.Bitrate {
		return true
	}
	if meta.SampleRate > 0 && cfg.Icecast.SampleRate > 0 && meta.SampleRate != cfg.Icecast.SampleRate {
		return true
	}
	return false
}

func extractArtworkKey(local *LocalBackend, probePath, srcPath string) string {
	if local.ArtworkPath() == "" || !dirWritable(local.ArtworkPath()) {
		return ""
	}
	tmpDir, err := os.MkdirTemp("", "gostream_artwork")
	if err != nil {
		return ""
	}
	defer os.RemoveAll(tmpDir)
	artworkPath := filepath.Join(tmpDir, "artwork.jpg")
	if err := upload.ExtractArtwork(srcPath, artworkPath); err != nil {
		return ""
	}
	info, err := os.Stat(artworkPath)
	if err != nil || info.Size() == 0 {
		return ""
	}
	file, err := os.Open(artworkPath)
	if err != nil {
		return ""
	}
	defer file.Close()
	key := fmt.Sprintf("artworks/%d_%s.jpg", time.Now().UnixNano(), filepath.Base(srcPath))
	if err := local.Upload(key, file, "image/jpeg"); err != nil {
		return ""
	}
	return key
}

func writePrepared(database *db.DB, items []preparedTrack, result *ScanResult) error {
	inserts := []*db.Track{}
	updates := []*db.Track{}
	flush := func() error {
		if err := database.ApplyTrackBatch(inserts, updates); err != nil {
			return err
		}
		result.Imported += len(inserts)
		result.Updated += len(updates)
		inserts = nil
		updates = nil
		return nil
	}
	for _, item := range items {
		if item.errText != "" {
			result.Errors = appendError(result.Errors, item.errText)
			continue
		}
		if item.track == nil || item.skip {
			result.Skipped++
			continue
		}
		if item.existingID > 0 || item.forceUpdate {
			updates = append(updates, item.track)
		} else {
			inserts = append(inserts, item.track)
		}
		if len(inserts)+len(updates) >= 100 {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	return flush()
}

func collectMP3(root, prefix string) ([]mediaFile, error) {
	if root == "" {
		return nil, nil
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", root)
	}
	var walkErr error
	files := []mediaFile{}
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			walkErr = err
			return nil
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(path), ".mp3") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			walkErr = err
			return nil
		}
		fileInfo, err := d.Info()
		if err != nil {
			walkErr = err
			return nil
		}
		files = append(files, mediaFile{
			Path:    path,
			Key:     prefix + "/" + filepath.ToSlash(rel),
			Size:    fileInfo.Size(),
			ModTime: fileInfo.ModTime(),
		})
		return nil
	})
	if err != nil {
		return files, err
	}
	return files, walkErr
}

func fileChanged(row db.KnownTrack, file mediaFile) bool {
	if row.FileSizeBytes != file.Size {
		return true
	}
	if row.FileMtime == nil {
		return true
	}
	return row.FileMtime.Unix() != file.ModTime.Unix()
}

func appendError(errors []string, msg string) []string {
	if len(errors) >= 50 {
		return errors
	}
	return append(errors, msg)
}

func shouldPrune(rootErr error, walkFailed bool) bool {
	return rootErr == nil && !walkFailed
}
