package traefikplugin

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const purgeTimeout = 10 * time.Second

// purgeRequest represents the Cloudflare cache purge API request body.
type purgeRequest struct {
	Hosts []string `json:"hosts"`
}

// PurgeCache purges Cloudflare cache for the given host.
func PurgeCache(ctx context.Context, config *Config, host string, logger *log.Logger) {
	httpClient := &http.Client{Timeout: purgeTimeout}

	url := "https://api.cloudflare.com/client/v4/zones/" + config.CloudflareZone + "/purge_cache"

	payload, err := json.Marshal(purgeRequest{Hosts: []string{host}})
	if err != nil {
		logger.Printf("failed to marshal purge request: %v", err)
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(payload)))
	if err != nil {
		logger.Printf("failed to create purge request: %v", err)
		return
	}

	req.Header.Set("Authorization", "Bearer "+config.CloudflareToken)
	req.Header.Set("Content-Type", "application/json")

	res, err := httpClient.Do(req)
	if err != nil {
		logger.Printf("purge request failed: %v", err)
		return
	}

	defer func() {
		if cerr := res.Body.Close(); cerr != nil {
			logger.Printf("failed to close response body: %v", cerr)
		}
	}()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		logger.Printf("failed to read purge response: %v", err)
		return
	}

	if res.StatusCode == http.StatusOK {
		logger.Print("Cloudflare cache purged: OK")
		if config.DryRun {
			logger.Println(string(body))
		}
		return
	}

	logger.Printf("purge request failed with status %d %s", res.StatusCode, http.StatusText(res.StatusCode))
	logger.Println(string(body))
}
