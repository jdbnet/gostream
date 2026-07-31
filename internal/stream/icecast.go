package stream

import (
	"encoding/base64"
	"fmt"
	"net"
	"strings"
	"time"

	"gostream/internal/config"
)

const icecastHandshakeTimeout = 15 * time.Second

// connectIcecastSource opens a TCP connection to Icecast and performs the
// HTTP PUT source handshake. Icecast 2.5+ requires Expect: 100-continue; without
// it the server sends no response and expects the body immediately, which caused
// the previous HTTP/1.0 client to block on Read until source-timeout fired.
func connectIcecastSource(cfg *config.Config) (net.Conn, error) {
	addr := fmt.Sprintf("%s:%d", cfg.Icecast.Host, cfg.Icecast.Port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}

	if err := conn.SetDeadline(time.Now().Add(icecastHandshakeTimeout)); err != nil {
		conn.Close()
		return nil, err
	}

	handshake := buildIcecastHandshake(cfg)
	if _, err := conn.Write([]byte(handshake)); err != nil {
		conn.Close()
		return nil, fmt.Errorf("write handshake: %w", err)
	}

	if err := waitForIcecastHandshake(conn); err != nil {
		conn.Close()
		return nil, err
	}

	if err := conn.SetDeadline(time.Time{}); err != nil {
		conn.Close()
		return nil, err
	}

	if tc, ok := conn.(*net.TCPConn); ok {
		_ = tc.SetKeepAlive(true)
		_ = tc.SetKeepAlivePeriod(30 * time.Second)
	}

	return conn, nil
}

func buildIcecastHandshake(cfg *config.Config) string {
	host := cfg.Icecast.Host
	if cfg.Icecast.Port != 80 && cfg.Icecast.Port != 443 {
		host = fmt.Sprintf("%s:%d", cfg.Icecast.Host, cfg.Icecast.Port)
	}

	auth := base64.StdEncoding.EncodeToString([]byte("source:" + cfg.Icecast.Password))

	return fmt.Sprintf(
		"PUT %s HTTP/1.1\r\n"+
			"Host: %s\r\n"+
			"Authorization: Basic %s\r\n"+
			"User-Agent: GoStream/1.0\r\n"+
			"Accept: */*\r\n"+
			"Content-Type: audio/mpeg\r\n"+
			"Expect: 100-continue\r\n"+
			"Ice-Name: GoStream Radio\r\n"+
			"Ice-Bitrate: %d\r\n"+
			"Ice-Channels: %d\r\n"+
			"Ice-Samplerate: %d\r\n"+
			"\r\n",
		cfg.Icecast.Mount,
		host,
		auth,
		cfg.Icecast.Bitrate,
		cfg.Icecast.Channels,
		cfg.Icecast.SampleRate,
	)
}

func waitForIcecastHandshake(conn net.Conn) error {
	buf := make([]byte, 1024)
	var received strings.Builder

	for received.Len() < 8192 {
		n, err := conn.Read(buf)
		if n > 0 {
			received.Write(buf[:n])
			body := received.String()

			if strings.Contains(body, "100 Continue") {
				return nil
			}
			if strings.Contains(body, "200 OK") {
				return nil
			}
			if strings.Contains(body, "401") {
				return fmt.Errorf("authentication failed: %s", firstStatusLine(body))
			}
			if strings.Contains(body, "403") {
				return fmt.Errorf("forbidden: %s", firstStatusLine(body))
			}
			if strings.Contains(body, " 4") || strings.Contains(body, " 5") {
				if strings.HasPrefix(body, "HTTP/") {
					return fmt.Errorf("icecast rejected connection: %s", firstStatusLine(body))
				}
			}
		}
		if err != nil {
			return fmt.Errorf("read handshake response: %w", err)
		}
	}

	return fmt.Errorf("icecast handshake response incomplete")
}

func firstStatusLine(response string) string {
	if line, _, ok := strings.Cut(response, "\n"); ok {
		return strings.TrimSpace(line)
	}
	return strings.TrimSpace(response)
}
