package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type Photo struct {
	ID          string    `json:"id"`
	Version     string    `json:"version"`
	Original    string    `json:"original"`
	Display     string    `json:"display"`
	Width       int       `json:"width"`
	Height      int       `json:"height"`
	Orientation string    `json:"orientation"`
	DateTaken   time.Time `json:"date_taken,omitempty"`
	SyncedAt    time.Time `json:"synced_at"`
}

type Scene struct {
	ID       string       `json:"id"`
	PhotoIDs []string     `json:"photo_ids"`
	Photos   []ScenePhoto `json:"photos,omitempty"`
	Image    string       `json:"image,omitempty"`
	Layout   string       `json:"layout"`
}

type ScenePhoto struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type Catalog struct {
	AlbumName       string           `json:"album_name"`
	Photos          map[string]Photo `json:"photos"`
	Scenes          []Scene          `json:"scenes"`
	Version         string           `json:"version"`
	LastSync        time.Time        `json:"last_sync,omitempty"`
	LastSyncAdded   int              `json:"last_sync_added"`
	LastSyncUpdated int              `json:"last_sync_updated"`
}

type Store struct {
	dir     string
	mu      sync.RWMutex
	catalog Catalog
}

func Open(dir string) (*Store, error) {
	for _, sub := range []string{"", "originals", "display", "scenes"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0755); err != nil {
			return nil, err
		}
	}
	s := &Store{dir: dir, catalog: Catalog{Photos: map[string]Photo{}, Scenes: []Scene{}}}
	data, err := os.ReadFile(filepath.Join(dir, "catalog.json"))
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &s.catalog); err != nil {
		return nil, err
	}
	if s.catalog.Photos == nil {
		s.catalog.Photos = map[string]Photo{}
	}
	return s, nil
}

func (s *Store) Dir() string { return s.dir }
func (s *Store) Snapshot() Catalog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c := s.catalog
	c.Photos = make(map[string]Photo, len(s.catalog.Photos))
	for k, v := range s.catalog.Photos {
		c.Photos[k] = v
	}
	c.Scenes = append([]Scene(nil), s.catalog.Scenes...)
	return c
}

func (s *Store) Save(c Catalog) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(s.dir, ".catalog-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err = tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = os.Rename(name, filepath.Join(s.dir, "catalog.json")); err != nil {
		return err
	}
	s.catalog = c
	return nil
}

func Key(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:12])
}

func SortedPhotos(c Catalog) []Photo {
	photos := make([]Photo, 0, len(c.Photos))
	for _, p := range c.Photos {
		photos = append(photos, p)
	}
	sort.Slice(photos, func(i, j int) bool {
		if photos[i].DateTaken.Equal(photos[j].DateTaken) {
			return photos[i].ID < photos[j].ID
		}
		return photos[i].DateTaken.Before(photos[j].DateTaken)
	})
	return photos
}
