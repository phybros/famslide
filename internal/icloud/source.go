package icloud

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var shareTokenPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,128}$`)

type Backend string

const (
	LegacyBackend   Backend = "sharedstreams"
	CloudKitBackend Backend = "cloudkit"
)

func ParseAlbumURL(raw string) (Backend, string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" {
		return "", "", errors.New("invalid public iCloud Shared Album URL")
	}
	var backend Backend
	var token string
	switch {
	case u.Hostname() == "www.icloud.com" && (u.Path == "/sharedalbum" || strings.HasPrefix(u.Path, "/sharedalbum/")):
		backend = LegacyBackend
		token = strings.Split(u.Fragment, ";")[0]
	case u.Hostname() == "photos.icloud.com" && strings.HasPrefix(u.Path, "/shared/album/"):
		backend = CloudKitBackend
		token = strings.TrimSuffix(strings.TrimPrefix(u.Path, "/shared/album/"), "/")
	default:
		return "", "", errors.New("ICLOUD_ALBUM_URL must be a public Shared Album link from photos.icloud.com or www.icloud.com")
	}
	if !shareTokenPattern.MatchString(token) {
		return "", "", errors.New("invalid public Shared Album token")
	}
	return backend, token, nil
}

func NewSource(raw string) (AlbumSource, error) {
	backend, _, err := ParseAlbumURL(raw)
	if err != nil {
		return nil, err
	}
	if backend == CloudKitBackend {
		return NewCloudKitAlbum(raw)
	}
	return NewPublicAlbum(raw)
}

type Asset struct {
	ID        string
	Version   string
	Width     int
	Height    int
	DateTaken time.Time
}

type Album struct {
	Name   string
	Assets []Asset
}

type AlbumSource interface {
	FetchAlbum(ctx context.Context) (*Album, error)
	DownloadAsset(ctx context.Context, asset Asset, dst string) error
}
