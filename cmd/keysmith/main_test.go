package main

import (
	"bytes"
	"errors"
	"flag"
	"strings"
	"testing"
)

func TestParseOptionsSelectsRequestedFrontend(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want commandOptions
	}{
		{name: "default", args: nil, want: commandOptions{mode: guiMode}},
		{name: "tui shorthand", args: []string{"--tui"}, want: commandOptions{mode: tuiMode}},
		{name: "tui boolean", args: []string{"--tui=true"}, want: commandOptions{mode: tuiMode}},
		{name: "explicit gui", args: []string{"--gui=true"}, want: commandOptions{mode: guiMode}},
		{name: "version", args: []string{"--version"}, want: commandOptions{mode: guiMode, version: true}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseOptions(test.args, &bytes.Buffer{})
			if err != nil {
				t.Fatalf("parseOptions() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("parseOptions() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestParseOptionsRejectsConflictingFrontends(t *testing.T) {
	_, err := parseOptions([]string{"--gui", "--tui=true"}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "cannot be used together") {
		t.Fatalf("parseOptions() error = %v, want conflicting frontend error", err)
	}
}

func TestParseOptionsProvidesClearHelp(t *testing.T) {
	var output bytes.Buffer
	_, err := parseOptions([]string{"--help"}, &output)
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("parseOptions() error = %v, want flag.ErrHelp", err)
	}
	for _, text := range []string{
		"Usage: keysmith [options]",
		"-tui",
		"terminal UI",
		"-gui",
		"desktop GUI (default)",
		"-version",
	} {
		if !strings.Contains(output.String(), text) {
			t.Errorf("help output %q does not contain %q", output.String(), text)
		}
	}
}
