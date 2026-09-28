package syncer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"famslide/internal/icloud"
	"famslide/internal/media"
	"famslide/internal/storage"
)

type State struct {
	Status    string    `json:"sync_status"`
	LastError string    `json:"last_error,omitempty"`
	StartedAt time.Time `json:"sync_started,omitempty"`
}

type Syncer struct {
	source        icloud.AlbumSource
	store         *storage.Store
	width, height int
	prune         bool
	mu            sync.Mutex
	state         State
}

func New(source icloud.AlbumSource, store *storage.Store, width, height int, prune bool) *Syncer {
	return &Syncer{source: source, store: store, width: width, height: height, prune: prune, state: State{Status: "idle"}}
}
func (s *Syncer) State() State { s.mu.Lock(); defer s.mu.Unlock(); return s.state }

func (s *Syncer) Run(ctx context.Context) error {
	s.mu.Lock()
	if s.state.Status == "running" {
		s.mu.Unlock()
		return errors.New("sync already running")
	}
	s.state = State{Status: "running", StartedAt: time.Now()}
	s.mu.Unlock()
	slog.Info("sync started")
	err := s.sync(ctx)
	s.mu.Lock()
	if err != nil {
		s.state = State{Status: "error", LastError: err.Error()}
	} else {
		s.state = State{Status: "idle"}
	}
	s.mu.Unlock()
	if err != nil {
		slog.Error("sync failed", "error", err)
	} else {
		slog.Info("sync completed")
	}
	return err
}

func (s *Syncer) sync(ctx context.Context) error {
	album, err := s.source.FetchAlbum(ctx)
	if err != nil {
		return err
	}
	c := s.store.Snapshot()
	c.AlbumName = album.Name
	seen := make(map[string]bool, len(album.Assets))
	var obsolete []storage.Photo
	added, updated, skipped := 0, 0, 0
	var firstErr error
	for _, asset := range album.Assets {
		if err := ctx.Err(); err != nil {
			return err
		}
		seen[asset.ID] = true
		prior, exists := c.Photos[asset.ID]
		if asset.Version == "" {
			skipped++
			continue
		}
		if exists && prior.Version == asset.Version {
			if _, displayErr := os.Stat(filepath.Join(s.store.Dir(), prior.Display)); displayErr == nil {
				if _, originalErr := os.Stat(filepath.Join(s.store.Dir(), prior.Original)); originalErr == nil {
					skipped++
					continue
				}
			}
		}
		key := storage.Key(asset.ID + asset.Version)
		originalRel := "originals/" + key + ".img"
		displayRel := "display/" + key + ".jpg"
		original := filepath.Join(s.store.Dir(), originalRel)
		display := filepath.Join(s.store.Dir(), displayRel)
		tmp, err := os.CreateTemp(filepath.Join(s.store.Dir(), "originals"), ".download-*")
		if err != nil {
			return err
		}
		tmpName := tmp.Name()
		tmp.Close()
		err = s.source.DownloadAsset(ctx, asset, tmpName)
		if err != nil {
			os.Remove(tmpName)
			if firstErr == nil {
				firstErr = fmt.Errorf("asset download failed: %w", err)
			}
			continue
		}
		w, h, err := media.Derivative(tmpName, display, s.width, s.height)
		if err != nil {
			os.Remove(tmpName)
			if firstErr == nil {
				firstErr = fmt.Errorf("image processing failed: %w", err)
			}
			continue
		}
		if err := os.Rename(tmpName, original); err != nil {
			os.Remove(tmpName)
			return err
		}
		orientation := "landscape"
		if h >= w {
			orientation = "portrait"
		}
		c.Photos[asset.ID] = storage.Photo{ID: asset.ID, Version: asset.Version, Original: originalRel, Display: displayRel, Width: w, Height: h, Orientation: orientation, DateTaken: asset.DateTaken, SyncedAt: time.Now()}
		if exists {
			updated++
			if prior.Version != asset.Version {
				obsolete = append(obsolete, prior)
			}
		} else {
			added++
		}
	}
	if s.prune && firstErr == nil {
		for id := range c.Photos {
			if !seen[id] {
				obsolete = append(obsolete, c.Photos[id])
				delete(c.Photos, id)
			}
		}
	}
	photos := storage.SortedPhotos(c)
	if added > 0 || updated > 0 || len(c.Scenes) == 0 || len(c.Photos) != len(s.store.Snapshot().Photos) || (len(c.Scenes) > 0 && len(c.Scenes[0].Photos) == 0) {
		c.Scenes = media.BuildScenes(photos)
		c.Version = storage.Key(fmt.Sprintf("%d:%d:%s", time.Now().UnixNano(), len(c.Scenes), c.AlbumName))
	}
	if firstErr == nil {
		c.LastSync = time.Now()
		c.LastSyncAdded = added
		c.LastSyncUpdated = updated
	}
	if err := s.store.Save(c); err != nil {
		return err
	}
	if s.prune && firstErr == nil {
		if err := pruneCachedMedia(s.store.Dir(), c); err != nil {
			return err
		}
	} else {
		for _, photo := range obsolete {
			os.Remove(filepath.Join(s.store.Dir(), photo.Original))
		}
	}
	slog.Info("sync result", "remote", len(album.Assets), "added", added, "updated", updated, "skipped", skipped, "scenes", len(c.Scenes))
	return firstErr
}

// pruneCachedMedia removes files that no longer belong to the saved catalog.
// The old scenes directory contains composited JPEGs from earlier versions.
func pruneCachedMedia(dir string, catalog storage.Catalog) error {
	originals := make(map[string]bool, len(catalog.Photos))
	display := make(map[string]bool, len(catalog.Photos))
	for _, photo := range catalog.Photos {
		originals[filepath.Base(photo.Original)] = true
		display[filepath.Base(photo.Display)] = true
	}
	for _, cache := range []struct {
		name string
		keep map[string]bool
	}{
		{name: "originals", keep: originals},
		{name: "display", keep: display},
		{name: "scenes", keep: nil},
	} {
		folder := filepath.Join(dir, cache.name)
		entries, err := os.ReadDir(folder)
		if err != nil {
			return fmt.Errorf("read %s cache: %w", cache.name, err)
		}
		for _, entry := range entries {
			if entry.IsDir() || cache.keep[entry.Name()] {
				continue
			}
			if err := os.Remove(filepath.Join(folder, entry.Name())); err != nil {
				return fmt.Errorf("prune %s cache: %w", cache.name, err)
			}
		}
	}
	return nil
}
