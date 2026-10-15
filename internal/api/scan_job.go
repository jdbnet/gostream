package api

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"gostream/internal/storage"
)

type scanJob struct {
	mu       sync.Mutex
	progress storage.ScanProgress
}

func (s *Server) startLibraryJob(refresh bool) bool {
	s.scans.mu.Lock()
	if s.scans.progress.Running {
		s.scans.mu.Unlock()
		return false
	}
	s.scans.progress = storage.ScanProgress{Running: true, Phase: "starting"}
	s.scans.mu.Unlock()

	go func() {
		report := func(progress storage.ScanProgress) {
			s.scans.mu.Lock()
			s.scans.progress = progress
			s.scans.mu.Unlock()
		}
		var result *storage.ScanResult
		var err error
		if refresh {
			result, err = storage.RefreshMetadata(s.database, s.store, s.cfg, report)
		} else {
			result, err = storage.ScanLibrary(s.database, s.store, s.cfg, report)
		}
		if result != nil && s.engine != nil {
			s.engine.ForgetTracks(result.RemovedIDs)
		}
		s.scans.mu.Lock()
		defer s.scans.mu.Unlock()
		if result != nil {
			s.scans.progress = result.Progress("done", result.Imported+result.Updated+result.Skipped, result.Imported+result.Updated+result.Skipped, false)
		}
		s.scans.progress.Running = false
		if err != nil {
			s.scans.progress.Phase = "error"
			s.scans.progress.Errors = append(s.scans.progress.Errors, err.Error())
		}
	}()
	return true
}

func (s *Server) scanSnapshot() storage.ScanProgress {
	s.scans.mu.Lock()
	defer s.scans.mu.Unlock()
	return s.scans.progress
}

func (s *Server) handleStorageScan(c *gin.Context) {
	if !s.startLibraryJob(false) {
		c.JSON(http.StatusConflict, s.scanSnapshot())
		return
	}
	c.JSON(http.StatusAccepted, s.scanSnapshot())
}

func (s *Server) handleStorageScanStatus(c *gin.Context) {
	c.JSON(http.StatusOK, s.scanSnapshot())
}

func (s *Server) handleRefreshMetadata(c *gin.Context) {
	if !s.startLibraryJob(true) {
		c.JSON(http.StatusConflict, s.scanSnapshot())
		return
	}
	c.JSON(http.StatusAccepted, s.scanSnapshot())
}
