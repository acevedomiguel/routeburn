package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/acevedomiguel/burnblocks/openrouter"
)

// fakeAPI serves the three OpenRouter endpoints the snapshot needs.
func fakeAPI(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/credits":
			io.WriteString(w, `{"data":{"total_credits":100,"total_usage":57.9}}`)
		case "/key":
			io.WriteString(w, `{"data":{"label":"prod","limit":50,"limit_remaining":31.2,"limit_reset":"monthly","usage_monthly":18.8}}`)
		case "/analytics/query":
			body, _ := io.ReadAll(r.Body)
			if bytes.Contains(body, []byte(`"dimensions"`)) {
				io.WriteString(w, `{"data":[{"model":"openai/gpt-5","total_usage":9.5,"tokens_total":1000000}]}`)
				return
			}
			io.WriteString(w, `{"data":[{"day":"2026-11-20","total_usage":0.4,"tokens_total":100000}]}`)
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestRunJSONPrintsContract(t *testing.T) {
	srv := fakeAPI(t)
	client, err := openrouter.NewClient(
		openrouter.WithBaseURL(srv.URL),
		openrouter.WithManagementKey("mgmt"),
		openrouter.WithAPIKey("api"),
	)
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	dir := t.TempDir()
	rt := &runtime{
		opts:   options{jsonMode: true},
		cfg:    openrouter.DefaultConfig(),
		creds:  openrouter.Creds{Management: "mgmt", APIKey: "api"},
		client: client,
		cache:  openrouter.NewCacheStore(filepath.Join(dir, "snapshot.json")),
		now:    func() time.Time { return time.Now().UTC() },
	}

	var stdout, stderr bytes.Buffer
	if code := runJSON(rt, &stdout, &stderr); code != exitOK {
		t.Fatalf("exit = %d (stderr %q)", code, stderr.String())
	}

	var decoded map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if got := int(decoded["schema_version"].(float64)); got != openrouter.SchemaVersion {
		t.Errorf("schema_version = %v, want %d", got, openrouter.SchemaVersion)
	}
	if decoded["cached"] != false {
		t.Errorf("cached = %v", decoded["cached"])
	}
	account := decoded["account"].(map[string]any)
	if account["remaining"].(float64) != 42.1 {
		t.Errorf("remaining = %v", account["remaining"])
	}
	if decoded["key"] == nil {
		t.Error("key section missing")
	}

	// A successful run must write the cache for the next launch.
	if _, err := rt.cache.Load(); err != nil {
		t.Errorf("cache was not written: %v", err)
	}
}

func TestRunJSONFallsBackToCacheWhenAPIFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"boom"}}`, http.StatusBadGateway)
	}))
	t.Cleanup(srv.Close)

	client, err := openrouter.NewClient(openrouter.WithBaseURL(srv.URL), openrouter.WithManagementKey("mgmt"), openrouter.WithSleep(func(time.Duration) {}))
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	dir := t.TempDir()
	store := openrouter.NewCacheStore(filepath.Join(dir, "snapshot.json"))
	if err := store.Save(cachedSnapshot()); err != nil {
		t.Fatalf("seed: %v", err)
	}

	rt := &runtime{
		opts:   options{jsonMode: true},
		cfg:    openrouter.DefaultConfig(),
		client: client,
		cache:  store,
		now:    func() time.Time { return time.Now().UTC() },
	}

	var stdout, stderr bytes.Buffer
	if code := runJSON(rt, &stdout, &stderr); code != exitOK {
		t.Fatalf("exit = %d, want 0 with a cache fallback (stderr %q)", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"cached": true`)) {
		t.Errorf("stdout does not flag cached data:\n%s", stdout.String())
	}
}

func TestRunJSONWithoutCacheAndAPIFailureExitsNonzero(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"boom"}}`, http.StatusBadGateway)
	}))
	t.Cleanup(srv.Close)

	client, err := openrouter.NewClient(openrouter.WithBaseURL(srv.URL), openrouter.WithManagementKey("mgmt"), openrouter.WithSleep(func(time.Duration) {}))
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	rt := &runtime{
		opts:   options{jsonMode: true},
		cfg:    openrouter.DefaultConfig(),
		client: client,
		cache:  nil,
		now:    func() time.Time { return time.Now().UTC() },
	}

	var stdout, stderr bytes.Buffer
	if code := runJSON(rt, &stdout, &stderr); code != exitFailure {
		t.Fatalf("exit = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte("routeburn:")) {
		t.Errorf("stderr = %q", stderr.String())
	}
}

func cachedSnapshot() openrouter.Snapshot {
	now := time.Now().UTC()
	return openrouter.Snapshot{
		SchemaVersion: openrouter.SchemaVersion,
		GeneratedAt:   now.Add(-2 * time.Hour),
		Range:         openrouter.DisplayRange(now),
		Account:       openrouter.AccountCredits{TotalPurchased: 100, TotalUsed: 10, Remaining: 90},
		Daily:         openrouter.AggregateDaily(openrouter.DisplayRange(now), nil),
		Runway:        openrouter.RunwayEstimate{Basis: openrouter.BasisNoRecentSpend},
	}
}
