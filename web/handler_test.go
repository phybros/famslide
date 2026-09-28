package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"famslide/internal/icloud"
	"famslide/internal/storage"
	"famslide/internal/syncer"
)

type failingSource struct{}

func (failingSource) FetchAlbum(context.Context) (*icloud.Album, error) {
	return nil, errors.New("offline")
}
func (failingSource) DownloadAsset(context.Context, icloud.Asset, string) error {
	return errors.New("offline")
}

func TestAPIKeepsServingOldSceneAndHidesAlbumURL(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	oldID := "0123456789abcdef01234567"
	if err := os.WriteFile(filepath.Join(dir, "scenes", oldID+".jpg"), []byte("old scene"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(storage.Catalog{AlbumName: "Family", Photos: map[string]storage.Photo{}, Scenes: []storage.Scene{{ID: "fedcba9876543210fedcba98", Image: "/media/fedcba9876543210fedcba98"}}, Version: "v2"}); err != nil {
		t.Fatal(err)
	}
	worker := syncer.New(failingSource{}, store, 108, 192, false)
	handler := (&Server{Store: store, Sync: worker}).Handler()
	r := httptest.NewRecorder()
	handler.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/media/"+oldID, nil))
	if r.Code != 200 || r.Body.String() != "old scene" {
		t.Fatalf("old scene unavailable: %d", r.Code)
	}
	displayID := "abcdef0123456789abcdef01"
	if err := os.WriteFile(filepath.Join(dir, "display", displayID+".jpg"), []byte("individual photo"), 0644); err != nil {
		t.Fatal(err)
	}
	r = httptest.NewRecorder()
	handler.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/media/"+displayID, nil))
	if r.Code != 200 || r.Body.String() != "individual photo" {
		t.Fatalf("photo derivative unavailable: %d", r.Code)
	}
	r = httptest.NewRecorder()
	handler.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/status", nil))
	var status map[string]any
	if err := json.Unmarshal(r.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(r.Body.String(), "icloud.com") {
		t.Fatal("album URL leaked")
	}
	r = httptest.NewRecorder()
	handler.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/sync", nil))
	if r.Code != http.StatusAccepted {
		t.Fatalf("sync request: %d", r.Code)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if worker.State().Status == "error" {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("manual sync did not run after response")
}
