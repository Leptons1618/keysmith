//go:build !tui

package gui

import (
	"errors"
	"strings"
	"testing"
)

type recordingState struct {
	copied    []string
	agent     []string
	tests     []testTransition
	used      []string
	forgotten []string
	err       error
}

type testTransition struct {
	key string
	ok  bool
}

func (s *recordingState) MarkCopied(key string) error {
	s.copied = append(s.copied, key)
	return s.err
}

func (s *recordingState) MarkAgentLoaded(key string) error {
	s.agent = append(s.agent, key)
	return s.err
}

func (s *recordingState) RecordTest(key string, ok bool) error {
	s.tests = append(s.tests, testTransition{key: key, ok: ok})
	return s.err
}

func (s *recordingState) MarkUsed(key string) error {
	s.used = append(s.used, key)
	return s.err
}

func (s *recordingState) ForgetKey(key string) error {
	s.forgotten = append(s.forgotten, key)
	return s.err
}

func TestStoreTransitionsUseSelectedKey(t *testing.T) {
	state := &recordingState{}
	if err := recordCopy(state, "work"); err != nil {
		t.Fatal(err)
	}
	if err := recordAgentLoad(state, "work"); err != nil {
		t.Fatal(err)
	}
	if err := recordTest(state, "work", true); err != nil {
		t.Fatal(err)
	}
	if err := recordServiceUse(state, "work"); err != nil {
		t.Fatal(err)
	}

	if len(state.copied) != 1 || state.copied[0] != "work" ||
		len(state.agent) != 1 || state.agent[0] != "work" ||
		len(state.tests) != 1 || state.tests[0].key != "work" || !state.tests[0].ok ||
		len(state.used) != 1 || state.used[0] != "work" {
		t.Fatalf("transitions did not target work: %+v", state)
	}
}

func TestStoreTransitionsPropagatePersistenceFailures(t *testing.T) {
	stateErr := errors.New("disk full")
	state := &recordingState{err: stateErr}

	operations := map[string]func() error{
		"copy":        func() error { return recordCopy(state, "work") },
		"agent load":  func() error { return recordAgentLoad(state, "work") },
		"test result": func() error { return recordTest(state, "work", false) },
		"service use": func() error { return recordServiceUse(state, "work") },
	}
	for name, operation := range operations {
		t.Run(name, func(t *testing.T) {
			err := operation()
			if !errors.Is(err, stateErr) {
				t.Fatalf("error = %v, want %v", err, stateErr)
			}
			if !strings.Contains(err.Error(), "could not save") {
				t.Fatalf("error %q does not explain persistence failure", err)
			}
		})
	}
}

func TestForgetKeyAdapterPropagatesPersistenceFailure(t *testing.T) {
	stateErr := errors.New("read-only state")
	state := &recordingState{err: stateErr}
	err := forgetDeletedKey(state, "work")
	if !errors.Is(err, stateErr) {
		t.Fatalf("error = %v, want %v", err, stateErr)
	}
	if !strings.Contains(err.Error(), "could not save") {
		t.Fatalf("error %q does not explain persistence failure", err)
	}
}
