//go:build test

package cf_cache_buster_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	plugin "github.com/tilak999/cf-cache-buster"
)

func TestDemo(t *testing.T) {
	cfg := plugin.CreateConfig()
	cfg.Headers = append(cfg.Headers, "x-invalidate-cache")
	cfg.DryRun = true
	cfg.CloudflareToken = "test"
	cfg.CloudflareZone = "test"

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.Header().Add("x-invalidate-cache", "test-value")
		rw.WriteHeader(http.StatusOK)
	})

	handler, err := plugin.New(ctx, next, cfg, "demo-plugin")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/ping", nil)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(recorder, req)

	// Assert response status is OK
	if recorder.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", recorder.Code)
	}

	// Assert the header was set on the response
	if val := recorder.Header().Get("x-invalidate-cache"); val != "test-value" {
		t.Errorf("expected x-invalidate-cache header to be 'test-value', got %q", val)
	}
}

func TestNewSucceedsWithoutCloudflareConfig(t *testing.T) {
	cfg := plugin.CreateConfig()
	cfg.Headers = append(cfg.Headers, "x-invalidate-cache")

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusOK)
	})

	handler, err := plugin.New(ctx, next, cfg, "demo-plugin")
	if err != nil {
		t.Fatalf("expected plugin to initialize without Cloudflare config: %v", err)
	}
	if handler == nil {
		t.Fatal("expected a handler instance")
	}
}

func TestNewFailsWithEmptyHeaders(t *testing.T) {
	cfg := plugin.CreateConfig()
	// Headers left empty

	ctx := context.Background()
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {})

	_, err := plugin.New(ctx, next, cfg, "demo-plugin")
	if err == nil {
		t.Fatal("expected error when headers are empty, got nil")
	}
}

func TestMultipleHeadersDetected(t *testing.T) {
	cfg := plugin.CreateConfig()
	cfg.Headers = []string{"x-header-one", "x-header-two", "x-header-missing"}
	cfg.DryRun = true
	cfg.CloudflareToken = "test"
	cfg.CloudflareZone = "test"

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.Header().Set("x-header-one", "value-one")
		rw.Header().Set("x-header-two", "value-two")
		// x-header-missing intentionally not set
		rw.WriteHeader(http.StatusOK)
	})

	handler, err := plugin.New(ctx, next, cfg, "demo-plugin")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/test", nil)
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", recorder.Code)
	}
	if v := recorder.Header().Get("x-header-one"); v != "value-one" {
		t.Errorf("expected x-header-one='value-one', got %q", v)
	}
	if v := recorder.Header().Get("x-header-two"); v != "value-two" {
		t.Errorf("expected x-header-two='value-two', got %q", v)
	}
}

func TestNoHeadersDetected(t *testing.T) {
	cfg := plugin.CreateConfig()
	cfg.Headers = []string{"x-never-present"}
	cfg.DryRun = true
	cfg.CloudflareToken = "test"
	cfg.CloudflareZone = "test"

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusCreated)
	})

	handler, err := plugin.New(ctx, next, cfg, "demo-plugin")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/test", nil)
	handler.ServeHTTP(recorder, req)

	// Should still return the status code even when no headers are detected
	if recorder.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", recorder.Code)
	}
}

func TestWriteHeaderCapturesHeaders(t *testing.T) {
	cfg := plugin.CreateConfig()
	cfg.Headers = []string{"x-capture-me"}
	cfg.CloudflareToken = "test"
	cfg.CloudflareZone = "test"

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.Header().Set("x-capture-me", "captured")
		rw.WriteHeader(http.StatusAccepted)
	})

	handler, err := plugin.New(ctx, next, cfg, "demo-plugin")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/test", nil)
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusAccepted {
		t.Errorf("expected status 202, got %d", recorder.Code)
	}
	if v := recorder.Header().Get("x-capture-me"); v != "captured" {
		t.Errorf("expected x-capture-me='captured', got %q", v)
	}
}

func TestConcurrentRequests(t *testing.T) {
	cfg := plugin.CreateConfig()
	cfg.Headers = []string{"x-concurrent"}
	cfg.DryRun = true
	cfg.CloudflareToken = "test"
	cfg.CloudflareZone = "test"

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.Header().Set("x-concurrent", "yes")
		rw.WriteHeader(http.StatusOK)
	})

	handler, err := plugin.New(ctx, next, cfg, "demo-plugin")
	if err != nil {
		t.Fatal(err)
	}

	//nolint:intrange
	for i := 0; i < 10; i++ {
		go func() {
			recorder := httptest.NewRecorder()
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/test", nil)
			handler.ServeHTTP(recorder, req)
		}()
	}
}

