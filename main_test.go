package cf_cache_buster_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	plugin "github.com/tilak999/cf-cache-buster"
)

func TestHeaderDetection(t *testing.T) {
	cfg := plugin.CreateConfig()
	cfg.Headers = append(cfg.Headers, "x-invalidate-cache")
	cfg.DryRun = true
	cfg.CloudflareToken = "test-token"
	cfg.CloudflareZone = "test-zone"

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.Header().Add("x-invalidate-cache", "test-value")
		rw.WriteHeader(http.StatusOK)
	})

	handler, err := plugin.New(ctx, next, cfg, "test-plugin")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/ping", nil)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestEmptyHeadersConfig(t *testing.T) {
	cfg := plugin.CreateConfig()
	cfg.CloudflareToken = "test-token"
	cfg.CloudflareZone = "test-zone"

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusOK)
	})

	_, err := plugin.New(ctx, next, cfg, "test-plugin")
	if err == nil {
		t.Fatal("expected error for empty headers config, got nil")
	}
}

func TestMissingCloudflareCredentials(t *testing.T) {
	tests := []struct {
		name  string
		zone  string
		token string
	}{
		{"missing both", "", ""},
		{"missing zone", "", "test-token"},
		{"missing token", "test-zone", ""},
		{"whitespace zone", "  ", "test-token"},
		{"whitespace token", "test-zone", "  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := plugin.CreateConfig()
			cfg.Headers = append(cfg.Headers, "x-invalidate-cache")
			cfg.CloudflareZone = tt.zone
			cfg.CloudflareToken = tt.token

			ctx := context.Background()
			next := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
				rw.WriteHeader(http.StatusOK)
			})

			_, err := plugin.New(ctx, next, cfg, "test-plugin")
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tt.name)
			}
		})
	}
}

func TestNoMatchingHeaders(t *testing.T) {
	cfg := plugin.CreateConfig()
	cfg.Headers = append(cfg.Headers, "x-invalidate-cache")
	cfg.DryRun = true
	cfg.CloudflareToken = "test-token"
	cfg.CloudflareZone = "test-zone"

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.Header().Add("x-unrelated-header", "some-value")
		rw.WriteHeader(http.StatusOK)
	})

	handler, err := plugin.New(ctx, next, cfg, "test-plugin")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/ping", nil)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestStatusCodeProxying(t *testing.T) {
	cfg := plugin.CreateConfig()
	cfg.Headers = append(cfg.Headers, "x-invalidate-cache")
	cfg.DryRun = true
	cfg.CloudflareToken = "test-token"
	cfg.CloudflareZone = "test-zone"

	ctx := context.Background()

	statusCodes := []int{
		http.StatusOK,
		http.StatusNotFound,
		http.StatusInternalServerError,
		http.StatusCreated,
	}

	for _, expectedCode := range statusCodes {
		t.Run(http.StatusText(expectedCode), func(t *testing.T) {
			code := expectedCode
			next := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
				rw.Header().Add("x-invalidate-cache", "purge")
				rw.WriteHeader(code)
			})

			handler, err := plugin.New(ctx, next, cfg, "test-plugin")
			if err != nil {
				t.Fatal(err)
			}

			recorder := httptest.NewRecorder()
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/test", nil)
			if err != nil {
				t.Fatal(err)
			}

			handler.ServeHTTP(recorder, req)

			if recorder.Code != code {
				t.Errorf("expected status %d, got %d", code, recorder.Code)
			}
		})
	}
}
