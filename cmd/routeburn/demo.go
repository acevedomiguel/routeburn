package main

import (
	"context"
	"fmt"
	"io"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/acevedomiguel/routeburn/internal/app"
	"github.com/acevedomiguel/routeburn/internal/domain"
	"github.com/acevedomiguel/routeburn/internal/ui"
)

// demoModel builds the dashboard on the synthetic UI fixture. It exists so
// screenshots and recordings can be produced without credentials, a
// configuration file, or any network access. The fixture data is fictional.
func demoModel(now func() time.Time) *app.Model {
	return app.New(app.Deps{
		Config: app.Config{
			RefreshInterval: 0,
			NoCache:         true,
		},
		// A placeholder management value keeps the model out of the setup
		// state; it is never sent anywhere.
		Creds: app.Creds{Management: "demo-fixture"},
		Now:   now,
		Fetch: func(_ context.Context, opts app.FetchOptions) (domain.Snapshot, error) {
			return ui.SampleSnapshotAt(opts.Now), nil
		},
	})
}

// runDemo starts the dashboard on the synthetic fixture. It is intentionally
// hidden from the usage text.
func runDemo(stdin io.Reader, stdout, stderr io.Writer) int {
	model := demoModel(func() time.Time { return time.Now().UTC() })

	program := tea.NewProgram(model,
		tea.WithInput(stdin),
		tea.WithOutput(stdout),
		tea.WithAltScreen(),
	)
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(stderr, "routeburn: %v\n", err)
		return exitFailure
	}
	return exitOK
}
