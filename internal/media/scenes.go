package media

import (
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
	var portraits, landscapes []storage.Photo
	for _, p := range photos {
		if p.Height >= p.Width {
			portraits = append(portraits, p)
		} else {
			landscapes = append(landscapes, p)
		}
	}
	mixed := min(len(portraits)/4, len(landscapes)/4)
	for i := 0; i < mixed; i++ {
		plans = append(plans, ScenePlan{[]storage.Photo{portraits[i], landscapes[2*i], landscapes[2*i+1]}, "mixed"})
	}
	portraits, landscapes = portraits[mixed:], landscapes[2*mixed:]
	for i := 0; i < len(portraits); {
		if len(portraits)-i >= 3 {
			plans = append(plans, ScenePlan{[]storage.Photo{portraits[i]}, "single"})
			plans = append(plans, ScenePlan{[]storage.Photo{portraits[i+1], portraits[i+2]}, "side"})
			i += 3
		} else if len(portraits)-i == 2 {
			plans = append(plans, ScenePlan{[]storage.Photo{portraits[i], portraits[i+1]}, "side"})
			i += 2
		} else {
			plans = append(plans, ScenePlan{[]storage.Photo{portraits[i]}, "single"})
			i++
		}
	}
	for i := 0; i < len(landscapes); i += 2 {
		if i+1 < len(landscapes) {
			plans = append(plans, ScenePlan{[]storage.Photo{landscapes[i], landscapes[i+1]}, "stack"})
		} else {
			plans = append(plans, ScenePlan{[]storage.Photo{landscapes[i]}, "single"})
		}
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
			versions = append(versions, p.ID+":"+p.Version+":"+p.Display)
			key := strings.TrimSuffix(filepath.Base(p.Display), ".jpg")
			scenePhotos = append(scenePhotos, storage.ScenePhoto{ID: p.ID, URL: "/media/" + key, Width: p.Width, Height: p.Height})
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
