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
	Width        int
	Height       int
	Orientation  string
	Prune        bool
}

func Load() (Config, error) {
	c := Config{AlbumURL: strings.TrimSpace(os.Getenv("ICLOUD_ALBUM_URL")), Port: env("HTTP_PORT", "8080"), DataDir: env("DATA_DIR", "/data"), Orientation: env("DISPLAY_ORIENTATION", "portrait")}
	var err error
	c.SyncInterval, err = time.ParseDuration(env("SYNC_INTERVAL", "15m"))
	if err != nil || c.SyncInterval < time.Minute {
		return c, fmt.Errorf("SYNC_INTERVAL must be at least 1m")
	}
	c.Width, err = strconv.Atoi(env("DISPLAY_WIDTH", "1080"))
	if err != nil || c.Width < 320 || c.Width > 4096 {
		return c, fmt.Errorf("DISPLAY_WIDTH must be between 320 and 4096")
	}
	c.Height, err = strconv.Atoi(env("DISPLAY_HEIGHT", "1920"))
	if err != nil || c.Height < 320 || c.Height > 4096 {
		return c, fmt.Errorf("DISPLAY_HEIGHT must be between 320 and 4096")
	}
	c.Prune, err = strconv.ParseBool(env("PRUNE_REMOVED_PHOTOS", "true"))
	if err != nil {
		return c, fmt.Errorf("invalid PRUNE_REMOVED_PHOTOS")
	}
	if c.Orientation != "portrait" && c.Orientation != "landscape" {
		return c, fmt.Errorf("DISPLAY_ORIENTATION must be portrait or landscape")
	}
	if (c.Orientation == "portrait") != (c.Height > c.Width) {
		return c, fmt.Errorf("display dimensions do not match DISPLAY_ORIENTATION")
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
