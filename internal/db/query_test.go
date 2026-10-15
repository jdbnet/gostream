package db

import (
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func openRawSQLite(t *testing.T, path string) *sqlx.DB {
	t.Helper()
	raw, err := sqlx.Connect("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestTextSearchMode(t *testing.T) {
	if textSearchMode("mysql", "U2") != searchLike {
		t.Fatal("short token should use LIKE")
	}
	if textSearchMode("mysql", "U2 beautiful") != searchLike {
		t.Fatal("mixed short token should use LIKE")
	}
	if textSearchMode("mysql", "The The") != searchFullText {
		t.Fatal("long tokens start on full text")
	}
	if !shouldFallbackToLike("mysql", "The The", searchFullText, 0, nil) {
		t.Fatal("empty full text result should fall back to LIKE")
	}
	if shouldFallbackToLike("mysql", "U2", searchLike, 0, nil) {
		t.Fatal("LIKE queries should not fall back again")
	}
	if shouldFallbackToLike("sqlite", "The The", searchFullText, 0, nil) {
		t.Fatal("sqlite should keep FTS results")
	}
}

func TestFilterSQL(t *testing.T) {
	yearMin := 1990
	yearMax := 1999
	filter := TrackFilter{Query: "U2", Artist: "U2", YearMin: &yearMin, YearMax: &yearMax, Sort: "year"}
	built, err := buildTrackQuery("mysql", filter, searchLike)
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{"LIKE ?", "t.artist = ?", "t.year >= ?", "t.year <= ?", "ORDER BY t.year", "LIMIT ?"} {
		if !strings.Contains(built.selectSQL, part) {
			t.Fatalf("missing %s in %s", part, built.selectSQL)
		}
	}
	if strings.Contains(built.selectSQL, "AGAINST") {
		t.Fatalf("short query used full text: %s", built.selectSQL)
	}
}

func TestPickSQLIsLimited(t *testing.T) {
	query, _ := buildPickSQL("sqlite", PickQuery{PlaylistID: 1, Mode: "weighted", UseArtists: true, ExcludeArtists: []string{"U2"}, UseHistory: true})
	if !strings.Contains(query, "LIMIT 1") {
		t.Fatalf("picker loaded more than one row: %s", query)
	}
	if strings.Contains(query, "SELECT * FROM playlist_tracks") {
		t.Fatal("picker should not load the playlist")
	}
}

func TestBulkFilterExceedsIDCap(t *testing.T) {
	database := openTestDB(t)
	playlist, err := database.CreatePlaylist("All")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < TrackIDCap+5; i++ {
		track := &Track{Title: "Song", Artist: "Bulk", S3Key: "tracks/bulk-" + itoa(i) + ".mp3"}
		if err := database.InsertTrack(track); err != nil {
			t.Fatal(err)
		}
	}
	ids, total, err := database.QueryTrackIDs(TrackFilter{Artist: "Bulk"}, TrackIDCap)
	if err != nil {
		t.Fatal(err)
	}
	if total != TrackIDCap+5 || len(ids) != TrackIDCap {
		t.Fatalf("ids cap total=%d len=%d", total, len(ids))
	}
	added, err := database.AddTracksByFilter(playlist.ID, TrackFilter{Artist: "Bulk"})
	if err != nil {
		t.Fatal(err)
	}
	if added != TrackIDCap+5 {
		t.Fatalf("added %d", added)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
