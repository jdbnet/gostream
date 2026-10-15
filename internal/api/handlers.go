package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"gostream/internal/config"
	"gostream/internal/db"
	"gostream/internal/storage"
	"gostream/internal/stream"
)

// Tracks
func (s *Server) handleGetTracks(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}
	offset := (page - 1) * limit
	searchQuery := c.Query("q")

	tracks, total, err := s.database.GetTracks(searchQuery, offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"tracks": tracks,
		"total":  total,
	})
}

func (s *Server) handleUploadTrack(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file missing"})
		return
	}
	defer file.Close()

	title := c.PostForm("title")
	artist := c.PostForm("artist")

	tmpDir, err := os.MkdirTemp("", "gostream_upload")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer os.RemoveAll(tmpDir)

	inPath := filepath.Join(tmpDir, "input.mp3")
	if err := c.SaveUploadedFile(header, inPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	opts := storage.TrackImportOptions{
		Title:  title,
		Artist: artist,
	}
	if playlistID := c.PostForm("playlist_id"); playlistID != "" {
		if pid, err := strconv.Atoi(playlistID); err == nil {
			opts.PlaylistID = pid
		}
	}

	t, err := storage.ProcessTrack(s.database, s.store, inPath, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, t)
}

func (s *Server) handleUpdateTrack(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var req struct {
		Title  string `json:"title"`
		Artist string `json:"artist"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.database.UpdateTrack(id, req.Title, req.Artist); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) handleDeleteTrack(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	t, err := s.database.GetTrack(id)
	if err == nil {
		_ = s.store.Delete(t.S3Key)
		_ = s.database.DeleteTrack(id)
		if s.engine != nil {
			s.engine.ForgetTracks([]int{id})
		}
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Jingles
func (s *Server) handleGetJingles(c *gin.Context) {
	jingles, err := s.database.GetJingles()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, jingles)
}

func (s *Server) handleUploadJingle(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file missing"})
		return
	}
	defer file.Close()

	name := c.PostForm("name")
	if name == "" {
		name = header.Filename
	}

	tmpDir, err := os.MkdirTemp("", "gostream_jingle")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer os.RemoveAll(tmpDir)

	inPath := filepath.Join(tmpDir, "input.mp3")
	if err := c.SaveUploadedFile(header, inPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	j, err := storage.ProcessJingle(s.database, s.store, inPath, "", name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, j)
}

func (s *Server) handleStreamJingle(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	j, err := s.database.GetJingle(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "jingle not found"})
		return
	}
	reader, err := s.store.GetStream(j.S3Key)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	defer reader.Close()
	c.DataFromReader(http.StatusOK, -1, "audio/mpeg", reader, nil)
}

func (s *Server) handleDeleteJingle(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	j, err := s.database.GetJingle(id)
	if err == nil {
		_ = s.store.Delete(j.S3Key)
		_ = s.database.DeleteJingle(id)
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) handleUpdateJingle(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var req struct {
		Name string `json:"name"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.database.UpdateJingle(id, req.Name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Playlists
func (s *Server) handleGetPlaylists(c *gin.Context) {
	playlists, err := s.database.GetPlaylists()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	type playlistResponse struct {
		db.Playlist
		TrackCount    int `json:"track_count"`
		TotalDuration int `json:"total_duration"`
	}
	out := make([]playlistResponse, 0, len(playlists))
	for _, playlist := range playlists {
		count, duration, _ := s.database.PlaylistTotals(&playlist)
		out = append(out, playlistResponse{Playlist: playlist, TrackCount: count, TotalDuration: duration})
	}
	c.JSON(http.StatusOK, out)
}

func (s *Server) handleCreatePlaylist(c *gin.Context) {
	var req struct {
		Name             string `json:"name"`
		Kind             string `json:"kind"`
		Rules            string `json:"rules"`
		PlaybackMode     string `json:"playback_mode"`
		ArtistSeparation int    `json:"artist_separation"`
		TrackRepeatHours int    `json:"track_repeat_hours"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	p, err := s.database.CreatePlaylist(req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if req.Kind != "" || req.Rules != "" || req.PlaybackMode != "" || req.ArtistSeparation > 0 || req.TrackRepeatHours > 0 {
		if req.Kind != "" {
			p.Kind = req.Kind
		}
		if req.Rules != "" {
			p.Rules = req.Rules
		}
		if req.PlaybackMode != "" {
			p.PlaybackMode = req.PlaybackMode
		}
		if req.ArtistSeparation > 0 {
			p.ArtistSeparation = req.ArtistSeparation
		}
		if req.TrackRepeatHours > 0 {
			p.TrackRepeatHours = req.TrackRepeatHours
		}
		if err := s.database.SavePlaylist(p); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, p)
}

func (s *Server) handleUpdatePlaylist(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		IsActive         *bool   `json:"is_active"`
		Name             *string `json:"name"`
		Kind             *string `json:"kind"`
		Rules            *string `json:"rules"`
		PlaybackMode     *string `json:"playback_mode"`
		ArtistSeparation *int    `json:"artist_separation"`
		TrackRepeatHours *int    `json:"track_repeat_hours"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.IsActive != nil {
		if *req.IsActive {
			s.database.SetActivePlaylist(id)
		} else {
			s.database.SetActivePlaylist(0)
		}
		s.engine.Reload()
	}
	if req.Name != nil || req.Kind != nil || req.Rules != nil || req.PlaybackMode != nil || req.ArtistSeparation != nil || req.TrackRepeatHours != nil {
		playlist, err := s.database.GetPlaylist(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "playlist not found"})
			return
		}
		if req.Name != nil {
			playlist.Name = *req.Name
		}
		if req.Kind != nil {
			playlist.Kind = *req.Kind
		}
		if req.Rules != nil {
			playlist.Rules = *req.Rules
		}
		if req.PlaybackMode != nil {
			playlist.PlaybackMode = *req.PlaybackMode
		}
		if req.ArtistSeparation != nil {
			playlist.ArtistSeparation = *req.ArtistSeparation
		}
		if req.TrackRepeatHours != nil {
			playlist.TrackRepeatHours = *req.TrackRepeatHours
		}
		if err := s.database.SavePlaylist(playlist); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		s.engine.Reload()
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) handleDeletePlaylist(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	s.database.DeletePlaylist(id)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) handleGetPlaylistTracks(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "200"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit <= 0 {
		limit = 200
	}
	if limit > 500 {
		limit = 500
	}
	page, err := s.database.ListPlaylistPage(id, c.Query("q"), offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"tracks":         page.Tracks,
		"total":          page.Total,
		"total_duration": page.TotalDuration,
		"artwork":        page.Artwork,
	})
}

func (s *Server) handleAddPlaylistTrack(c *gin.Context) {
	playlistID, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		TrackID  int `json:"track_id"`
		Position int `json:"position"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := s.database.AddTrackToPlaylist(playlistID, req.TrackID, req.Position)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) handleRemovePlaylistTrack(c *gin.Context) {
	playlistID, _ := strconv.Atoi(c.Param("id"))
	trackID, _ := strconv.Atoi(c.Param("trackId"))
	err := s.database.RemoveTrackFromPlaylist(playlistID, trackID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) handleGetHistory(c *gin.Context) {
	history, err := s.database.GetRecentHistory(10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, history)
}

// Timetable
func (s *Server) handleGetTimetable(c *gin.Context) {
	entries, err := s.database.GetTimetable()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, entries)
}

func (s *Server) handleSaveTimetable(c *gin.Context) {
	var entry db.TimetableEntry
	if err := c.BindJSON(&entry); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.database.SaveTimetableEntry(&entry); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	s.engine.CheckTimetable()

	c.JSON(http.StatusOK, entry)
}

func (s *Server) handleDeleteTimetable(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := s.database.DeleteTimetableEntry(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	s.engine.CheckTimetable()

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) handleGetConfig(c *gin.Context) {
	safeCfg := *s.cfg
	if safeCfg.Database.Password != "" {
		safeCfg.Database.Password = "********"
	}
	if safeCfg.Storage.S3.SecretKey != "" {
		safeCfg.Storage.S3.SecretKey = "********"
	}
	if safeCfg.Icecast.Password != "" && safeCfg.Icecast.Password != "hackme" {
		safeCfg.Icecast.Password = "********"
	}
	if safeCfg.Icecast.AdminPassword != "" && safeCfg.Icecast.AdminPassword != "admin" {
		safeCfg.Icecast.AdminPassword = "********"
	}

	c.JSON(http.StatusOK, safeCfg)
}

func (s *Server) handleSaveConfig(c *gin.Context) {
	var req config.Config
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Database.Password == "********" {
		req.Database.Password = s.cfg.Database.Password
	}
	if req.Storage.S3.SecretKey == "********" {
		req.Storage.S3.SecretKey = s.cfg.Storage.S3.SecretKey
	}
	if req.Icecast.Password == "********" {
		req.Icecast.Password = s.cfg.Icecast.Password
	}
	if req.Icecast.AdminPassword == "********" {
		req.Icecast.AdminPassword = s.cfg.Icecast.AdminPassword
	}

	req.Normalize()

	if err := config.Save(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	*s.cfg = req

	if err := config.ApplyTimezone(s.cfg); err != nil {
		fmt.Printf("Timezone update failed: %v\n", err)
	} else {
		fmt.Printf("Timezone set to %s\n", s.cfg.EffectiveTimezone())
	}

	if newDB, err := db.Connect(s.cfg); err == nil {
		if s.database != nil {
			s.database.Close()
		}
		s.database = newDB
		fmt.Println("Database reconnected successfully")
	} else {
		fmt.Printf("Database reconnection failed: %v\n", err)
	}

	if newStore, err := storage.New(s.cfg); err == nil {
		s.store = newStore
		fmt.Println("Storage reconnected successfully")
		if local, ok := newStore.(*storage.LocalBackend); ok && !local.Writable() {
			fmt.Printf("Warning: tracks path %s is read-only; uploads are disabled\n", local.TracksPath())
		}
	} else {
		fmt.Printf("Storage reconnection failed: %v\n", err)
	}

	if s.database != nil && s.store != nil {
		if s.engine != nil {
			s.engine.Stop()
		}
		s.engine = stream.NewEngine(s.cfg, s.database, s.store)
		s.engine.Start()
		fmt.Printf("Stream engine restarted successfully\n")
	}

	c.JSON(http.StatusOK, gin.H{"status": "saved and applied successfully"})
}

func (s *Server) handleRequestTrack(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	track, err := s.database.GetTrack(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "track not found"})
		return
	}

	s.engine.RequestTrack(*track)
	c.JSON(http.StatusOK, gin.H{"status": "track requested"})
}

func (s *Server) handleGetArtwork(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	reader, err := s.store.GetStream(key)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	defer reader.Close()

	c.DataFromReader(http.StatusOK, -1, "image/jpeg", reader, map[string]string{
		"Cache-Control": "public, max-age=31536000",
	})
}
