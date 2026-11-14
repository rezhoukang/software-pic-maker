package main

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"ezruantu/internal/workspace"
)

func (a *App) legacyRecoveryFile() string {
	exe, _ := os.Executable()
	root := filepath.Dir(a.dataDir)
	if projectDataDirectory(exe) != filepath.Join(root, "data") {
		return ""
	}
	paths, _ := filepath.Glob(filepath.Join(root, "dist", "*", "data", "recovery.json"))
	paths = append(paths, filepath.Join(root, "build", "bin", "data", "recovery.json"))
	latest := ""
	var modified time.Time
	for _, p := range paths {
		if info, e := os.Stat(p); e == nil && info.ModTime().After(modified) {
			latest = p
			modified = info.ModTime()
		}
	}
	return latest
}

// In a source checkout every build shares the real project/data directory.
// A portable copy outside the checkout keeps data beside its executable.
func projectDataDirectory(executable string) string {
	for directory := filepath.Dir(executable); ; directory = filepath.Dir(directory) {
		config, err := os.ReadFile(filepath.Join(directory, "wails.json"))
		if err == nil && (strings.Contains(string(config), "EZ软图") || strings.Contains(string(config), "EZ毕图")) {
			if _, err = os.Stat(filepath.Join(directory, "go.mod")); err == nil {
				return filepath.Join(directory, "data")
			}
		}
		if filepath.Dir(directory) == directory {
			break
		}
	}
	return filepath.Join(filepath.Dir(executable), "data")
}
func (a *App) workspaceStore() (*workspace.Store, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.files == nil {
		store := workspace.New(a.dataDir)
		session := a.GetSession()
		if err := store.Initialize(session.Project, session.Recovered); err != nil {
			return nil, err
		}
		a.files = store
	}
	return a.files, nil
}
func (a *App) ListWorkspace() (workspace.Tree, error) {
	s, e := a.workspaceStore()
	if e != nil {
		return workspace.Tree{}, e
	}
	return s.List()
}
func (a *App) ReadWorkspaceFile(path string) (workspace.Document, error) {
	s, e := a.workspaceStore()
	if e != nil {
		return workspace.Document{}, e
	}
	return s.Read(path)
}
func (a *App) CreateWorkspaceFolder(parent, name string) (string, error) {
	s, e := a.workspaceStore()
	if e != nil {
		return "", e
	}
	return s.CreateFolder(parent, name)
}
func (a *App) CreateWorkspaceFile(parent, name, raw string) (workspace.Document, error) {
	s, e := a.workspaceStore()
	if e != nil {
		return workspace.Document{}, e
	}
	return s.CreateFile(parent, name, raw)
}
func (a *App) WriteWorkspaceFile(path, raw, revision string) (workspace.Document, error) {
	s, e := a.workspaceStore()
	if e != nil {
		return workspace.Document{}, e
	}
	return s.Write(path, raw, revision)
}
func (a *App) RenameWorkspaceEntry(path, name string) (string, error) {
	s, e := a.workspaceStore()
	if e != nil {
		return "", e
	}
	return s.Rename(path, name)
}
func (a *App) DeleteWorkspaceEntry(path string) error {
	s, e := a.workspaceStore()
	if e != nil {
		return e
	}
	return s.Delete(path)
}
func (a *App) ImportWorkspaceProject(raw string) (string, error) {
	s, e := a.workspaceStore()
	if e != nil {
		return "", e
	}
	return s.ImportProject(raw)
}
