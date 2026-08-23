package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

// StateFilePath resolves the state file path per call so it always follows
// the current HOME (tests swap HOME; a package-level var would freeze the
// real user's path at init). The schema is identical to the Python version
// so both builds interoperate.
func StateFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".ssh-key-gui-state.json"
	}
	return filepath.Join(home, ".ssh-key-gui-state.json")
}

// Store mirrors the Python app's persisted per-key workflow state.
type Store struct {
	UsedKeys        map[string]bool
	CopiedKeys      map[string]bool
	TestedKeysOK    map[string]bool
	AgentLoadedKeys map[string]bool
}

type storeFile struct {
	UsedKeys        []string        `json:"used_keys"`
	CopiedKeys      []string        `json:"copied_keys"`
	TestedKeysOK    map[string]bool `json:"tested_keys_ok"`
	AgentLoadedKeys []string        `json:"agent_loaded_keys"`
}

// LoadStore reads the state file, tolerating absence and corruption
// (both degrade to an empty store, like the Python version).
func LoadStore() *Store {
	s := newStore()
	data, err := os.ReadFile(StateFilePath())
	if err != nil {
		return s
	}
	var f storeFile
	if err := json.Unmarshal(data, &f); err != nil {
		return s
	}
	for _, k := range f.UsedKeys {
		s.UsedKeys[k] = true
	}
	for _, k := range f.CopiedKeys {
		s.CopiedKeys[k] = true
	}
	for k, v := range f.TestedKeysOK {
		s.TestedKeysOK[k] = v
	}
	for _, k := range f.AgentLoadedKeys {
		s.AgentLoadedKeys[k] = true
	}
	return s
}

// Save writes the state file atomically.
func (s *Store) Save() error {
	f := storeFile{
		UsedKeys:        sortedKeysOf(s.UsedKeys),
		CopiedKeys:      sortedKeysOf(s.CopiedKeys),
		TestedKeysOK:    s.TestedKeysOK,
		AgentLoadedKeys: sortedKeysOf(s.AgentLoadedKeys),
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	tmp := StateFilePath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, StateFilePath())
}

func newStore() *Store {
	return &Store{
		UsedKeys:        map[string]bool{},
		CopiedKeys:      map[string]bool{},
		TestedKeysOK:    map[string]bool{},
		AgentLoadedKeys: map[string]bool{},
	}
}

func sortedKeysOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
