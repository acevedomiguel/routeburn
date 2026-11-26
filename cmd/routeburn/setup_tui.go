package main

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/acevedomiguel/burnblocks/openrouter"
	"github.com/acevedomiguel/routeburn/internal/ui"
)

// setupStep is the current stage of the guided setup.
type setupStep int

const (
	setupManagement setupStep = iota
	setupValidating
	setupDone
)

// setupModel is the first-run credential wizard. It mirrors ui.SetupView so
// terminal rendering stays consistent with the rest of the app.
type setupModel struct {
	opts   options
	paths  openrouter.Paths
	step   setupStep
	width  int
	height int
	input  textinput.Model
	styles ui.Styles
	banner string
	notice string
	creds  openrouter.Creds
	store  *openrouter.CredentialStore
	client *openrouter.Client
	now    func() time.Time
}

// validationMsg carries the result of a live credential check.
type validationMsg struct{ err error }

func newSetupModel(opts options, paths openrouter.Paths) *setupModel {
	input := textinput.New()
	input.EchoMode = textinput.EchoPassword
	input.EchoCharacter = '*'
	input.Prompt = ""
	input.Focus()
	input.CharLimit = 200

	return &setupModel{
		opts:   opts,
		paths:  paths,
		step:   setupManagement,
		input:  input,
		styles: ui.NewStyles(!opts.noColor),
		store:  credentialsStoreFor(paths, opts),
		now:    func() time.Time { return time.Now().UTC() },
	}
}

func (s *setupModel) Init() tea.Cmd { return textinput.Blink }

func (s *setupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width, s.height = msg.Width, msg.Height
		return s, nil

	case validationMsg:
		return s.handleValidation(msg)

	case tea.KeyPressMsg:
		return s.handleKey(msg)
	}

	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	return s, cmd
}

func (s *setupModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		// Exit without writing a partial configuration.
		return s, tea.Quit
	case "esc":
		// Setup has a single required step, so Esc cannot mean "skip".
		return s, tea.Quit
	case "enter":
		return s.submit()
	}

	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	return s, cmd
}

func (s *setupModel) submit() (tea.Model, tea.Cmd) {
	value := strings.TrimSpace(s.input.Value())
	switch s.step {
	case setupManagement:
		if value == "" {
			s.banner = "A management key is required."
			return s, nil
		}
		s.creds.Management = value
		s.input.SetValue("")
		s.banner = ""
		s.step = setupValidating
		management := s.creds.Management
		return s, func() tea.Msg {
			client, err := openrouter.NewClient(
				openrouter.WithManagementKey(management),
				openrouter.WithVersion(version),
			)
			if err != nil {
				return validationMsg{err: err}
			}
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			_, err = client.Credits(ctx)
			return validationMsg{err: err}
		}
	}
	return s, nil
}

func (s *setupModel) handleValidation(msg validationMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		// Keep the entered token in memory so retry needs no retyping.
		s.step = setupManagement
		s.banner = "Validation failed: " + openrouter.Redact(msg.err.Error())
		s.notice = "Press Enter to retry, or Esc to cancel."
		return s, nil
	}
	return s, s.saveAndQuit()
}

// saveAndQuit stores the credentials, then exits the wizard.
func (s *setupModel) saveAndQuit() tea.Cmd {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.store.Save(ctx, openrouter.CredentialManagement, s.creds.Management); err != nil {
		s.banner = "Could not store the management key: " + openrouter.Redact(err.Error())
		return nil
	}

	cfgPath := s.paths.ConfigFile
	if s.opts.configPath != "" {
		cfgPath = s.opts.configPath
	}
	if err := openrouter.WriteSample(cfgPath); err != nil {
		s.banner = "Credentials saved, but the configuration could not be written: " + err.Error()
		return nil
	}

	s.step = setupDone
	return tea.Quit
}

func (s *setupModel) View() tea.View {
	keyLen := len([]rune(s.input.Value()))
	return tea.NewView(ui.SetupView(ui.SetupOptions{
		Width:      s.width,
		Height:     s.height,
		Styles:     s.styles,
		Banner:     s.banner,
		KeyLen:     keyLen,
		Validating: s.step == setupValidating,
		Notice:     s.notice,
	}))
}

// runSetupInteractive runs the wizard and reports the outcome.
func runSetupInteractive(paths openrouter.Paths, opts options, stdin io.Reader, stdout, stderr io.Writer) int {
	model := newSetupModel(opts, paths)
	program := tea.NewProgram(model, tea.WithInput(stdin), tea.WithOutput(stdout))
	final, err := program.Run()
	if err != nil {
		fmt.Fprintf(stderr, "routeburn: %v\n", err)
		return exitFailure
	}

	m, ok := final.(*setupModel)
	if !ok || m.step != setupDone {
		fmt.Fprintln(stderr, "routeburn: setup cancelled; nothing was saved.")
		return exitCredentials
	}
	if m.banner != "" {
		fmt.Fprintf(stderr, "routeburn: %s\n", m.banner)
		return exitFailure
	}

	cfgPath := paths.ConfigFile
	if opts.configPath != "" {
		cfgPath = opts.configPath
	}
	fmt.Fprintf(stdout, "Credentials saved. Configuration written to %s\n", cfgPath)
	fmt.Fprintln(stdout, "Run `routeburn` to open the dashboard.")
	return exitOK
}
