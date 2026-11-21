package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

// version is overridden at release time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run executes the CLI with explicit streams so tests never touch the real
// terminal or process exit status.
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("routeburn", flag.ContinueOnError)
	fs.SetOutput(stderr)
	showVersion := fs.Bool("version", false, "print version and exit")
	fs.Usage = func() { printUsage(stderr) }

	// flag calls Usage itself for -h/-help; handle help here so the
	// request prints to stdout while parse errors keep going to stderr.
	for _, a := range args {
		if a == "-h" || a == "-help" || a == "--help" {
			printUsage(stdout)
			return 0
		}
	}

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *showVersion {
		fmt.Fprintf(stdout, "routeburn version %s\n", version)
		return 0
	}

	// The dashboard is not implemented yet: running bare must not pretend
	// to work, but it also must not fail like a flag error.
	fmt.Fprintln(stderr, "routeburn: the dashboard is not available in this build yet")
	return 1
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, `Usage:
  routeburn            Start the dashboard
  routeburn --version  Print version and exit
  routeburn --help     Print this help and exit

Flags:
  --version            Print version and exit
  --help               Print this help and exit
`)
}
