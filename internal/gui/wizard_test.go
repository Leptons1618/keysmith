//go:build !tui

package gui

import (
	"errors"
	"strings"
	"testing"

	"keysmith/internal/core"
)

func TestKeyForTestKeepsRetrySubject(t *testing.T) {
	tests := []struct {
		name     string
		selected string
		retry    string
		want     string
	}{
		{name: "first run uses selected key", selected: "selected", retry: "", want: "selected"},
		{name: "retry keeps result key", selected: "different", retry: "result-key", want: "result-key"},
		{name: "no selected key", selected: "", retry: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := keyForTest(tt.selected, tt.retry); got != tt.want {
				t.Fatalf("keyForTest(%q, %q) = %q, want %q", tt.selected, tt.retry, got, tt.want)
			}
		})
	}
}

func TestDeleteAndForgetPropagatesDeleteFailure(t *testing.T) {
	deleteErr := errors.New("permission denied")
	forgotten := false

	err := deleteAndForget("work", func(string) error { return deleteErr }, func(string) error {
		forgotten = true
		return nil
	})
	if !errors.Is(err, deleteErr) {
		t.Fatalf("deleteAndForget error = %v, want wrapped delete error", err)
	}
	if forgotten {
		t.Fatal("workflow state was forgotten after key deletion failed")
	}
}

func TestDeleteAndForgetPropagatesStateFailure(t *testing.T) {
	stateErr := errors.New("state file is read-only")
	deleted := ""

	err := deleteAndForget("work", func(key string) error {
		deleted = key
		return nil
	}, func(string) error { return stateErr })
	if !errors.Is(err, stateErr) {
		t.Fatalf("deleteAndForget error = %v, want wrapped state error", err)
	}
	if deleted != "work" {
		t.Fatalf("deleted key = %q, want work", deleted)
	}
	if !strings.Contains(err.Error(), "workflow state") {
		t.Fatalf("error %q does not explain the partial failure", err)
	}
}

func TestKeyIndexFindsSelectedKey(t *testing.T) {
	keys := []core.KeyInfo{{Name: "alpha"}, {Name: "work"}, {Name: "zeta"}}
	if got := keyIndex(keys, "work"); got != 1 {
		t.Fatalf("keyIndex(work) = %d, want 1", got)
	}
	if got := keyIndex(keys, "missing"); got != -1 {
		t.Fatalf("keyIndex(missing) = %d, want -1", got)
	}
}
