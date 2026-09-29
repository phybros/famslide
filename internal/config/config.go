package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"famslide/internal/icloud"
)

type Config struct {
	AlbumURL     string
	SyncInterval time.Duration
	Port         string
	DataDir      string
	Prune        bool
	ManualSync   bool
}

func Load() (Config, error) {
	c := Config{AlbumURL: strings.TrimSpace(os.Getenv("ICLOUD_ALBUM_URL")), Port: env("HTTP_PORT", "8080"), DataDir: env("DATA_DIR", "/data")}
	var err error
	c.SyncInterval, err = time.ParseDuration(env("SYNC_INTERVAL", "15m"))
	if err != nil || c.SyncInterval < time.Minute {
		return c, fmt.Errorf("SYNC_INTERVAL must be at least 1m")
	}
	c.Prune, err = strconv.ParseBool(env("PRUNE_REMOVED_PHOTOS", "true"))
	if err != nil {
		return c, fmt.Errorf("invalid PRUNE_REMOVED_PHOTOS")
	}
	c.ManualSync, err = strconv.ParseBool(env("ENABLE_MANUAL_SYNC", "false"))
	if err != nil {
		return c, fmt.Errorf("invalid ENABLE_MANUAL_SYNC")
	}
	if c.AlbumURL == "" {
		return c, fmt.Errorf("ICLOUD_ALBUM_URL is required")
	}
	if _, _, err := icloud.ParseAlbumURL(c.AlbumURL); err != nil {
		return c, err
	}
	if _, err := strconv.Atoi(c.Port); err != nil {
		return c, fmt.Errorf("HTTP_PORT must be a number")
	}
	return c, nil
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
