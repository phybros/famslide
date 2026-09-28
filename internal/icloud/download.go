package icloud

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func downloadAssetURL(ctx context.Context, client *http.Client, location, dst string) error {
	u, err := url.Parse(location)
	if err != nil || u.Scheme != "https" || !strings.HasSuffix(u.Hostname(), ".icloud-content.com") || u.User != nil {
		return errors.New("unexpected asset host")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, location, nil)
	if err != nil {
		return errors.New("invalid asset URL")
	}
	downloadClient := *client
	downloadClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 || req.URL.Scheme != "https" || !strings.HasSuffix(req.URL.Hostname(), ".icloud-content.com") {
			return errors.New("unexpected asset redirect")
		}
		return nil
	}
	resp, err := downloadClient.Do(req)
	if err != nil {
		return errors.New("asset download failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("asset download returned HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	n, err := io.Copy(f, io.LimitReader(resp.Body, 32<<20+1))
	if err != nil {
		return err
	}
	if n > 32<<20 {
		return errors.New("asset exceeds 32 MiB")
	}
	if n == 0 {
		return errors.New("empty asset")
	}
	return f.Sync()
}
