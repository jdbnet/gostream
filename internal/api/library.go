package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gostream/internal/db"
)

type libraryFilter struct {
	Query           string   `json:"q"`
	Artist          string   `json:"artist"`
	Artists         []string `json:"artists"`
	Album           string   `json:"album"`
	Albums          []string `json:"albums"`
	Genre           string   `json:"genre"`
	Genres          []string `json:"genres"`
	YearMin         *int     `json:"year_min"`
	YearMax         *int     `json:"year_max"`
	DurationMin     *int     `json:"duration_min"`
	DurationMax     *int     `json:"duration_max"`
	AddedFrom       string   `json:"added_from"`
	AddedTo         string   `json:"added_to"`
	AddedWithinDays *int     `json:"added_within_days"`
	Sort            string   `json:"sort"`
	Dir             string   `json:"dir"`
	Limit           int      `json:"limit"`
	Offset          int      `json:"offset"`
}

func (f libraryFilter) model() db.TrackFilter {
	filter := db.TrackFilter{
		Query:           f.Query,
		Artist:          f.Artist,
		Artists:         f.Artists,
		Album:           f.Album,
		Albums:          f.Albums,
		Genre:           f.Genre,
		Genres:          f.Genres,
		YearMin:         f.YearMin,
		YearMax:         f.YearMax,
		DurationMin:     f.DurationMin,
		DurationMax:     f.DurationMax,
		AddedWithinDays: f.AddedWithinDays,
		Sort:            f.Sort,
		Limit:           f.Limit,
		Offset:          f.Offset,
	}
	filter.Desc = f.Dir == "desc" || ((f.Sort == "" || f.Sort == "date_added") && f.Dir != "asc")
	if from := parseFilterTime(f.AddedFrom); from != nil {
		filter.AddedFrom = from
	}
	if to := parseFilterTime(f.AddedTo); to != nil {
		filter.AddedTo = to
	}
	return filter
}

func parseFilterTime(value string) *time.Time {
	if value == "" {
		return nil
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return &parsed
	}
	if parsed, err := time.Parse("2006-01-02", value); err == nil {
		return &parsed
	}
	return nil
}

func filterFromQuery(c *gin.Context) libraryFilter {
	filter := libraryFilter{
		Query:  c.Query("q"),
		Artist: c.Query("artist"),
		Album:  c.Query("album"),
		Genre:  c.Query("genre"),
		Sort:   c.Query("sort"),
		Dir:    c.Query("dir"),
		Limit:  queryInt(c, "limit"),
		Offset: queryInt(c, "offset"),
	}
	filter.YearMin = queryIntPtr(c, "year_min")
	filter.YearMax = queryIntPtr(c, "year_max")
	filter.DurationMin = queryIntPtr(c, "duration_min")
	filter.DurationMax = queryIntPtr(c, "duration_max")
	filter.AddedWithinDays = queryIntPtr(c, "added_within_days")
	filter.AddedFrom = c.Query("added_from")
	filter.AddedTo = c.Query("added_to")
	return filter
}

func queryInt(c *gin.Context, key string) int {
	value, _ := strconv.Atoi(c.Query(key))
	return value
}

func queryIntPtr(c *gin.Context, key string) *int {
	raw := c.Query(key)
	if raw == "" {
		return nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return nil
	}
	return &value
}

func (s *Server) handleLibraryTracks(c *gin.Context) {
	filter := filterFromQuery(c).model()
	if filter.Limit <= 0 {
		filter.Limit = 100
	}
	if filter.Limit > 200 {
		filter.Limit = 200
	}
	tracks, total, err := s.database.QueryTracks(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tracks": tracks, "total": total})
}

func (s *Server) handleLibraryArtists(c *gin.Context) {
	filter := filterFromQuery(c).model()
	rows, total, err := s.database.ListArtists(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"artists": rows, "total": total})
}

func (s *Server) handleLibraryAlbums(c *gin.Context) {
	filter := filterFromQuery(c).model()
	rows, total, err := s.database.ListAlbums(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"albums": rows, "total": total})
}

func (s *Server) handleLibraryGenres(c *gin.Context) {
	filter := filterFromQuery(c).model()
	rows, total, err := s.database.ListGenres(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"genres": rows, "total": total})
}

func (s *Server) handleLibraryTrackIDs(c *gin.Context) {
	var body libraryFilter
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ids, total, err := s.database.QueryTrackIDs(body.model(), db.TrackIDCap)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ids": ids, "total": total, "capped": total > len(ids)})
}

func (s *Server) handleBulkAddPlaylistTracks(c *gin.Context) {
	playlistID, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		TrackIDs []int          `json:"track_ids"`
		Filter   *libraryFilter `json:"filter"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var added int
	var err error
	if req.Filter != nil {
		added, err = s.database.AddTracksByFilter(playlistID, req.Filter.model())
	} else if len(req.TrackIDs) > 0 {
		added, err = s.database.AddTracksToPlaylist(playlistID, req.TrackIDs)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "track_ids or filter required"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"added": added})
}

func (s *Server) handleBulkRemovePlaylistTracks(c *gin.Context) {
	playlistID, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		TrackIDs []int `json:"track_ids"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.database.RemoveTracksFromPlaylist(playlistID, req.TrackIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) handleReorderPlaylistTracks(c *gin.Context) {
	playlistID, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		TrackIDs []int `json:"track_ids"`
		TrackID  int   `json:"track_id"`
		ToIndex  int   `json:"to_index"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var err error
	if req.TrackID > 0 {
		err = s.database.MovePlaylistTrack(playlistID, req.TrackID, req.ToIndex)
	} else {
		err = s.database.ReorderPlaylist(playlistID, req.TrackIDs)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
