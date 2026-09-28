package icloud

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestPublicAlbumRedirectAndDownload(t *testing.T) {
	source, err := NewPublicAlbum("https://www.icloud.com/sharedalbum/#B125ON9t3mbLNC")
	if err != nil {
		t.Fatal(err)
	}
	redirected := false
	source.client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := ""
		status := 200
		switch {
		case strings.HasSuffix(req.URL.Path, "/webstream") && !redirected:
			redirected = true
			status = 330
			body = `{"X-Apple-MMe-Host":"p42-sharedstreams.icloud.com"}`
		case strings.HasSuffix(req.URL.Path, "/webstream"):
			if req.URL.Host != "p42-sharedstreams.icloud.com" {
				t.Fatalf("wrong redirect host: %s", req.URL.Host)
			}
			body = `{"streamName":"Family","photos":[{"photoGuid":"one","dateCreated":"2026-01-01T00:00:00Z","derivatives":{"2304":{"checksum":"checksum1","width":"1536","height":"2304","fileSize":"100"}}},{"photoGuid":"video","mediaAssetType":"video"}]}`
		case strings.HasSuffix(req.URL.Path, "/webasseturls"):
			body = `{"items":{"checksum1":{"url_location":"cvws.icloud-content.com","url_path":"/photo.jpg"}}}`
		case req.URL.Host == "cvws.icloud-content.com":
			body = "image bytes"
		default:
			t.Fatalf("unexpected request: %s", req.URL.Path)
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: req}, nil
	})
	album, err := source.FetchAlbum(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if album.Name != "Family" || len(album.Assets) != 1 || album.Assets[0].Version != "checksum1" {
		t.Fatalf("bad album: %#v", album)
	}
	dst := filepath.Join(t.TempDir(), "download")
	if err := source.DownloadAsset(context.Background(), album.Assets[0], dst); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(dst)
	if err != nil || string(data) != "image bytes" {
		t.Fatalf("bad download: %q, %v", data, err)
	}
}
