package icloud

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var hostPattern = regexp.MustCompile(`^p[0-9]+-sharedstreams\.icloud\.com$`)

type PublicAlbum struct {
	token  string
	host   string
	client *http.Client
	mu     sync.Mutex
}

func NewPublicAlbum(albumURL string) (*PublicAlbum, error) {
	backend, token, err := ParseAlbumURL(albumURL)
	if err != nil {
		return nil, err
	}
	if backend != LegacyBackend {
		return nil, errors.New("expected legacy Shared Album URL")
	}
	partition := 0
	chars := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	part := token[1:3]
	if token[0] == 'A' {
		part = token[1:2]
	}
	for _, c := range part {
		n := strings.IndexRune(chars, c)
		if n < 0 {
			return nil, errors.New("invalid album token")
		}
		partition = partition*62 + n
	}
	return &PublicAlbum{token: token, host: fmt.Sprintf("p%02d-sharedstreams.icloud.com", partition), client: &http.Client{Timeout: 45 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

type streamResponse struct {
	Name   string `json:"streamName"`
	Photos []struct {
		ID          string `json:"photoGuid"`
		MediaType   string `json:"mediaAssetType"`
		Date        string `json:"dateCreated"`
		Derivatives map[string]struct {
			Checksum string      `json:"checksum"`
			Width    json.Number `json:"width"`
			Height   json.Number `json:"height"`
			FileSize json.Number `json:"fileSize"`
		} `json:"derivatives"`
	} `json:"photos"`
}

func (p *PublicAlbum) FetchAlbum(ctx context.Context) (*Album, error) {
	var stream streamResponse
	if err := p.post(ctx, "webstream", map[string]any{"streamCtag": nil}, &stream); err != nil {
		return nil, err
	}
	if stream.Photos == nil {
		return nil, errors.New("iCloud response missing photo list")
	}
	album := &Album{Name: stream.Name, Assets: make([]Asset, 0, len(stream.Photos))}
	for _, photo := range stream.Photos {
		if photo.MediaType == "video" || photo.ID == "" {
			continue
		}
		type derivative struct {
			checksum            string
			width, height, size int
		}
		var choices []derivative
		for _, d := range photo.Derivatives {
			w, _ := strconv.Atoi(d.Width.String())
			h, _ := strconv.Atoi(d.Height.String())
			size, _ := strconv.Atoi(d.FileSize.String())
			if d.Checksum != "" && w > 0 && h > 0 {
				choices = append(choices, derivative{d.Checksum, w, h, size})
			}
		}
		if len(choices) == 0 {
			album.Assets = append(album.Assets, Asset{ID: photo.ID})
			continue
		}
		// Prefer a display-size source and avoid downloading huge originals when available.
		sort.Slice(choices, func(i, j int) bool { return choices[i].width*choices[i].height < choices[j].width*choices[j].height })
		chosen := choices[len(choices)-1]
		for _, d := range choices {
			if max(d.width, d.height) >= 2000 {
				chosen = d
				break
			}
		}
		date, _ := time.Parse(time.RFC3339, photo.Date)
		if date.IsZero() {
			if n, e := strconv.ParseInt(photo.Date, 10, 64); e == nil {
				date = time.Unix(n, 0)
			}
		}
		album.Assets = append(album.Assets, Asset{ID: photo.ID, Version: chosen.checksum, Width: chosen.width, Height: chosen.height, DateTaken: date})
	}
	return album, nil
}

func (p *PublicAlbum) DownloadAsset(ctx context.Context, asset Asset, dst string) error {
	var response struct {
		Items map[string]struct {
			Location string `json:"url_location"`
			Path     string `json:"url_path"`
		} `json:"items"`
	}
	if err := p.post(ctx, "webasseturls", map[string]any{"photoGuids": []string{asset.ID}}, &response); err != nil {
		return err
	}
	item, ok := response.Items[asset.Version]
	if !ok {
		return errors.New("selected image derivative unavailable")
	}
	if item.Location == "" || item.Path == "" {
		return errors.New("invalid asset location")
	}
	location := "https://" + item.Location + item.Path
	return downloadAssetURL(ctx, p.client, location, dst)
}

func (p *PublicAlbum) post(ctx context.Context, endpoint string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	p.mu.Lock()
	host := p.host
	p.mu.Unlock()
	for attempt := 0; attempt < 3; attempt++ {
		u := "https://" + host + "/" + p.token + "/sharedstreams/" + endpoint
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "text/plain")
		req.Header.Set("Origin", "https://www.icloud.com")
		resp, err := p.client.Do(req)
		if err != nil {
			return errors.New("iCloud request failed")
		}
		data, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
		resp.Body.Close()
		if err != nil {
			return errors.New("iCloud response read failed")
		}
		if resp.StatusCode == 330 {
			var redirect map[string]any
			if json.Unmarshal(data, &redirect) != nil {
				return errors.New("invalid iCloud redirect")
			}
			next, _ := redirect["X-Apple-MMe-Host"].(string)
			if !hostPattern.MatchString(next) {
				return errors.New("unexpected iCloud redirect host")
			}
			host = next
			p.mu.Lock()
			p.host = next
			p.mu.Unlock()
			continue
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("iCloud returned HTTP %d", resp.StatusCode)
		}
		if err := json.Unmarshal(data, out); err != nil {
			return errors.New("invalid iCloud response")
		}
		return nil
	}
	return errors.New("too many iCloud redirects")
}
