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

Open `http://<server-ip>:8080/` for the slideshow or `http://<server-ip>:8080/admin` for sync status. Photos are cached in `./data`. To pin a release, replace `:latest` with a version tag such as `:v0.0.1`.

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
```

Start it with `docker compose up -d`.

## Development

The repository's `docker-compose.yml` builds from the local source. Set `ICLOUD_ALBUM_URL` in `.env`, then run:

```sh
docker compose up -d --build
go test ./...
node --test web/*.test.cjs
```
