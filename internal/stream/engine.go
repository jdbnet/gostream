package stream

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gostream/internal/config"
	"gostream/internal/db"
	"gostream/internal/s3"
)

type Engine struct {
	cfg        *config.Config
	database   *db.DB
	s3Client   *s3.Client
	
	skipChan   chan struct{}
	reloadChan chan struct{}
	stopChan   chan struct{}
	updatePlaylistChan chan struct{}
	
	// internal state
	activePlaylist []db.Track
	currentPlaylistID int
	jingles        []db.Jingle
	playCount      int
	currentTrack   *db.Track // For status API
	currentJingle  *db.Jingle // For status API
	isConnected    bool
	isReconnecting bool
}

func NewEngine(cfg *config.Config, database *db.DB, s3Client *s3.Client) *Engine {
	return &Engine{
		cfg:        cfg,
		database:   database,
		s3Client:   s3Client,
		skipChan:   make(chan struct{}),
		reloadChan: make(chan struct{}),
		stopChan:   make(chan struct{}),
		updatePlaylistChan: make(chan struct{}),
	}
}

func (e *Engine) Start() {
	go e.run()
	go e.timetableWorker()
}

func (e *Engine) timetableWorker() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-e.stopChan:
			return
		case <-ticker.C:
			e.CheckTimetable()
		}
	}
}

func (e *Engine) Stop() {
	select {
	case e.stopChan <- struct{}{}:
	default:
	}
}

func (e *Engine) Skip() {
	select {
	case e.skipChan <- struct{}{}:
	default:
	}
}

func (e *Engine) Reload() {
	select {
	case e.reloadChan <- struct{}{}:
	default:
	}
}

type StreamStatus struct {
	IsConnected    bool       `json:"is_connected"`
	IsReconnecting bool       `json:"is_reconnecting"`
	CurrentTrack   *db.Track  `json:"current_track"`
	CurrentJingle  *db.Jingle `json:"current_jingle"`
}

func (e *Engine) Status() StreamStatus {
	return StreamStatus{
		IsConnected:    e.isConnected,
		IsReconnecting: e.isReconnecting,
		CurrentTrack:   e.currentTrack,
		CurrentJingle:  e.currentJingle,
	}
}

func (e *Engine) evaluateTimetable() int {
	p, err := e.database.GetActivePlaylist()
	if err == nil && p != nil {
		return p.ID
	}

	now := time.Now()
	dayOfWeek := int(now.Weekday())
	currentMinute := now.Hour()*60 + now.Minute()
	
	entries, err := e.database.GetTimetable()
	if err == nil {
		for _, entry := range entries {
			if entry.DayOfWeek == dayOfWeek && currentMinute >= entry.StartMinute && currentMinute < entry.EndMinute {
				return entry.PlaylistID
			}
		}
	}
	
	return 0
}

func (e *Engine) CheckTimetable() {
	select {
	case e.updatePlaylistChan <- struct{}{}:
	default:
	}
}

func (e *Engine) loadMedia() {
	targetID := e.evaluateTimetable()
	e.currentPlaylistID = targetID
	
	if targetID > 0 {
		tracks, _ := e.database.GetPlaylistTracks(targetID)
		
		// Shuffle tracks
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(tracks), func(i, j int) {
			tracks[i], tracks[j] = tracks[j], tracks[i]
		})
		e.activePlaylist = tracks
	} else {
		e.activePlaylist = []db.Track{}
	}

	// load jingles
	jingles, _ := e.database.GetJingles()
	e.jingles = jingles
}

func (e *Engine) updateIcecastMetadata(artist, title string) {
	display := title
	if artist != "" {
		display = artist + " - " + title
	}
	
	protocol := e.cfg.Icecast.Protocol
	if protocol == "" {
		protocol = "http"
	}
	
	apiURL := fmt.Sprintf("%s://%s:%d/admin/metadata?mount=%s&mode=updinfo&song=%s",
		protocol,
		e.cfg.Icecast.Host,
		e.cfg.Icecast.Port,
		e.cfg.Icecast.Mount,
		url.QueryEscape(display),
	)
	
	req, _ := http.NewRequest("GET", apiURL, nil)
	req.SetBasicAuth("admin", e.cfg.Icecast.AdminPassword)
	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		resp.Body.Close()
	} else {
		log.Printf("Failed to update icecast metadata: %v", err)
	}
}

