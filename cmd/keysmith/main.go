// Command keysmith manages SSH keys with two frontends:
// a desktop GUI (default) and a terminal UI (--tui).
package main

import (
	"flag"
	"fmt"
	"os"
)

var version = "dev"

func main() {
	tuiMode := flag.Bool("tui", false, "run the terminal UI instead of the desktop GUI")
	guiMode := flag.Bool("gui", false, "run the desktop GUI (default)")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	switch {
	case *showVersion:
		fmt.Println("keysmith", version)
	default:
		if err := runFrontend(*tuiMode && !*guiMode); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	}
}
