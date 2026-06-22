package cf_cache_buster

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// purgePayload is the JSON body for the Cloudflare purge API.
type purgePayload struct {
	Hosts []string `json:"hosts"`
}

//nolint:gochecknoglobals // timeout duration
var timeoutDuration = 10 * time.Second

//nolint:gochecknoglobals // client for concurrent use and reuses connections
var purgeClient = &http.Client{Timeout: timeoutDuration}

// PurgeCache Purge cloudflare cache for the specified hosts.
func PurgeCache(config *Config, hosts []string, logger *log.Logger) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/purge_cache", config.CloudflareZone)

	payload, err := json.Marshal(purgePayload{Hosts: hosts})
	if err != nil {
		logger.Printf("failed to marshal purge payload: %v", err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		logger.Println(err)
		return
	}

	req.Header.Set("Authorization", "Bearer "+config.CloudflareToken)
	req.Header.Set("Content-Type", "application/json")

	res, err := purgeClient.Do(req)
	if err != nil {
		logger.Printf("failed to purge cache: %v", err)
		return
	}
	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil {
			logger.Printf("failed to close body: %v", closeErr)
		}
	}()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		logger.Println(err)
		return
	}

	if res.StatusCode == http.StatusOK {
		logger.Print("Cloudflare cache purged: OK")
		if config.DryRun {
			logger.Println(string(body))
		}
		return
	}

	logger.Printf("Request completed with status != 200: actual status [%d]", res.StatusCode)
	logger.Println(string(body))
}
