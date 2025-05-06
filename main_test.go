package traefikplugin_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	main "github.com/tilak999/traefikplugin"
)

func TestDemo(t *testing.T) {
	cfg := main.CreateConfig()
	cfg.Headers = append(cfg.Headers, "x-invalidate-cache")
	cfg.DryRun = true
	cfg.CloudflareToken = "test"
	cfg.CloudflareZone = "test"

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.Header().Add("x-invalidate-cache", "test-value")
		rw.WriteHeader(http.StatusOK)
	})

	handler, err := main.New(ctx, next, cfg, "demo-plugin")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/ping", nil)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(recorder, req)

	// assertHeader(t, req, "X-Host", "localhost")
	// assertHeader(t, req, "X-URL", "http://localhost")
	// assertHeader(t, req, "X-Method", "GET")
	// assertHeader(t, req, "X-Demo", "test")
}

/* func assertHeader(t *testing.T, req *http.Request, key, expected string) {
	t.Helper()

	if req.Header.Get(key) != expected {
		t.Errorf("invalid header value: %s", req.Header.Get(key))
	}
} */
