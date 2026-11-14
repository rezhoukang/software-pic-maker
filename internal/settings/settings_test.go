package settings

import (
	"os"
	"path/filepath"
	"testing"

	"ezruantu/internal/workspace"
)

func TestGlobalThemePersistsAndRejectsInvalidUpdates(t *testing.T) {
	root := t.TempDir()
	p, err := Load(root)
	if err != nil || p.Theme != "blue" {
		t.Fatal(p, err)
	}
	for _, theme := range []string{"mono", "sage", "blue"} {
		if err := SetTheme(root, theme); err != nil {
			t.Fatal(err)
		}
		p, err = Load(root)
		if err != nil || p.Theme != theme {
			t.Fatal(p, err)
		}
	}
	if err := SetTheme(root, "unknown"); err == nil {
		t.Fatal("invalid theme accepted")
	}
	p, err = Load(root)
	if err != nil || p.Theme != "blue" {
		t.Fatal("invalid update replaced valid settings", p, err)
	}
	if err := os.WriteFile(filepath.Join(root, workspace.Internal, "settings.json"), []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil {
		t.Fatal("corrupt settings were silently accepted")
	}
}
