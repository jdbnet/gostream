package stream

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
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
	
	// internal state
	activePlaylist []db.Track
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
	}
}

func (e *Engine) Start() {
	go e.run()
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

func (e *Engine) loadMedia() {
	// load playlist
	p, err := e.database.GetActivePlaylist()
	if err == nil && p != nil {
		tracks, _ := e.database.GetPlaylistTracks(p.ID)
		
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

func (e *Engine) run() {
	for {
		e.isReconnecting = true
		e.isConnected = false
		
		// connect to Icecast
		reader, writer := io.Pipe()
		
		req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("http://%s:%d%s", e.cfg.Icecast.Host, e.cfg.Icecast.Port, e.cfg.Icecast.Mount), reader)
		if err != nil {
			log.Printf("Stream error: failed to create request: %v", err)
			time.Sleep(time.Duration(e.cfg.Stream.ReconnectDelaySeconds) * time.Second)
			continue
		}
		
		auth := base64.StdEncoding.EncodeToString([]byte("source:" + e.cfg.Icecast.Password))
		req.Header.Set("Authorization", "Basic "+auth)
		req.Header.Set("Content-Type", "audio/mpeg")
		req.Header.Set("ice-name", "GoStream Radio")
		req.Header.Set("ice-bitrate", fmt.Sprintf("%d", e.cfg.Icecast.Bitrate))
		req.Header.Set("ice-channels", fmt.Sprintf("%d", e.cfg.Icecast.Channels))
		req.Header.Set("ice-samplerate", fmt.Sprintf("%d", e.cfg.Icecast.SampleRate))
		req.Header.Set("Icy-MetaData", "1")

		client := &http.Client{Timeout: 0}
		
		// Run stream loop in a separate goroutine so we can restart the connection
		errChan := make(chan error, 1)
		
		go func() {
			e.isReconnecting = false
			resp, err := client.Do(req)
			if err != nil {
				errChan <- err
				return
			}
			e.isConnected = true
			log.Printf("Connected to Icecast")
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				b, _ := io.ReadAll(resp.Body)
				errChan <- fmt.Errorf("icecast returned %d: %s", resp.StatusCode, string(b))
				return
			}
			errChan <- nil // connection closed cleanly? (unlikely for PUT)
		}()
		
		e.loadMedia()
		
		if len(e.activePlaylist) == 0 {
			log.Printf("Warning: No active playlist or tracks found. Idling...")
		}
		
		icyInterval := (e.cfg.Icecast.Bitrate * 1000 / 8) * 5 // Every 5 seconds roughly? Wait, Icecast usually requests a specific interval, but we are a source. The source doesn't inject metadata inline with MP3 frames to Icecast via PUT.
		// ACTUALLY: Icecast sources using PUT do not inject inline ICY metadata. They use a separate Admin API or the standard metadata update endpoint.
		// Wait, the prompt says: "Icecast supports inline metadata via the Icy-MetaData: 1 request header and periodic metadata blocks in the stream ... Inject StreamTitle='Artist - Title'; at the start of each new track"
		// If using `Icy-MetaData: 1` as source, we *do* inject it. The server sets the interval, but Icecast as a *source* using HTTP PUT? 
		// Actually, standard SHOUTcast source protocol injects. For Icecast PUT, we can use the Icecast Admin metadata API, or if the prompt demands inline, we do inline.
		// "Inject ICY metadata between frames at the configured interval (every icecast.bitrate * 1000 / 8 * 16 bytes) to update now-playing info"
		// Let's follow the prompt strictly: interval = icecast.bitrate * 1000 / 8 * 16 bytes.
		icyInterval = e.cfg.Icecast.Bitrate * 1000 / 8 * 16
		
		bytesSinceMeta := 0
		trackIdx := 0
		
		streamCtx, cancel := context.WithCancel(context.Background())
		
		var currentS3Stream io.ReadCloser
		var s3Reader *bufio.Reader
		
		// The main loop for pushing frames
		go func() {
			defer writer.Close()
			for {
				select {
				case <-streamCtx.Done():
					return
				case <-e.stopChan:
					return
				case <-e.reloadChan:
					e.loadMedia()
					trackIdx = 0
					currentS3Stream = nil // force next track
					e.skipChan <- struct{}{}
				default:
				}
				
				if len(e.activePlaylist) == 0 {
					time.Sleep(1 * time.Second)
					continue
				}
				
				if currentS3Stream == nil {
					// Need to pick next track or jingle
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
					} else {
						if trackIdx >= len(e.activePlaylist) {
							// Re-shuffle
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
					}
					
					stream, err := e.s3Client.GetStream(s3Key)
					if err != nil {
						log.Printf("Error getting s3 stream: %v", err)
						time.Sleep(1 * time.Second)
						continue
					}
					currentS3Stream = stream
					s3Reader = bufio.NewReader(currentS3Stream)
					
					// Inject metadata right away (but we need to wait for byte boundary? 
					// Wait, the prompt says "at the start of each new track", but we must inject at exact `icyInterval` bytes.
					// We'll queue the metadata to be injected at the next boundary.
					metaBytes := BuildIcyMetadata(artist, title)
					// But we also need to start the track. 
					// Let's read frames.
					
					trackStartTime := time.Now()
					var trackBytesWritten int64
					bytesPerSec := float64(e.cfg.Icecast.Bitrate * 1000 / 8)

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
							break frameLoop
						default:
						}
						
						if currentS3Stream == nil {
							break frameLoop
						}
						
						_, frameData, err := FindNextFrame(s3Reader)
						if err != nil {
							if err == io.EOF || err == io.ErrUnexpectedEOF {
								// end of track
								currentS3Stream.Close()
								currentS3Stream = nil
								break frameLoop
							}
							log.Printf("Frame error: %v", err)
							currentS3Stream.Close()
							currentS3Stream = nil
							break frameLoop
						}
						
						// write frame data
						written := 0
						for written < len(frameData) {
							toWrite := len(frameData) - written
							if bytesSinceMeta+toWrite >= icyInterval {
								// hit metadata boundary
								chunk := icyInterval - bytesSinceMeta
								_, err := writer.Write(frameData[written : written+chunk])
								if err != nil {
									currentS3Stream.Close()
									currentS3Stream = nil
									break frameLoop
								}
								trackBytesWritten += int64(chunk)
								written += chunk
								
								// write metadata
								_, err = writer.Write(metaBytes)
								if err != nil {
									currentS3Stream.Close()
									currentS3Stream = nil
									break frameLoop
								}
								metaBytes = []byte{0} // Empty metadata until track changes
								bytesSinceMeta = 0
							} else {
								_, err := writer.Write(frameData[written:])
								if err != nil {
									currentS3Stream.Close()
									currentS3Stream = nil
									break frameLoop
								}
								trackBytesWritten += int64(toWrite)
								bytesSinceMeta += toWrite
								written = len(frameData)
							}
						}
						
						// Rate limiting
						expectedDuration := time.Duration(float64(trackBytesWritten) / bytesPerSec * float64(time.Second))
						elapsed := time.Since(trackStartTime)
						
						bufferDuration := time.Duration(e.cfg.Stream.BufferSeconds) * time.Second
						if expectedDuration > elapsed + bufferDuration {
							time.Sleep(expectedDuration - (elapsed + bufferDuration))
						}
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
