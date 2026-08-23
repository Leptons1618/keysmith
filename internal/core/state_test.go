package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreRoundTrip(t *testing.T) {
	withTempHOME(t)

	s := LoadStore()
	s.UsedKeys["k1"] = true
	s.CopiedKeys["k1"] = true
	s.TestedKeysOK["k1"] = true
	s.AgentLoadedKeys["k2"] = true
	if err := s.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if _, err := os.Stat(StateFilePath()); err != nil {
		t.Fatalf("state file missing: %v", err)
	}

	s2 := LoadStore()
	if !s2.UsedKeys["k1"] || !s2.CopiedKeys["k1"] || !s2.TestedKeysOK["k1"] || !s2.AgentLoadedKeys["k2"] {
		t.Errorf("round trip lost data: %+v", s2)
	}
}

func TestStoreMissingFileIsEmpty(t *testing.T) {
	withTempHOME(t)
	if err := os.Remove(StateFilePath()); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	s := LoadStore()
	if len(s.UsedKeys) != 0 || len(s.TestedKeysOK) != 0 {
		t.Errorf("expected empty store, got %+v", s)
	}
}

func TestStoreCorruptFileTolerated(t *testing.T) {
	withTempHOME(t)
	if err := os.MkdirAll(filepath.Dir(StateFilePath()), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(StateFilePath(), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := LoadStore() // must not panic
	if len(s.UsedKeys) != 0 {
		t.Errorf("corrupt file should yield empty store, got %+v", s)
	}
}

func TestStoreSchemaMatchesPython(t *testing.T) {
	withTempHOME(t)

	// Write exactly what the Python app writes.
	py := `{
  "used_keys": ["alpha"],
  "copied_keys": ["alpha", "beta"],
  "tested_keys_ok": {"alpha": true},
  "agent_loaded_keys": []
}`
	if err := os.WriteFile(StateFilePath(), []byte(py), 0o600); err != nil {
		t.Fatal(err)
	}

	s := LoadStore()
	if !s.UsedKeys["alpha"] || !s.CopiedKeys["beta"] || !s.TestedKeysOK["alpha"] {
		t.Errorf("python schema not understood: %+v", s)
	}

	// And the Go write must be readable as the same schema.
	s.UsedKeys["gamma"] = true
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(StateFilePath())
	for _, key := range []string{`"used_keys"`, `"copied_keys"`, `"tested_keys_ok"`, `"agent_loaded_keys"`} {
		if !contains(string(data), key) {
			t.Errorf("saved schema missing %s", key)
		}
	}
}
