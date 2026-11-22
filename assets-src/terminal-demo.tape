# RouteBurn terminal demo capture for the project website.
#
# Run from the repository root after `make build`:
#   vhs assets-src/terminal-demo.tape
#
# The demo command renders the synthetic fixture from internal/ui/fixture.go;
# no credentials, configuration, or network access are involved. Every value
# in the capture is fictional.

Set Shell bash
Set FontSize 18
Set FontFamily "Menlo"
Set Width 1280
Set Height 800
Set Padding 24
Set TypingSpeed 50ms

# The recorder inherits the operator's environment; a global NO_COLOR would
# force the dashboard into its monochrome mode, which is not representative.
Type "env -u NO_COLOR ./routeburn demo"
Enter

# Give the TUI a moment to render the fixture snapshot.
Sleep 2s

Screenshot assets-src/terminal-demo.png

Type "q"
Sleep 500ms
