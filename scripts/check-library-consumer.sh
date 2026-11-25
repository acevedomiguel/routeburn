#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repo_root=$(git -C "$script_dir" rev-parse --show-toplevel)
consumer_dir=$(mktemp -d "${TMPDIR:-/tmp}/routeburn-consumer.XXXXXX")
trap 'rm -rf "$consumer_dir"' EXIT

cat > "$consumer_dir/main.go" <<'EOF'
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	routeburn "github.com/acevedomiguel/routeburn"
)

type fakeAPI struct{}

func (fakeAPI) Credits(context.Context) (routeburn.CreditSummary, error) {
	return routeburn.CreditSummary{TotalPurchased: 100, TotalUsed: 57.9, Remaining: 42.1}, nil
}

func (fakeAPI) Key(context.Context) (routeburn.KeyInfo, error) {
	return routeburn.KeyInfo{}, routeburn.ErrMissingAPIKey
}

func (fakeAPI) QueryAnalytics(context.Context, routeburn.AnalyticsQuery) ([]routeburn.AnalyticsRow, error) {
	return nil, nil
}

func (fakeAPI) NextRefreshAllowedAt() time.Time { return time.Time{} }

func main() {
	now := time.Date(2026, 11, 21, 15, 42, 0, 0, time.UTC)
	snap, err := routeburn.Fetch(context.Background(), fakeAPI{}, routeburn.FetchOptions{Now: now})
	if err != nil {
		panic(err)
	}
	if snap.Account.Remaining != 42.1 {
		panic(fmt.Sprintf("remaining = %v, want 42.1", snap.Account.Remaining))
	}
	if len(snap.Daily) != 30 {
		panic(fmt.Sprintf("daily buckets = %d, want 30", len(snap.Daily)))
	}
	if snap.Key != nil {
		panic("missing regular key produced key details")
	}

	var out bytes.Buffer
	if err := routeburn.WriteJSON(&out, snap, false); err != nil {
		panic(err)
	}
	var document struct {
		SchemaVersion int `json:"schema_version"`
	}
	if err := json.Unmarshal(out.Bytes(), &document); err != nil {
		panic(err)
	}
	if document.SchemaVersion != 2 {
		panic(fmt.Sprintf("schema version = %d, want 2", document.SchemaVersion))
	}
	fmt.Println("external library consumer passed")
}
EOF

cd "$consumer_dir"
go mod init example.com/routeburn-consumer
go mod edit -replace=github.com/acevedomiguel/routeburn="$repo_root"
GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local go get github.com/acevedomiguel/routeburn
GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local go run .
