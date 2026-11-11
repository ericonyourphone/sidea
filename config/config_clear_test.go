package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClearRemovesKeyAndKeepsOthers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"theme":"nord","art":"kitty","custom":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := clearIn(path, "theme"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	s := string(got)
	if strings.Contains(s, "theme") || !strings.Contains(s, "kitty") || !strings.Contains(s, "custom") {
		t.Fatalf("unexpected config after Clear: %s", s)
	}
	if err := clearIn(filepath.Join(t.TempDir(), "missing.json"), "theme"); err != nil {
		t.Fatalf("Clear on a missing file should be a no-op: %v", err)
	}
}

func TestWriteFileAtomicLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")
	for i := 0; i < 3; i++ {
		if err := WriteFileAtomic(path, []byte("{}")); err != nil {
			t.Fatal(err)
		}
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("want only token.json, got %d entries", len(entries))
	}
}
