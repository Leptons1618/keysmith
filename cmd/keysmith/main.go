// Command keysmith manages SSH keys with two frontends:
// a desktop GUI (default) and a terminal UI (--tui).
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
)

var version = "dev"

type frontendMode string

const (
	guiMode frontendMode = "gui"
	tuiMode frontendMode = "tui"
)

type commandOptions struct {
	mode    frontendMode
	version bool
}

var errFrontendConflict = errors.New("--gui and --tui cannot be used together")

func printUsage(output io.Writer) {
	fmt.Fprint(output, "Usage: keysmith [options]\n\nOptions:\n")
	fmt.Fprint(output, "  -tui    run the terminal UI instead of the desktop GUI\n")
	fmt.Fprint(output, "  -gui    run the desktop GUI (default)\n")
	fmt.Fprint(output, "  -version\n")
	fmt.Fprint(output, "        print the keysmith version and exit\n")
	fmt.Fprint(output, "  -h, -help\n")
	fmt.Fprint(output, "        print this help and exit\n")
}

func parseOptions(args []string, output io.Writer) (commandOptions, error) {
	flags := flag.NewFlagSet("keysmith", flag.ContinueOnError)
	flags.SetOutput(output)
	flags.Usage = func() { printUsage(output) }

	tui := flags.Bool("tui", false, "run the terminal UI")
	gui := flags.Bool("gui", false, "run the desktop GUI (default)")
	showVersion := flags.Bool("version", false, "print version and exit")
	if err := flags.Parse(args); err != nil {
		return commandOptions{}, err
	}
	if *tui && *gui {
		return commandOptions{}, errFrontendConflict
	}

	options := commandOptions{mode: guiMode, version: *showVersion}
	if *tui {
		options.mode = tuiMode
	}
	return options, nil
}

func main() {
	var parsedOutput bytes.Buffer
	options, err := parseOptions(os.Args[1:], &parsedOutput)
	if errors.Is(err, flag.ErrHelp) {
		_, _ = os.Stdout.Write(parsedOutput.Bytes())
		return
	}
	if errors.Is(err, errFrontendConflict) {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		printUsage(os.Stderr)
		os.Exit(2)
	}
	if err != nil {
		_, _ = os.Stderr.Write(parsedOutput.Bytes())
		os.Exit(2)
	}

	if options.version {
		fmt.Println("keysmith", version)
		return
	}
	if err := runFrontend(options.mode); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
