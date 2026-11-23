package main

import (
	"context"
	"fmt"
	"io"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

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
		Creds:   app.Creds{Management: "demo-fixture"},
		Now:     now,
		Version: version,
		Fetch: func(_ context.Context, opts app.FetchOptions) (domain.Snapshot, error) {
			return ui.SampleSnapshotAt(opts.Now), nil
		},
	})
}

// demoColorProfile pins the color profile so recordings render in color
// regardless of the recorder's environment.
func demoColorProfile() colorprofile.Profile {
	return colorprofile.ANSI256
}

// runDemo starts the dashboard on the synthetic fixture. It is intentionally
// hidden from the usage text.
func runDemo(stdin io.Reader, stdout, stderr io.Writer) int {
	// Recorders inherit the operator's environment, where NO_COLOR or a
	// non-tty probe would silently flatten the dashboard to plain text. The
	// demo exists to produce representative captures, so it pins the profile
	// the way an interactive terminal would present it.
	model := demoModel(func() time.Time { return time.Now().UTC() })

	program := tea.NewProgram(model,
		tea.WithInput(stdin),
		tea.WithOutput(stdout),
		tea.WithColorProfile(demoColorProfile()),
	)
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(stderr, "routeburn: %v\n", err)
		return exitFailure
	}
	return exitOK
}
