package icloud

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

const cloudKitRoot = "https://ckdatabasews.icloud.com/database/1/com.apple.photos.cloud/production"
const cloudKitRecordType = "CPLAssetAndMasterByAssetDateWithoutHiddenOrDeleted"

var cloudKitPartition = regexp.MustCompile(`^p[0-9]+-ckdatabasews\.icloud\.com$`)

type CloudKitAlbum struct {
	token  string
	client *http.Client
	mu     sync.RWMutex
	urls   map[string]string
}

type ckField struct {
	Value json.RawMessage `json:"value"`
}
type ckRecord struct {
	Name      string             `json:"recordName"`
	Type      string             `json:"recordType"`
	ChangeTag string             `json:"recordChangeTag"`
	Fields    map[string]ckField `json:"fields"`
}

func NewCloudKitAlbum(raw string) (*CloudKitAlbum, error) {
	backend, token, err := ParseAlbumURL(raw)
	if err != nil {
		return nil, err
	}
	if backend != CloudKitBackend {
		return nil, errors.New("expected photos.icloud.com Shared Album URL")
	}
	return &CloudKitAlbum{token: token, client: &http.Client{Timeout: 45 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

func (c *CloudKitAlbum) FetchAlbum(ctx context.Context) (*Album, error) {
	resolveURL := cloudKitRoot + "/public/records/resolve?ckjsBuildVersion=2626&shortGUID=" + url.QueryEscape(c.token)
	var resolved struct {
		Results []struct {
			Zone   json.RawMessage `json:"zoneID"`
			Access struct {
				Token     string `json:"token"`
				Partition string `json:"databasePartition"`
			} `json:"anonymousPublicAccess"`
			Share ckRecord `json:"share"`
		} `json:"results"`
	}
	if err := c.post(ctx, resolveURL, map[string]any{"shortGUIDs": []map[string]string{{"value": c.token}}}, &resolved); err != nil {
		return nil, err
	}
	if len(resolved.Results) == 0 {
		return nil, errors.New("public Shared Album link did not resolve")
	}
	result := resolved.Results[0]
	if len(result.Zone) == 0 || result.Access.Token == "" {
		return nil, errors.New("Shared Album is not publicly accessible")
	}
	partitionBase, err := normalizeCloudKitPartition(result.Access.Partition)
	if err != nil {
		return nil, fmt.Errorf("unexpected iCloud partition: %w", err)
	}
	name := fieldString(result.Share.Fields, "cloudkit.title")
	if name == "" {
		name = "Shared Album"
	}
	query := url.Values{"remapEnums": {"true"}, "getCurrentSyncToken": {"true"}, "sharing_url_key": {c.token}, "publicAccessAuthToken": {result.Access.Token}}
	queryURL := partitionBase + "/database/1/com.apple.photos.cloud/production/shared/records/query?" + query.Encode()
	var records []ckRecord
	continuation := ""
	for page := 0; page < 100; page++ {
		body := map[string]any{"zoneID": json.RawMessage(result.Zone), "query": map[string]any{"recordType": cloudKitRecordType}, "resultsLimit": 200}
		if continuation != "" {
			body["continuationMarker"] = continuation
		}
		var response struct {
			Records      []ckRecord `json:"records"`
			Continuation string     `json:"continuationMarker"`
		}
		if err := c.post(ctx, queryURL, body, &response); err != nil {
			return nil, err
		}
		if response.Records == nil {
			return nil, errors.New("iCloud query response missing records")
		}
		records = append(records, response.Records...)
		if response.Continuation == "" {
			break
		}
		if response.Continuation == continuation || page == 99 {
			return nil, errors.New("iCloud query pagination failed")
		}
		continuation = response.Continuation
	}
	album := &Album{Name: name, Assets: make([]Asset, 0, len(records)/2)}
	masters := make(map[string]ckRecord)
	for _, record := range records {
		if record.Type == "CPLMaster" {
			masters[record.Name] = record
		}
	}
	urls := make(map[string]string)
	for _, record := range records {
		if record.Type != "CPLAsset" || fieldBool(record.Fields, "isHidden") || fieldBool(record.Fields, "trashReason") {
			continue
		}
		var ref struct {
			RecordName string `json:"recordName"`
		}
		if json.Unmarshal(record.Fields["masterRef"].Value, &ref) != nil {
			continue
		}
		master, ok := masters[ref.RecordName]
		if !ok {
			continue
		}
		itemType := strings.ToLower(fieldString(master.Fields, "itemType"))
		if strings.Contains(itemType, "video") || strings.Contains(itemType, "movie") || strings.Contains(itemType, "quicktime") || strings.Contains(itemType, "mpeg-4") {
			continue
		}
		filename := "image"
		if encoded := fieldString(master.Fields, "filenameEnc"); encoded != "" {
			if data, err := base64.StdEncoding.DecodeString(encoded); err == nil && len(data) > 0 {
				filename = string(data)
			}
		}
		asset, location, ok := cloudKitAsset(master, record, filename)
		if !ok {
			continue
		}
		album.Assets = append(album.Assets, asset)
		urls[asset.ID] = location
	}
	if len(records) > 0 && len(album.Assets) == 0 {
		return nil, errors.New("iCloud returned no displayable photos")
	}
	c.mu.Lock()
	c.urls = urls
	c.mu.Unlock()
	return album, nil
}

func normalizeCloudKitPartition(raw string) (string, error) {
	partition, err := url.Parse(raw)
	if err != nil {
		return "", errors.New("invalid partition URL")
	}
	if partition.Scheme != "https" || (partition.Hostname() != "ckdatabasews.icloud.com" && !cloudKitPartition.MatchString(partition.Hostname())) || (partition.Port() != "" && partition.Port() != "443") || (partition.Path != "" && partition.Path != "/") || partition.User != nil || partition.RawQuery != "" || partition.Fragment != "" {
		return "", fmt.Errorf("scheme=%q host=%q port=%q path_present=%t", partition.Scheme, partition.Hostname(), partition.Port(), partition.Path != "" && partition.Path != "/")
	}
	return "https://" + partition.Hostname(), nil
}

func cloudKitAsset(master, asset ckRecord, filename string) (Asset, string, bool) {
	type resource struct {
		key     string
		w, h    int
		url     string
		version string
	}
	var choices []resource
	for _, prefix := range []string{"resJPEGThumb", "resJPEGMed", "resJPEGLarge", "resJPEGFull"} {
		fileType := fieldString(master.Fields, prefix+"FileType")
		if fileType != "" && fileType != "public.jpeg" && fileType != "public.png" {
			continue
		}
		var value struct {
			DownloadURL string `json:"downloadURL"`
		}
		if json.Unmarshal(master.Fields[prefix+"Res"].Value, &value) != nil || value.DownloadURL == "" {
			continue
		}
		w, h := fieldInt(master.Fields, prefix+"Width"), fieldInt(master.Fields, prefix+"Height")
		if w < 1 || h < 1 {
			continue
		}
		version := fieldString(master.Fields, prefix+"Fingerprint")
		if version == "" {
			version = master.ChangeTag
		}
		choices = append(choices, resource{prefix, w, h, value.DownloadURL, version})
	}
	if len(choices) == 0 && (fieldString(master.Fields, "itemType") == "public.jpeg" || fieldString(master.Fields, "itemType") == "public.png") {
		var value struct {
			DownloadURL string `json:"downloadURL"`
		}
		if json.Unmarshal(master.Fields["resOriginalRes"].Value, &value) == nil && value.DownloadURL != "" {
			choices = append(choices, resource{"resOriginal", fieldInt(master.Fields, "resOriginalWidth"), fieldInt(master.Fields, "resOriginalHeight"), value.DownloadURL, master.ChangeTag})
		}
	}
	if len(choices) == 0 {
		return Asset{}, "", false
	}
	selected := choices[0]
	for _, r := range choices {
		if r.w*r.h > selected.w*selected.h && (max(r.w, r.h) <= 3000 || max(selected.w, selected.h) < 2000) {
			selected = r
		}
	}
	location := strings.ReplaceAll(selected.url, "${f}", url.PathEscape(filename))
	u, err := url.Parse(location)
	if err != nil || u.Scheme != "https" || !strings.HasSuffix(u.Hostname(), ".icloud-content.com") {
		return Asset{}, "", false
	}
	version := selected.key + ":" + selected.version
	if selected.version == "" {
		version = selected.key + ":" + master.Name
	}
	date := time.Time{}
	if ms := fieldInt64(asset.Fields, "assetDate"); ms > 0 {
		date = time.UnixMilli(ms)
	}
	return Asset{ID: master.Name, Version: version, Width: selected.w, Height: selected.h, DateTaken: date}, location, true
}

func fieldString(fields map[string]ckField, key string) string {
	var value string
	json.Unmarshal(fields[key].Value, &value)
	return value
}
func fieldInt(fields map[string]ckField, key string) int { return int(fieldInt64(fields, key)) }
func fieldInt64(fields map[string]ckField, key string) int64 {
	var value int64
	json.Unmarshal(fields[key].Value, &value)
	return value
}
func fieldBool(fields map[string]ckField, key string) bool {
	raw := fields[key].Value
	var number int64
	if json.Unmarshal(raw, &number) == nil {
		return number != 0
	}
	var value bool
	json.Unmarshal(raw, &value)
	return value
}

func (c *CloudKitAlbum) DownloadAsset(ctx context.Context, asset Asset, dst string) error {
	c.mu.RLock()
	location := c.urls[asset.ID]
	c.mu.RUnlock()
	if location == "" {
		return errors.New("asset URL unavailable; refresh album metadata")
	}
	return downloadAssetURL(ctx, c.client, location, dst)
}

func (c *CloudKitAlbum) post(ctx context.Context, endpoint string, body any, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return errors.New("invalid iCloud request")
	}
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Origin", "https://www.icloud.com")
	req.Header.Set("Referer", "https://www.icloud.com/")
	req.Header.Set("Accept", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return errors.New("iCloud request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("iCloud returned HTTP %d", resp.StatusCode)
	}
	response, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return errors.New("iCloud response read failed")
	}
	if json.Unmarshal(response, out) != nil {
		return errors.New("invalid iCloud response")
	}
	return nil
}
