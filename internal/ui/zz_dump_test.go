package ui_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/acevedomiguel/routeburn/internal/app"
	"github.com/acevedomiguel/routeburn/internal/ui"
)

func TestZZDump(t *testing.T) {
	now := time.Date(2026, 11, 21, 15, 42, 0, 0, time.UTC)
	snap := app.DemoSnapshot(now)
	out := ui.Render(&snap, ui.ViewOptions{Width: 100, Height: 30, Styles: ui.NewStyles(false), Now: now})

	var lines []string
	for _, l := range strings.Split(out, "\n") {
		lines = append(lines, strings.TrimRight(l, " "))
	}

	if err := os.WriteFile("/tmp/screenshot.txt", []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
}
