package web

import (
	"context"
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"famslide/internal/storage"
	"famslide/internal/syncer"
)

//go:embed index.html admin.html app.js layout.js playlist.js style.css
var files embed.FS

type Server struct {
	Store      *storage.Store
	Sync       *syncer.Syncer
	ManualSync bool
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	static, _ := fs.Sub(files, ".")
	mux.Handle("GET /app.js", http.FileServer(http.FS(static)))
	mux.Handle("GET /layout.js", http.FileServer(http.FS(static)))
	mux.Handle("GET /playlist.js", http.FileServer(http.FS(static)))
	mux.Handle("GET /style.css", http.FileServer(http.FS(static)))
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		servePage(w, "index.html")
	})
	mux.HandleFunc("GET /admin", func(w http.ResponseWriter, r *http.Request) { servePage(w, "admin.html") })
	mux.HandleFunc("GET /api/status", s.status)
	mux.HandleFunc("GET /api/photos", s.photos)
	mux.HandleFunc("GET /api/scenes", s.scenes)
	if s.ManualSync {
		mux.HandleFunc("POST /api/sync", s.syncNow)
	}
	mux.HandleFunc("GET /media/{id}", s.media)
	return mux
}

func servePage(w http.ResponseWriter, name string) {
	data, err := files.ReadFile(name)
	if err != nil {
		http.Error(w, "page unavailable", 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}
func jsonResponse(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(value)
}

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	c := s.Store.Snapshot()
	state := s.Sync.State()
	var disk int64
	filepath.WalkDir(s.Store.Dir(), func(path string, entry fs.DirEntry, err error) error {
		if err == nil && entry.Type().IsRegular() {
			if info, e := entry.Info(); e == nil {
				disk += info.Size()
			}
		}
		return nil
	})
	jsonResponse(w, map[string]any{"album_name": c.AlbumName, "photo_count": len(c.Photos), "scene_count": len(c.Scenes), "last_sync": c.LastSync, "last_sync_added": c.LastSyncAdded, "last_sync_updated": c.LastSyncUpdated, "sync_status": state.Status, "last_error": state.LastError, "disk_bytes": disk, "manifest_version": c.Version, "manual_sync_enabled": s.ManualSync})
}

func (s *Server) photos(w http.ResponseWriter, r *http.Request) {
	c := s.Store.Snapshot()
	photos := storage.SortedPhotos(c)
	jsonResponse(w, map[string]any{"photos": photos})
}

func (s *Server) scenes(w http.ResponseWriter, r *http.Request) {
	c := s.Store.Snapshot()
	jsonResponse(w, map[string]any{"version": c.Version, "scenes": c.Scenes})
}

func (s *Server) syncNow(w http.ResponseWriter, r *http.Request) {
	if s.Sync.State().Status == "running" {
		http.Error(w, "sync already running", http.StatusConflict)
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		s.Sync.Run(ctx)
	}()
	w.WriteHeader(http.StatusAccepted)
	jsonResponse(w, map[string]string{"status": "started"})
}

func (s *Server) media(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if len(id) != 24 || strings.Trim(id, "0123456789abcdef") != "" {
		http.NotFound(w, r)
		return
	}
	path := filepath.Join(s.Store.Dir(), "display", id+".jpg")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		path = filepath.Join(s.Store.Dir(), "scenes", id+".jpg")
	}
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("Content-Type", "image/jpeg")
	http.ServeFile(w, r, path)
}
