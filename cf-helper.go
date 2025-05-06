package traefikplugin

import (
	"io"
	"log"
	"net/http"
	"strings"
)

// PurgeCache Purge cloudflare cache.
func PurgeCache(config *Config, host string, logger *log.Logger) {
	urlTemplate := "https://api.cloudflare.com/client/v4/zones/$zoneId/purge_cache"
	url := strings.Replace(urlTemplate, "$zoneId", config.CloudflareZone, 1)

	payload := strings.NewReader("{\"hosts\": [\"" + host + "\"]}")

	req, err := http.NewRequest(http.MethodPost, url, payload)
	if err != nil {
		logger.Println(err)
		return
	}

	authToken := strings.Replace("Bearer $token", "$token", config.CloudflareToken, 1)
	req.Header.Add("Authorization", authToken)
	req.Header.Add("Content-Type", "application/json")

	res, err1 := http.DefaultClient.Do(req)
	if err1 != nil {
		logger.Println(err1)
		return
	}

	defer func() {
		err := res.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	body, err2 := io.ReadAll(res.Body)
	if err2 != nil {
		logger.Println(err2)
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
