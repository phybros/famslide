package syncer

import (
	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"

	"famslide/internal/icloud"
	"famslide/internal/storage"
)

type fakeSource struct {
	album     *icloud.Album
	downloads int
	fail      bool
}

func (f *fakeSource) FetchAlbum(context.Context) (*icloud.Album, error) {
	if f.fail {
		return nil, errors.New("offline")
	}
	return f.album, nil
}
func (f *fakeSource) DownloadAsset(_ context.Context, _ icloud.Asset, dst string) error {
	f.downloads++
	file, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer file.Close()
	img := image.NewRGBA(image.Rect(0, 0, 80, 120))
	for y := 0; y < 120; y++ {
		for x := 0; x < 80; x++ {
			img.Set(x, y, color.RGBA{120, 40, 90, 255})
		}
	}
	return jpeg.Encode(file, img, nil)
}

func TestIncrementalSyncKeepsCacheOffline(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	fake := &fakeSource{album: &icloud.Album{Name: "Family", Assets: []icloud.Asset{{ID: "one", Version: "v1"}, {ID: "two", Version: "v1"}}}}
	s := New(fake, store, 108, 192, false)
	if err := s.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	first := store.Snapshot()
	if len(first.Photos) != 2 || len(first.Scenes) != 1 || first.Version == "" || fake.downloads != 2 {
		t.Fatalf("unexpected first sync: %#v, downloads=%d", first, fake.downloads)
	}
	if len(first.Scenes[0].Photos) != 2 {
		t.Fatal("pair scene is missing individual photos")
	}
	if entries, err := os.ReadDir(filepath.Join(dir, "scenes")); err != nil || len(entries) != 0 {
		t.Fatalf("unexpected composed files: %d %v", len(entries), err)
	}
	for _, photo := range first.Scenes[0].Photos {
		if photo.URL == "" || photo.Width != 80 || photo.Height != 120 {
			t.Fatalf("bad scene photo: %#v", photo)
		}
	}
	if err := s.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if fake.downloads != 2 || store.Snapshot().Version != first.Version {
		t.Fatal("unchanged photos were rebuilt")
	}
	fake.album.Assets[0].Version = "v2"
	if err := s.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if fake.downloads != 3 || store.Snapshot().Version == first.Version {
		t.Fatal("updated photo did not get a new manifest")
	}
	if _, err := os.Stat(filepath.Join(dir, first.Photos["one"].Original)); !os.IsNotExist(err) {
		t.Fatalf("superseded download remained: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, first.Photos["one"].Display)); err != nil {
		t.Fatal("old playlist photo disappeared:", err)
	}
	updated := store.Snapshot()
	fake.fail = true
	if err := s.Run(context.Background()); err == nil {
		t.Fatal("expected offline error")
	}
	if got := store.Snapshot(); len(got.Photos) != 2 || got.Version != updated.Version {
		t.Fatal("offline sync damaged cache")
	}
	reopened, err := storage.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(reopened.Snapshot().Scenes) != len(first.Scenes) {
		t.Fatal("catalog did not survive restart")
	}
}

func TestPruningRemovesAllCachedMediaForMissingPhotos(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	fake := &fakeSource{album: &icloud.Album{Name: "Family", Assets: []icloud.Asset{
		{ID: "kept", Version: "v1"},
		{ID: "updated", Version: "v1"},
		{ID: "removed", Version: "v1"},
	}}}
	s := New(fake, store, 108, 192, true)
	if err := s.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	before := store.Snapshot()
	for _, path := range []string{"scenes/old.jpg", "display/orphan.jpg", "originals/orphan.img"} {
		if err := os.WriteFile(filepath.Join(dir, path), []byte("old"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	fake.album.Assets = []icloud.Asset{{ID: "kept", Version: "v1"}, {ID: "updated", Version: "v2"}}
	fake.fail = true
	if err := s.Run(context.Background()); err == nil {
		t.Fatal("expected offline error")
	}
	if len(store.Snapshot().Photos) != 3 {
		t.Fatal("offline sync pruned photos")
	}
	fake.fail = false
	if err := s.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	after := store.Snapshot()
	if len(after.Photos) != 2 || len(after.Scenes) != 1 {
		t.Fatalf("removed photo remains in catalog: %#v", after)
	}
	if _, ok := after.Photos["removed"]; ok {
		t.Fatal("removed photo remains in catalog")
	}
	for _, path := range []string{
		before.Photos["removed"].Original,
		before.Photos["removed"].Display,
		before.Photos["updated"].Original,
		before.Photos["updated"].Display,
		"scenes/old.jpg", "display/orphan.jpg", "originals/orphan.img",
	} {
		if _, err := os.Stat(filepath.Join(dir, path)); !os.IsNotExist(err) {
			t.Fatalf("obsolete cache file %s remained: %v", path, err)
		}
	}
	for _, photo := range after.Photos {
		for _, path := range []string{photo.Original, photo.Display} {
			if _, err := os.Stat(filepath.Join(dir, path)); err != nil {
				t.Fatalf("active cache file %s disappeared: %v", path, err)
			}
		}
	}
}
