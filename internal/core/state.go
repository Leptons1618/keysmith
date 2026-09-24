package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

// StateFilePath resolves the state file path per call so it always follows
// the current HOME (tests swap HOME; a package-level var would freeze the
// real user's path at init). The schema remains stable across GUI and TUI.
func StateFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".ssh-key-gui-state.json"
	}
	return filepath.Join(home, ".ssh-key-gui-state.json")
}

// Store holds persisted per-key workflow markers shared by both frontends.
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

// LoadStore reads the state file, tolerating absence and corruption.
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

// MarkCopied records a successful public-key copy and persists the change.
func (s *Store) MarkCopied(keyName string) error {
	return s.setBool(&s.CopiedKeys, keyName, true)
}

// MarkAgentLoaded records a successful agent load and persists the change.
func (s *Store) MarkAgentLoaded(keyName string) error {
	return s.setBool(&s.AgentLoadedKeys, keyName, true)
}

// RecordTest records the latest connection-test result for a key.
func (s *Store) RecordTest(keyName string, ok bool) error {
	return s.setBool(&s.TestedKeysOK, keyName, ok)
}

// MarkUsed records that a key was added to a service account.
func (s *Store) MarkUsed(keyName string) error {
	return s.setBool(&s.UsedKeys, keyName, true)
}

// ForgetKey removes all workflow markers for a deleted key and persists the
// change. The in-memory store is restored if persistence fails.
func (s *Store) ForgetKey(keyName string) error {
	used, hadUsed := s.UsedKeys[keyName]
	copied, hadCopied := s.CopiedKeys[keyName]
	tested, hadTested := s.TestedKeysOK[keyName]
	loaded, hadLoaded := s.AgentLoadedKeys[keyName]
	delete(s.UsedKeys, keyName)
	delete(s.CopiedKeys, keyName)
	delete(s.TestedKeysOK, keyName)
	delete(s.AgentLoadedKeys, keyName)
	if err := s.Save(); err != nil {
		if hadUsed {
			s.UsedKeys[keyName] = used
		}
		if hadCopied {
			s.CopiedKeys[keyName] = copied
		}
		if hadTested {
			s.TestedKeysOK[keyName] = tested
		}
		if hadLoaded {
			s.AgentLoadedKeys[keyName] = loaded
		}
		return err
	}
	return nil
}

func (s *Store) setBool(target *map[string]bool, keyName string, value bool) error {
	if *target == nil {
		*target = map[string]bool{}
	}
	old, existed := (*target)[keyName]
	(*target)[keyName] = value
	if err := s.Save(); err != nil {
		if existed {
			(*target)[keyName] = old
		} else {
			delete(*target, keyName)
		}
		return err
	}
	return nil
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
