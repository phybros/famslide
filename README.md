# Famslide

Famslide caches one public iCloud Shared Album and serves a fullscreen slideshow from local images. A Raspberry Pi can keep displaying photos when iCloud is unavailable. The browser receives optimized JPEGs and arranges individual photos for its current viewport, with gentle motion and fades. It never receives the album URL.

## Run with Docker

1. In Photos, enable **Public Website** for a Shared Album and copy its link. Both current `https://photos.icloud.com/shared/album/...` and older `https://www.icloud.com/sharedalbum/#...` links are supported.
2. Copy `.env.example` to `.env` and replace the example URL with your public album link. Treat this file as a secret.
3. Run `docker compose up -d --build`.
4. Open `http://<server-ip>:8080/` for the slideshow or `http://<server-ip>:8080/admin` for status and manual sync.

The first sync downloads the album and builds the display images, so a large album may take a while. The slideshow starts as soon as the first catalog is ready. Data persists in `./data` and survives container restarts. Existing caches are upgraded to the responsive scene manifest on startup without downloading photos again. Each browser plays every photo once per pass, then reshuffles and regroups them for the next pass. With only two photos, passes alternate between a pair and individual slides.

After a successful sync, photos removed from the album leave the slideshow catalog and their cached originals and display JPEGs are deleted. Old composed scene files are cleaned up too. Set `PRUNE_REMOVED_PHOTOS=false` to retain photos removed from the album.

## Configuration

| Variable | Default | Purpose |
| --- | --- | --- |
| `ICLOUD_ALBUM_URL` | required | Public Shared Album URL |
| `SYNC_INTERVAL` | `15m` | Time between sync attempts (minimum `1m`) |
| `HTTP_PORT` | `8080` | HTTP listener port |
| `DATA_DIR` | `/data` | Persistent cache directory |
| `DISPLAY_WIDTH` | `1080` | Maximum width of each display JPEG |
| `DISPLAY_HEIGHT` | `1920` | Maximum height of each display JPEG |
| `DISPLAY_ORIENTATION` | `portrait` | Validates the configured derivative dimensions; layouts follow the browser viewport |
| `PRUNE_REMOVED_PHOTOS` | `true` | Remove deleted album photos and their cached files after a successful sync |

The application never requires Apple credentials. Anyone with the public album link can access that album; keep the link and `.env` private. The HTTP server is intended for a trusted local network and has no user authentication.

## API

`GET /api/status` reports album, sync, and cache details. `GET /api/photos` lists cached photo metadata. `GET /api/scenes` returns a versioned manifest of photo groups and individual image URLs. `POST /api/sync` starts a sync. `GET /media/{photo-id}` serves one cached display JPEG. A browser adopts a new manifest at the next scene boundary after its periodic refresh.

## Development

Run `go test ./...`, `node --test web/*.test.cjs`, and `go run ./cmd/server` with `ICLOUD_ALBUM_URL` and `DATA_DIR` set. Apple uses two unofficial public album backends: shared streams for older links and anonymous CloudKit for current links. The `AlbumSource` interface in `internal/icloud` isolates those dependencies. Metadata lives in an atomically replaced JSON catalog; original and display photos are separate files. Failed syncs leave the last working manifest available.
