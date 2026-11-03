package media

import (
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"

	"famslide/internal/storage"
)

type ScenePlan struct {
	Photos []storage.Photo
	Layout string
}

func Plans(photos []storage.Photo) []ScenePlan {
	if len(photos) == 0 {
		return nil
	}
	plans := make([]ScenePlan, 0, len(photos))
	for _, photo := range photos {
		plans = append(plans, ScenePlan{[]storage.Photo{photo}, "single"})
	}
	sort.SliceStable(plans, func(i, j int) bool { return plans[i].Photos[0].DateTaken.Before(plans[j].Photos[0].DateTaken) })
	return plans
}

func BuildScenes(photos []storage.Photo) []storage.Scene {
	plans := Plans(photos)
	scenes := make([]storage.Scene, 0, len(plans))
	for _, plan := range plans {
		ids := make([]string, 0, len(plan.Photos))
		versions := make([]string, 0, len(plan.Photos))
		scenePhotos := make([]storage.ScenePhoto, 0, len(plan.Photos))
		for _, p := range plan.Photos {
			ids = append(ids, p.ID)
			focus, _ := json.Marshal(p.FaceBounds)
			versions = append(versions, p.ID+":"+p.Version+":"+p.Display+":"+string(focus))
			key := strings.TrimSuffix(filepath.Base(p.Display), ".jpg")
			scenePhotos = append(scenePhotos, storage.ScenePhoto{FaceBounds: p.FaceBounds, ID: p.ID, URL: "/media/" + key, Width: p.Width, Height: p.Height})
		}
		id := storage.Key("dynamic-v1:" + plan.Layout + ":" + strings.Join(versions, "|"))
		scenes = append(scenes, storage.Scene{ID: id, PhotoIDs: ids, Photos: scenePhotos, Layout: plan.Layout})
	}
	return scenes
}

// EnsureDynamicManifest upgrades a cache built by the earlier, composited-scene player.
// It needs no iCloud access, so an offline frame can show its existing photos immediately.
func EnsureDynamicManifest(store *storage.Store) error {
	c := store.Snapshot()
	if len(c.Photos) == 0 || (len(c.Scenes) > 0 && len(c.Scenes[0].Photos) > 0) {
		return nil
	}
	c.Scenes = BuildScenes(storage.SortedPhotos(c))
	c.Version = storage.Key("dynamic-v1:" + c.Version)
	return store.Save(c)
}
