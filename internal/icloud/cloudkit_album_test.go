package icloud

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAlbumURLBackends(t *testing.T) {
	for _, tc := range []struct {
		url     string
		backend Backend
		token   string
	}{
		{"https://www.icloud.com/sharedalbum/#B125ON9t3mbLNC", LegacyBackend, "B125ON9t3mbLNC"},
		{"https://photos.icloud.com/shared/album/045YeI20-8u3X31bBPD5z9B_A", CloudKitBackend, "045YeI20-8u3X31bBPD5z9B_A"},
		{"https://photos.icloud.com/shared/album/045YeI20-8u3X31bBPD5z9B_A?i=photo", CloudKitBackend, "045YeI20-8u3X31bBPD5z9B_A"},
	} {
		backend, token, err := ParseAlbumURL(tc.url)
		if err != nil || backend != tc.backend || token != tc.token {
			t.Fatalf("parse %s: %s %s %v", tc.url, backend, token, err)
		}
		source, err := NewSource(tc.url)
		if err != nil {
			t.Fatal(err)
		}
		if tc.backend == CloudKitBackend {
			if _, ok := source.(*CloudKitAlbum); !ok {
				t.Fatal("new link selected wrong source")
			}
		}
	}
	for _, raw := range []string{"https://evil.com/shared/album/045YeI20-8u3X31bBPD5z9B_A", "https://photos.icloud.com/shared/album/", "https://photos.icloud.com/shared/album/a/b"} {
		if _, _, err := ParseAlbumURL(raw); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
}

func TestCloudKitPartitionAcceptsDefaultHTTPSPort(t *testing.T) {
	for _, raw := range []string{"https://p42-ckdatabasews.icloud.com", "https://p42-ckdatabasews.icloud.com:443/"} {
		base, err := normalizeCloudKitPartition(raw)
		if err != nil || base != "https://p42-ckdatabasews.icloud.com" {
			t.Fatalf("normalize %s: %s %v", raw, base, err)
		}
	}
	if base, err := normalizeCloudKitPartition("https://ckdatabasews.icloud.com:443"); err != nil || base != "https://ckdatabasews.icloud.com" {
		t.Fatalf("root partition: %s %v", base, err)
	}
	for _, raw := range []string{"http://p42-ckdatabasews.icloud.com", "https://p42-ckdatabasews.icloud.com:444", "https://evil.com", "https://p42-ckdatabasews.icloud.com/other"} {
		if _, err := normalizeCloudKitPartition(raw); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
}

func TestCloudKitAlbumResolveQueryAndDownload(t *testing.T) {
	source, err := NewCloudKitAlbum("https://photos.icloud.com/shared/album/045YeI20-8u3X31bBPD5z9B_A")
	if err != nil {
		t.Fatal(err)
	}
	queries := 0
	source.client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := ""
		status := 200
		switch {
		case strings.HasSuffix(req.URL.Path, "/public/records/resolve"):
			body = `{"results":[{"zoneID":{"zoneName":"SharedZone","ownerRecordName":"owner"},"anonymousPublicAccess":{"token":"anonymous","databasePartition":"https://p42-ckdatabasews.icloud.com:443/"},"share":{"fields":{"cloudkit.title":{"value":"Family"}}}}]}`
		case strings.HasSuffix(req.URL.Path, "/shared/records/query"):
			if req.URL.Query().Get("publicAccessAuthToken") != "anonymous" || req.URL.Query().Get("sharing_url_key") != "045YeI20-8u3X31bBPD5z9B_A" {
				t.Fatal("missing anonymous access")
			}
			data, _ := io.ReadAll(req.Body)
			var request map[string]any
			json.Unmarshal(data, &request)
			queries++
			if queries == 1 {
				if request["continuationMarker"] != nil {
					t.Fatal("unexpected first marker")
				}
				body = `{"records":[{"recordType":"CPLMaster","recordName":"MASTER1","recordChangeTag":"v1","fields":{"itemType":{"value":"public.heic"},"filenameEnc":{"value":"aW1hZ2UuanBn"},"resJPEGMedRes":{"value":{"downloadURL":"https://cvws.icloud-content.com/${f}?sig=temporary"}},"resJPEGMedWidth":{"value":1024},"resJPEGMedHeight":{"value":768},"resJPEGMedFingerprint":{"value":"stable-image"}}}],"continuationMarker":"next"}`
			} else {
				if request["continuationMarker"] != "next" {
					t.Fatal("missing continuation")
				}
				body = `{"records":[{"recordType":"CPLAsset","recordName":"ASSET1","fields":{"masterRef":{"value":{"recordName":"MASTER1"}},"assetDate":{"value":1767225600000}}}]}`
			}
		case req.URL.Host == "cvws.icloud-content.com":
			if req.URL.EscapedPath() != "/image.jpg" {
				t.Fatalf("wrong filename: %s", req.URL.EscapedPath())
			}
			body = "image bytes"
		default:
			t.Fatalf("unexpected request: %s", req.URL.String())
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: req}, nil
	})
	album, err := source.FetchAlbum(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if album.Name != "Family" || len(album.Assets) != 1 || album.Assets[0].ID != "MASTER1" || album.Assets[0].Version != "resJPEGMed:stable-image" || album.Assets[0].DateTaken != time.UnixMilli(1767225600000) {
		t.Fatalf("unexpected album: %#v", album)
	}
	if queries != 2 {
		t.Fatalf("got %d query pages", queries)
	}
	dst := filepath.Join(t.TempDir(), "download")
	if err := source.DownloadAsset(context.Background(), album.Assets[0], dst); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(dst)
	if err != nil || string(data) != "image bytes" {
		t.Fatalf("download: %q %v", data, err)
	}
	if _, err := url.Parse(source.urls["MASTER1"]); err != nil {
		t.Fatal(err)
	}
}
