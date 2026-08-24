package api

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"

	"gostream/internal/config"
	"gostream/internal/db"
	"gostream/internal/storage"
	"gostream/internal/stream"
)

type Server struct {
	router   *gin.Engine
	cfg      *config.Config
	database *db.DB
	store    storage.Backend
	engine   *stream.Engine
}

func NewServer(cfg *config.Config, database *db.DB, store storage.Backend, engine *stream.Engine, frontendFS embed.FS) *Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	s := &Server{
		router:   r,
		cfg:      cfg,
		database: database,
		store:    store,
		engine:   engine,
	}

	api := r.Group("/api")
	api.Use(func(c *gin.Context) {
		if c.Request.URL.Path != "/api/config" && c.Request.URL.Path != "/api/status" {
			if s.database == nil || s.store == nil {
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "Database or storage not configured. Please save settings."})
				return
			}
		}
		c.Next()
	})
	{
		api.GET("/status", s.handleStatus)
		api.POST("/stream/skip", s.handleSkip)
		api.POST("/stream/reload", s.handleReload)
		api.POST("/stream/request/:id", s.handleRequestTrack)

		api.GET("/artwork", s.handleGetArtwork)

		api.GET("/tracks", s.handleGetTracks)
		api.POST("/tracks", s.handleUploadTrack)
		api.PUT("/tracks/:id", s.handleUpdateTrack)
		api.DELETE("/tracks/:id", s.handleDeleteTrack)

		api.GET("/jingles", s.handleGetJingles)
		api.POST("/jingles", s.handleUploadJingle)
		api.PUT("/jingles/:id", s.handleUpdateJingle)
		api.DELETE("/jingles/:id", s.handleDeleteJingle)

		api.GET("/playlists", s.handleGetPlaylists)
		api.POST("/playlists", s.handleCreatePlaylist)
		api.PUT("/playlists/:id", s.handleUpdatePlaylist)
		api.DELETE("/playlists/:id", s.handleDeletePlaylist)
		api.GET("/playlists/:id/tracks", s.handleGetPlaylistTracks)
		api.POST("/playlists/:id/tracks", s.handleAddPlaylistTrack)
		api.DELETE("/playlists/:id/tracks/:trackId", s.handleRemovePlaylistTrack)

		api.GET("/timetable", s.handleGetTimetable)
		api.POST("/timetable", s.handleSaveTimetable)
		api.DELETE("/timetable/:id", s.handleDeleteTimetable)

		api.GET("/history", s.handleGetHistory)

		api.GET("/config", s.handleGetConfig)
		api.POST("/config", s.handleSaveConfig)
		api.POST("/storage/scan", s.handleStorageScan)
	}

	staticFS, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		panic(err)
	}

	fileServer := http.FileServer(http.FS(staticFS))
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if len(path) >= 4 && path[:4] == "/api" {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		fPath := path
		if fPath == "/" {
			fPath = "index.html"
		} else if fPath != "" && fPath[0] == '/' {
			fPath = fPath[1:]
		}

		_, err := fs.Stat(staticFS, fPath)
		if err == nil {
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}

		c.Request.URL.Path = "/"
		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	return s
}

func (s *Server) Start() error {
	return s.router.Run(fmt.Sprintf(":%d", s.cfg.Server.Port))
}

func (s *Server) handleStatus(c *gin.Context) {
	c.JSON(http.StatusOK, s.engine.Status())
}

func (s *Server) handleSkip(c *gin.Context) {
	s.engine.Skip()
	c.JSON(http.StatusOK, gin.H{"status": "skipped"})
}

func (s *Server) handleReload(c *gin.Context) {
	s.engine.Reload()
	c.JSON(http.StatusOK, gin.H{"status": "reloaded"})
}
