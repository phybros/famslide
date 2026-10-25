package config

import (
	"testing"
	"time"
)

func TestLoadAcceptsBothPublicAlbumLinks(t *testing.T) {
	for _, link := range []string{
		"https://photos.icloud.com/shared/album/045YeI20-8u3X31bBPD5z9B_A",
		"https://www.icloud.com/sharedalbum/#B125ON9t3mbLNC",
	} {
		t.Setenv("ICLOUD_ALBUM_URL", link)
		config, err := Load()
		if err != nil || config.AlbumURL != link {
			t.Fatalf("loading %s: %v", link, err)
		}
	}
}

func TestPruningDefaultsOnAndCanBeDisabled(t *testing.T) {
	t.Setenv("ICLOUD_ALBUM_URL", "https://www.icloud.com/sharedalbum/#B125ON9t3mbLNC")
	t.Setenv("PRUNE_REMOVED_PHOTOS", "")
	c, err := Load()
	if err != nil || !c.Prune {
		t.Fatalf("default pruning: %#v, %v", c, err)
	}
	t.Setenv("PRUNE_REMOVED_PHOTOS", "false")
	c, err = Load()
	if err != nil || c.Prune {
		t.Fatalf("disabled pruning: %#v, %v", c, err)
	}
}

func TestManualSyncDefaultsOffAndCanBeEnabled(t *testing.T) {
	t.Setenv("ICLOUD_ALBUM_URL", "https://www.icloud.com/sharedalbum/#B125ON9t3mbLNC")
	t.Setenv("ENABLE_MANUAL_SYNC", "")
	c, err := Load()
	if err != nil || c.ManualSync {
		t.Fatalf("default manual sync: %#v, %v", c, err)
	}
	t.Setenv("ENABLE_MANUAL_SYNC", "true")
	c, err = Load()
	if err != nil || !c.ManualSync {
		t.Fatalf("enabled manual sync: %#v, %v", c, err)
	}
	t.Setenv("ENABLE_MANUAL_SYNC", "invalid")
	if _, err := Load(); err == nil {
		t.Fatal("invalid ENABLE_MANUAL_SYNC accepted")
	}
}

func TestSlideshowInterval(t *testing.T) {
	t.Setenv("ICLOUD_ALBUM_URL", "https://www.icloud.com/sharedalbum/#B125ON9t3mbLNC")
	for _, tc := range []struct {
		value string
		want  time.Duration
	}{
		{"", 20 * time.Second}, {"30s", 30 * time.Second}, {"1m", time.Minute}, {"2s", 2 * time.Second}, {"24h", 24 * time.Hour},
	} {
		t.Run(tc.value, func(t *testing.T) {
			t.Setenv("SLIDESHOW_INTERVAL", tc.value)
			c, err := Load()
			if err != nil || c.SlideshowInterval != tc.want {
				t.Fatalf("interval = %v, error = %v", c.SlideshowInterval, err)
			}
		})
	}
	for _, value := range []string{"invalid", "0", "-1s", "1s", "25h"} {
		t.Setenv("SLIDESHOW_INTERVAL", value)
		if _, err := Load(); err == nil {
			t.Fatalf("accepted invalid interval %q", value)
		}
	}
}
