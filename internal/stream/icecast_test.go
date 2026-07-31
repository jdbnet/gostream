package stream

import (
	"net"
	"strings"
	"testing"

	"gostream/internal/config"
)

func TestWaitForIcecastHandshake100Continue(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	go func() {
		_, _ = server.Write([]byte("HTTP/1.1 100 Continue\r\nServer: Icecast 2.5.0\r\n\r\n"))
	}()

	if err := waitForIcecastHandshake(client); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestWaitForIcecastHandshake200OK(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	go func() {
		_, _ = server.Write([]byte("HTTP/1.0 200 OK\r\nServer: Icecast 2.4.4\r\n\r\n"))
	}()

	if err := waitForIcecastHandshake(client); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestWaitForIcecastHandshake401(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	go func() {
		_, _ = server.Write([]byte("HTTP/1.1 401 Authentication Required\r\n\r\n"))
	}()

	if err := waitForIcecastHandshake(client); err == nil {
		t.Fatal("expected authentication error")
	}
}

func TestBuildIcecastHandshakeIncludesExpectContinue(t *testing.T) {
	cfg := &config.Config{}
	cfg.Icecast.Host = "icecast.example.com"
	cfg.Icecast.Port = 8000
	cfg.Icecast.Mount = "/gostream"
	cfg.Icecast.Password = "secret"
	cfg.Icecast.Bitrate = 192
	cfg.Icecast.Channels = 2
	cfg.Icecast.SampleRate = 44100

	handshake := buildIcecastHandshake(cfg)

	for _, want := range []string{
		"PUT /gostream HTTP/1.1",
		"Host: icecast.example.com:8000",
		"Expect: 100-continue",
		"Content-Type: audio/mpeg",
		"Ice-Bitrate: 192",
	} {
		if !strings.Contains(handshake, want) {
			t.Fatalf("handshake missing %q:\n%s", want, handshake)
		}
	}
}
