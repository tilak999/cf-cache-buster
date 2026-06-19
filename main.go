// Package cachebuster a Traefik middleware plugin that detects specific
// response headers and triggers Cloudflare cache purge.
package cachebuster

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

// Config the plugin configuration.
type Config struct {
	CloudflareZone  string   `json:"cloudflarezone,omitempty"`
	CloudflareToken string   `json:"cloudflaretoken,omitempty"`
	Headers         []string `json:"headers,omitempty"`
	DryRun          bool     `json:"dryrun,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		Headers:         make([]string, 0),
		DryRun:          false,
		CloudflareZone:  "",
		CloudflareToken: "",
	}
}

// HeaderDetectionPlugin is a Traefik middleware plugin that detects
// configured response headers and purges Cloudflare cache when found.
type HeaderDetectionPlugin struct {
	config *Config
	next   http.Handler
	logger *log.Logger
	name   string
}

// customResponseWriter wraps http.ResponseWriter to intercept headers
// before they are written to the client.
type customResponseWriter struct {
	http.ResponseWriter
	plugin          *HeaderDetectionPlugin
	detectedHeaders map[string]string
}

// New creates a new HeaderDetectionPlugin instance.
func New(_ context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if len(config.Headers) == 0 {
		return nil, errors.New("headers cannot be empty")
	}

	if strings.TrimSpace(config.CloudflareZone) == "" || strings.TrimSpace(config.CloudflareToken) == "" {
		return nil, fmt.Errorf("cloudflare zone or token is not defined, zone=%q | token=%q",
			config.CloudflareZone, config.CloudflareToken)
	}

	logger := log.New(os.Stdout, "[cf-cache-buster] ", log.Ldate|log.Ltime)
	logger.Printf("Plugin %s initialized, ready to accept connections.\n", name)

	return &HeaderDetectionPlugin{
		config: config,
		logger: logger,
		next:   next,
		name:   name,
	}, nil
}

// WriteHeader captures configured headers from the upstream response
// before writing the status code to the client.
func (crw *customResponseWriter) WriteHeader(code int) {
	for _, header := range crw.plugin.config.Headers {
		value := crw.ResponseWriter.Header().Get(header)
		if value != "" {
			crw.detectedHeaders[header] = value
		}
	}
	crw.ResponseWriter.WriteHeader(code)
}

func (a *HeaderDetectionPlugin) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	crw := &customResponseWriter{
		ResponseWriter:  rw,
		plugin:          a,
		detectedHeaders: make(map[string]string),
	}

	a.next.ServeHTTP(crw, req)

	if len(crw.detectedHeaders) > 0 {
		a.logger.Printf("req [host:%s][path:%s]", req.Host, req.URL.Path)
		if a.config.DryRun {
			for k, v := range crw.detectedHeaders {
				a.logger.Printf("  %s=%s", k, v)
			}
		}
		go PurgeCache(context.Background(), a.config, req.Host, a.logger)
	}
}
