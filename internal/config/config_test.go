package config

import "testing"

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