func (e *Engine) run() {
	for {
		e.isReconnecting = true
		e.isConnected = false
		
		conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", e.cfg.Icecast.Host, e.cfg.Icecast.Port))
		if err != nil {
			log.Printf("Failed to connect: %v", err)
			time.Sleep(time.Duration(e.cfg.Stream.ReconnectDelaySeconds) * time.Second)
			continue
		}

		auth := base64.StdEncoding.EncodeToString([]byte("source:" + e.cfg.Icecast.Password))

		handshake := fmt.Sprintf(
			"PUT %s HTTP/1.0\r\n"+
			"Authorization: Basic %s\r\n"+
			"Content-Type: audio/mpeg\r\n"+
			"ice-name: GoStream Radio\r\n"+
			"ice-bitrate: %d\r\n"+
			"ice-channels: %d\r\n"+
			"ice-samplerate: %d\r\n"+
			"\r\n",
			e.cfg.Icecast.Mount, auth,
			e.cfg.Icecast.Bitrate,
			e.cfg.Icecast.Channels,
			e.cfg.Icecast.SampleRate,
		)

		_, err = conn.Write([]byte(handshake))
		if err != nil {
			log.Printf("Handshake failed: %v", err)
			conn.Close()
			time.Sleep(time.Duration(e.cfg.Stream.ReconnectDelaySeconds) * time.Second)
			continue
		}

		// Read Icecast's response
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			log.Printf("Failed to read response: %v", err)
			conn.Close()
			time.Sleep(time.Duration(e.cfg.Stream.ReconnectDelaySeconds) * time.Second)
			continue
		}
		response := string(buf[:n])
		if !strings.Contains(response, "200 OK") {
			log.Printf("Icecast rejected connection: %s", response)
			conn.Close()
			time.Sleep(time.Duration(e.cfg.Stream.ReconnectDelaySeconds) * time.Second)
			continue
		}

		log.Printf("Connected to Icecast")
		e.isConnected = true
		e.isReconnecting = false

		errChan := make(chan error, 1)

		go func() {
			buf := make([]byte, 1)
			_, err := conn.Read(buf)
			if err != nil {
				errChan <- err
			} else {
				errChan <- fmt.Errorf("icecast closed connection")
			}
		}()

		e.loadMedia()

		if len(e.activePlaylist) == 0 {
			log.Printf("Warning: No active playlist or tracks found. Idling...")
		}

		trackIdx := 0
		streamCtx, cancel := context.WithCancel(context.Background())

		frameBuffer := make(chan []byte, 500)
		var nextS3Stream io.ReadCloser

		// Consumer: Reads from frameBuffer and writes to Icecast
		go func() {
			defer conn.Close()
			
			streamStartTime := time.Now()
			var streamBytesWritten int64
			bytesPerSec := float64(e.cfg.Icecast.Bitrate * 1000 / 8)
			
			for {
				select {
				case <-streamCtx.Done():
					return
				case frameData, ok := <-frameBuffer:
					if !ok {
						return // Channel closed
					}
					
					written := 0
					for written < len(frameData) {
						n, err := conn.Write(frameData[written:])
						if err != nil {
							errChan <- err
							return
						}
						streamBytesWritten += int64(n)
						written += n
					}

					// Rate limiting
					expectedDuration := time.Duration(float64(streamBytesWritten) / bytesPerSec * float64(time.Second))
					elapsed := time.Since(streamStartTime)

					// If we starved and fell significantly behind realtime, reset the clock 
					// so we don't aggressively burst frames to Icecast to catch up
					if elapsed > expectedDuration+time.Second {
						streamStartTime = time.Now()
						streamBytesWritten = 0
						expectedDuration = 0
						elapsed = 0
					}

					bufferDuration := time.Duration(e.cfg.Stream.BufferSeconds) * time.Second
					if expectedDuration > elapsed + bufferDuration {
						time.Sleep(expectedDuration - (elapsed + bufferDuration))
					}
				}
			}
		}()

		// Producer: Reads from S3 and writes to frameBuffer
		go func() {
			defer close(frameBuffer)
			
			var currentS3Stream io.ReadCloser
			var s3Reader *bufio.Reader
			
			for {
				select {
				case <-streamCtx.Done():
					if currentS3Stream != nil {
						currentS3Stream.Close()
					}
					if nextS3Stream != nil {
						nextS3Stream.Close()
					}
					return
				case <-e.stopChan:
					return
				case <-e.reloadChan:
					e.loadMedia()
					trackIdx = 0
					if currentS3Stream != nil {
						currentS3Stream.Close()
						currentS3Stream = nil
					}
					if nextS3Stream != nil {
						nextS3Stream.Close()
						nextS3Stream = nil
					}
					e.skipChan <- struct{}{}
				case <-e.updatePlaylistChan:
					e.loadMedia()
					trackIdx = 0
					// Invalidate prefetch so next track comes from new playlist
					if nextS3Stream != nil {
						nextS3Stream.Close()
						nextS3Stream = nil
					}
				default:
				}

				if len(e.activePlaylist) == 0 {
					time.Sleep(1 * time.Second)
					continue
				}

				if currentS3Stream == nil {
					e.playCount++
					playJingle := false
					if e.cfg.Stream.JingleInterval > 0 && e.playCount > e.cfg.Stream.JingleInterval && len(e.jingles) > 0 {
						playJingle = true
						e.playCount = 0
					}

					var s3Key string
					var title string
					var artist string
					var dbTrackID int

					if playJingle {
						j := e.jingles[rand.Intn(len(e.jingles))]
						s3Key = j.S3Key
						title = j.Name
						artist = ""
						e.currentJingle = &j
						e.currentTrack = nil
						e.database.RecordPlay(j.ID, true)
						log.Printf("Playing jingle: %s", title)
						go e.updateIcecastMetadata("", title)
					} else {
						if trackIdx >= len(e.activePlaylist) {
							e.loadMedia()
							trackIdx = 0
						}
						if len(e.activePlaylist) == 0 {
							continue
						}

						t := e.activePlaylist[trackIdx]
						trackIdx++
						s3Key = t.S3Key
						title = t.Title
						artist = t.Artist
						dbTrackID = t.ID
						e.currentTrack = &t
						e.currentJingle = nil
						e.database.IncrementPlayCount(dbTrackID)
						e.database.RecordPlay(dbTrackID, false)
						log.Printf("Playing track: %s - %s", artist, title)
						go e.updateIcecastMetadata(artist, title)
					}

					if nextS3Stream != nil {
						currentS3Stream = nextS3Stream
						nextS3Stream = nil
					} else {
						stream, err := e.s3Client.GetStream(s3Key)
						if err != nil {
							log.Printf("Error getting s3 stream: %v", err)
							time.Sleep(1 * time.Second)
							continue
						}
						currentS3Stream = stream
					}
					s3Reader = bufio.NewReader(currentS3Stream)

					// Kick off next prefetch immediately
					go func(nIdx int, pCount int) {
						if len(e.activePlaylist) == 0 {
							return
						}
						var nKey string
						
						willPlayJingle := false
						if e.cfg.Stream.JingleInterval > 0 && pCount >= e.cfg.Stream.JingleInterval && len(e.jingles) > 0 {
							willPlayJingle = true
						}
						
						if willPlayJingle {
							j := e.jingles[rand.Intn(len(e.jingles))]
							nKey = j.S3Key
						} else {
							if nIdx >= len(e.activePlaylist) {
								nIdx = 0
							}
							nKey = e.activePlaylist[nIdx].S3Key
						}
						
						stream, err := e.s3Client.GetStream(nKey)
						if err == nil {
							nextS3Stream = stream
						}
					}(trackIdx, e.playCount)

					frameLoop:
					for {
						select {
						case <-streamCtx.Done():
							if currentS3Stream != nil {
								currentS3Stream.Close()
								currentS3Stream = nil
							}
							break frameLoop
						case <-e.skipChan:
							if currentS3Stream != nil {
								currentS3Stream.Close()
								currentS3Stream = nil
							}
							if nextS3Stream != nil {
								nextS3Stream.Close()
								nextS3Stream = nil
							}
							break frameLoop
						default:
						}

						if currentS3Stream == nil {
							break frameLoop
						}

						_, frameData, err := FindNextFrame(s3Reader)
						if err != nil {
							if err == io.EOF || err == io.ErrUnexpectedEOF {
								currentS3Stream.Close()
								currentS3Stream = nil
								break frameLoop
							}
							log.Printf("Frame error: %v", err)
							currentS3Stream.Close()
							currentS3Stream = nil
							break frameLoop
						}

						frameBuffer <- frameData
					}
				}
			}
		}()

		// wait for connection to drop
		select {
		case err = <-errChan:
			log.Printf("Icecast connection dropped: %v", err)
		case <-e.stopChan:
			log.Printf("Engine stopped")
			cancel()
			return
		}

		cancel()
		e.isConnected = false
		time.Sleep(time.Duration(e.cfg.Stream.ReconnectDelaySeconds) * time.Second)
	}
}
