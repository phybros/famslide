# Famslide

Famslide syncs a public iCloud Shared Album to local storage and displays it as a fullscreen slideshow.

## Docker

In Photos, enable **Public Website** for a Shared Album and copy its link. Then run:

```sh
docker run -d --name famslide --restart unless-stopped \
  -p 8080:8080 \
  -v "$(pwd)/data:/data" \
  -e 'ICLOUD_ALBUM_URL=https://photos.icloud.com/shared/album/YOUR_PUBLIC_ALBUM_TOKEN' \
  ghcr.io/phybros/famslide:latest
```

Open `http://<server-ip>:8080/` for the slideshow or `http://<server-ip>:8080/admin` for sync status. Photos are cached in `./data`. Sync runs at startup and every 15 minutes by default. To pin a release, replace `:latest` with a version tag such as `:v0.0.1`.

The slideshow adapts to the browser window. Newly cached display images are limited to 1920 pixels on their longest edge. Existing cached images remain until their album photos change.

## Docker Compose

Put your album link in a `.env` file:

```dotenv
ICLOUD_ALBUM_URL=https://photos.icloud.com/shared/album/YOUR_PUBLIC_ALBUM_TOKEN
```

Use this Compose file:

```yaml
services:
  famslide:
    image: ghcr.io/phybros/famslide:latest
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - ./data:/data
    environment:
      ICLOUD_ALBUM_URL: ${ICLOUD_ALBUM_URL}
      ENABLE_MANUAL_SYNC: ${ENABLE_MANUAL_SYNC:-false}
```

Start it with `docker compose up -d`.

## Network access

The examples above publish port 8080 on the host. Anyone who can reach that port can view the slideshow, its cached photos, and `/admin` status. Do not expose it to the internet unless you intend to make those photos public. Use a firewall, private network, or authenticated reverse proxy if access should be restricted.

Manual sync is disabled by default. Setting `ENABLE_MANUAL_SYNC=true` shows the **Sync now** button on `/admin` and enables `POST /api/sync`. This endpoint has no authentication: anyone who can reach it can trigger syncs, consuming network and processing resources. Protect access before enabling it on an untrusted network. For `docker run`, add `-e ENABLE_MANUAL_SYNC=true`; for Compose, set it in `.env`.

## Environment variables

| Variable | Default | Function |
| --- | --- | --- |
| `ICLOUD_ALBUM_URL` | Required | Public iCloud Shared Album link to sync. |
| `SYNC_INTERVAL` | `15m` | Time between automatic syncs; minimum `1m`. |
| `HTTP_PORT` | `8080` | Port the HTTP server listens on. |
| `DATA_DIR` | `/data` | Directory for the photo cache and catalog. |
| `PRUNE_REMOVED_PHOTOS` | `true` | Remove cached photos no longer in the album after a successful sync. |
| `ENABLE_MANUAL_SYNC` | `false` | Show the sync button and enable `POST /api/sync` without authentication. |

## Development

The repository's `docker-compose.yml` builds from the local source. Set `ICLOUD_ALBUM_URL` in `.env`, then run:

```sh
docker compose up -d --build
go test ./...
node --test web/*.test.cjs
```
