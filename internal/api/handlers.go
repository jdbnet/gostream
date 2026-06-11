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
	if title == "" {
		title = header.Filename
	}

	// Save to temp
	tmpDir, _ := os.MkdirTemp("", "gostream_upload")
	defer os.RemoveAll(tmpDir)

	inPath := filepath.Join(tmpDir, "input.mp3")
	outPath := filepath.Join(tmpDir, "output.mp3")

	inFile, err := os.Create(inPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Copy data to inFile
	// ... (We need an io.Copy here, wait, instead of importing io, I'll just use io.Copy)
	// But let's just use gin's SaveUploadedFile
	inFile.Close()
	if err := c.SaveUploadedFile(header, inPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
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
		DurationSeconds: 0, // Would need ffprobe to get duration, ignoring for now or set to 0
		FileSizeBytes: stat.Size(),
		S3Key:         s3Key,
	}
	if err := s.database.InsertTrack(t); err != nil {
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
	
	if req.IsActive != nil && *req.IsActive {
		s.database.SetActivePlaylist(id)
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

func (s *Server) handleGetConfig(c *gin.Context) {
	// redact passwords
	safeCfg := *s.cfg
	safeCfg.Database.Password = "********"
	safeCfg.S3.SecretKey = "********"
	safeCfg.Icecast.Password = "********"
	
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
	}

	c.JSON(http.StatusOK, gin.H{"status": "saved, restart required for some changes"})
}
