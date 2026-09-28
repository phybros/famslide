package media

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"famslide/internal/storage"
)

func TestPlansShowEachPhotoOnce(t *testing.T) {
	var photos []storage.Photo
	for i := 0; i < 24; i++ {
		w, h := 80, 120
		if i%2 == 0 {
			w, h = 120, 80
		}
		photos = append(photos, storage.Photo{ID: fmt.Sprint(i), Width: w, Height: h, DateTaken: time.Unix(int64(i), 0)})
	}
	plans := Plans(photos)
	seen := map[string]int{}
	layouts := map[string]bool{}
	for _, plan := range plans {
		layouts[plan.Layout] = true
		for _, p := range plan.Photos {
			seen[p.ID]++
		}
	}
	for _, p := range photos {
		if seen[p.ID] != 1 {
			t.Fatalf("photo %s appears %d times", p.ID, seen[p.ID])
		}
	}
	for _, layout := range []string{"single", "side", "stack", "mixed"} {
		if !layouts[layout] {
			t.Fatalf("missing %s layout", layout)
		}
	}
}

func TestUpgradeOldCatalogWithoutNetwork(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	photo := storage.Photo{ID: "one", Version: "v1", Display: "display/0123456789abcdef01234567.jpg", Width: 120, Height: 80}
	if err := os.WriteFile(filepath.Join(dir, photo.Display), []byte("cached"), 0644); err != nil {
		t.Fatal(err)
	}
	old := storage.Catalog{Photos: map[string]storage.Photo{"one": photo}, Scenes: []storage.Scene{{ID: "old", Image: "/media/old", PhotoIDs: []string{"one"}}}, Version: "old-version"}
	if err := store.Save(old); err != nil {
		t.Fatal(err)
	}
	if err := EnsureDynamicManifest(store); err != nil {
		t.Fatal(err)
	}
	got := store.Snapshot()
	if got.Version == old.Version || len(got.Scenes) != 1 || len(got.Scenes[0].Photos) != 1 || got.Scenes[0].Photos[0].URL != "/media/0123456789abcdef01234567" {
		t.Fatalf("migration failed: %#v", got)
	}
	version := got.Version
	if err := EnsureDynamicManifest(store); err != nil {
		t.Fatal(err)
	}
	if store.Snapshot().Version != version {
		t.Fatal("manifest changed on repeat migration")
	}
}
