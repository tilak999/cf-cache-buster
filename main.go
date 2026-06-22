// Package cf_cache_buster a Traefik middleware plugin that detects specific
// response headers and triggers Cloudflare cache purge.
package cf_cache_buster

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"time"
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

// HeaderDetectionPlugin a HeaderDetectionPlugin plugin.
type HeaderDetectionPlugin struct {
	config       *Config
	next         http.Handler
	logger       *log.Logger
	name         string
	debouncer    *Debouncer
}

// CustomResponseWriter Custom response writer.
type CustomResponseWriter struct {
	http.ResponseWriter
	WatchedHeaders  []string          `json:"watchedHeaders,omitempty"`
	URL             string            `json:"url,omitempty"`
	DetectedHeaders map[string]string `json:"detectedHeaders,omitempty"`
}

const debounceDuration = 5 * time.Second

// New created a new Demo plugin.
func New(_ context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	logger := log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime)

	if len(config.Headers) == 0 {
		return nil, errors.New("headers cannot be empty")
	}

	if config.CloudflareToken == "" || config.CloudflareZone == "" {
		logger.Printf("Cloudflare credentials not configured, skipping purge setup; zone=%s | token=%s",
			config.CloudflareZone, config.CloudflareToken)
	}

	logger.Println("Plugin initialized, ready to accept connections.")

	return &HeaderDetectionPlugin{
		config:    config,
		logger:    logger,
		next:      next,
		name:      name,
		debouncer: NewDebouncer(debounceDuration),
	}, nil
}

// WriteHeader captures headers before they're written.
func (crw *CustomResponseWriter) WriteHeader(code int) {
	for _, header := range crw.WatchedHeaders {
		value := crw.ResponseWriter.Header().Get(header)
		if value != "" {
			crw.DetectedHeaders[header] = value
		}
	}
	crw.ResponseWriter.WriteHeader(code)
}

// Purge debounces and purges host.
func (a *HeaderDetectionPlugin) Purge(host string) {
	a.debouncer.Run(func() {
		hosts := []string{host}
		defer func() {
			if r := recover(); r != nil {
				a.logger.Printf("panic in PurgeCache recovered: %v", r)
			}
		}()
		PurgeCache(a.config, hosts, a.logger)
	})
}

func (a *HeaderDetectionPlugin) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	customRW := &CustomResponseWriter{
		ResponseWriter:  rw,
		WatchedHeaders:  a.config.Headers,
		DetectedHeaders: make(map[string]string),
	}
	a.next.ServeHTTP(customRW, req)
	if len(customRW.DetectedHeaders) > 0 {
		a.logger.Printf("req [host:%s][path:%s]", req.Host, req.URL.Path)
		if a.config.DryRun {
			for k, v := range customRW.DetectedHeaders {
				a.logger.Printf("%s=%s", k, v)
			}
		}
		if a.config.CloudflareToken != "" && a.config.CloudflareZone != "" {
			a.Purge(req.Host)
		} else {
			a.logger.Println("Cloudflare credentials missing; skipping cache purge")
		}
	}
}
