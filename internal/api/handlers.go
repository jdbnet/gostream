package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gostream/internal/config"
	"gostream/internal/db"
	"gostream/internal/s3"
	"gostream/internal/stream"
	"gostream/internal/upload"
)

// Tracks
func (s *Server) handleGetTracks(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit := 50
	offset := (page - 1) * limit
	tracks, err := s.database.GetTracks(offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tracks)
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

	// Save to temp
	tmpDir, _ := os.MkdirTemp("", "gostream_upload")
	defer os.RemoveAll(tmpDir)

	inPath := filepath.Join(tmpDir, "input.mp3")
	outPath := filepath.Join(tmpDir, "output.mp3")

	if err := c.SaveUploadedFile(header, inPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var trackDuration int
	metaTitle, metaArtist, duration, metaErr := upload.ExtractMetadata(inPath)
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
		ext := filepath.Ext(header.Filename)
		title = header.Filename[:len(header.Filename)-len(ext)]
	}

	// Normalize
	if err := upload.NormalizeAudio(inPath, outPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ffmpeg failed: " + err.Error()})
		return
	}

	// Read output stats
	stat, err := os.Stat(outPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	outFile, err := os.Open(outPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer outFile.Close()

	// S3 Key
	s3Key := fmt.Sprintf("tracks/%d_%s", time.Now().UnixNano(), filepath.Base(outPath))
	if err := s.s3.UploadLocalFile(s3Key, outFile, "audio/mpeg"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// insert to db
	t := &db.Track{
		Title:         title,
		Artist:        artist,
		DurationSeconds: trackDuration,
		FileSizeBytes: stat.Size(),
		S3Key:         s3Key,
	}
	if err := s.database.InsertTrack(t); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	playlistID := c.PostForm("playlist_id")
	if playlistID != "" {
		if pid, err := strconv.Atoi(playlistID); err == nil {
			// Append to the end of the playlist
			tracks, _ := s.database.GetPlaylistTracks(pid)
			s.database.AddTrackToPlaylist(pid, t.ID, len(tracks))
		}
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
		s.s3.DeleteFile(t.S3Key)
		s.database.DeleteTrack(id)
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

	tmpDir, _ := os.MkdirTemp("", "gostream_jingle")
	defer os.RemoveAll(tmpDir)

	inPath := filepath.Join(tmpDir, "input.mp3")
	outPath := filepath.Join(tmpDir, "output.mp3")

	if err := c.SaveUploadedFile(header, inPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := upload.NormalizeAudio(inPath, outPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ffmpeg failed: " + err.Error()})
		return
	}
	
	outFile, err := os.Open(outPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer outFile.Close()

	s3Key := fmt.Sprintf("jingles/%d_%s", time.Now().UnixNano(), filepath.Base(outPath))
	if err := s.s3.UploadLocalFile(s3Key, outFile, "audio/mpeg"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	j := &db.Jingle{
		Name:  name,
		S3Key: s3Key,
	}
	if err := s.database.InsertJingle(j); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, j)
}

func (s *Server) handleDeleteJingle(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	j, err := s.database.GetJingle(id)
	if err == nil {
		s.s3.DeleteFile(j.S3Key)
		s.database.DeleteJingle(id)
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
	c.JSON(http.StatusOK, playlists)
}

func (s *Server) handleCreatePlaylist(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
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
	c.JSON(http.StatusOK, p)
}

func (s *Server) handleUpdatePlaylist(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		IsActive *bool `json:"is_active"`
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
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) handleDeletePlaylist(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	s.database.DeletePlaylist(id)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) handleGetPlaylistTracks(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	tracks, err := s.database.GetPlaylistTracks(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tracks)
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
	
	// Refresh engine
	s.engine.CheckTimetable()
	
	c.JSON(http.StatusOK, entry)
}

func (s *Server) handleDeleteTimetable(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := s.database.DeleteTimetableEntry(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	// Refresh engine
	s.engine.CheckTimetable()
	
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) handleGetConfig(c *gin.Context) {
	// redact passwords
	safeCfg := *s.cfg
	if safeCfg.Database.Password != "" {
		safeCfg.Database.Password = "********"
	}
	if safeCfg.S3.SecretKey != "" {
		safeCfg.S3.SecretKey = "********"
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
	
	// Keep passwords if they are ********
	if req.Database.Password == "********" {
		req.Database.Password = s.cfg.Database.Password
	}
	if req.S3.SecretKey == "********" {
		req.S3.SecretKey = s.cfg.S3.SecretKey
	}
	if req.Icecast.Password == "********" {
		req.Icecast.Password = s.cfg.Icecast.Password
	}
	if req.Icecast.AdminPassword == "********" {
		req.Icecast.AdminPassword = s.cfg.Icecast.AdminPassword
	}
	
	if err := config.Save(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	// Update in memory
	*s.cfg = req

	// Attempt to connect/reconnect to DB to create tables immediately
	if newDB, err := db.Connect(s.cfg); err == nil {
		if s.database != nil {
			s.database.Close()
		}
		s.database = newDB
		fmt.Println("Database reconnected successfully")
	} else {
		fmt.Printf("Database reconnection failed: %v\n", err)
	}

	// Attempt to connect/reconnect to S3
	if newS3, err := s3.NewClient(s.cfg); err == nil {
		s.s3 = newS3
		fmt.Println("S3 reconnected successfully")
	} else {
		fmt.Printf("S3 reconnection failed: %v\n", err)
	}

	// Restart Engine with new config
	if s.database != nil && s.s3 != nil {
		if s.engine != nil {
			s.engine.Stop()
		}
		s.engine = stream.NewEngine(s.cfg, s.database, s.s3)
		s.engine.Start()
		fmt.Println("Stream engine restarted successfully")
	}

	c.JSON(http.StatusOK, gin.H{"status": "saved and applied successfully"})
}
