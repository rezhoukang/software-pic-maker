package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"tuxu/internal/core"
)

func TestCloseConfirmation(t *testing.T) {
	app := NewApp()
	app.ctx = context.Background()
	prompts, quits := 0, 0
	app.requestClose = func(context.Context) { prompts++ }
	app.quit = func(ctx context.Context) {
		quits++
		if app.beforeClose(ctx) {
			t.Fatal("confirmed quit was blocked by OnBeforeClose")
		}
	}
	if app.beforeClose(app.ctx) || prompts != 0 {
		t.Fatal("clean project should close without confirmation")
	}
	app.SetDirty(true)
	if !app.beforeClose(app.ctx) || prompts != 1 || quits != 0 {
		t.Fatal("unsaved project must wait for confirmation")
	}
	// Cancelling does not authorize any later attempt to close.
	if !app.beforeClose(app.ctx) || prompts != 2 {
		t.Fatal("cancelled close must leave the project protected")
	}
	app.ConfirmQuit()
	if quits != 1 || prompts != 2 {
		t.Fatal("confirm must quit without requesting confirmation again")
	}
	// Late frontend dirty updates must not re-block the confirmed close.
	app.SetDirty(true)
	if app.beforeClose(app.ctx) {
		t.Fatal("late dirty update blocked a confirmed quit")
	}
}

func TestRecoveryKeepsLastValidProject(t *testing.T) {
	app := &App{dataDir: t.TempDir()}
	project := core.Project{SchemaVersion: "1.0", Name: "有效恢复副本", Diagrams: core.Templates()}
	b, _ := json.Marshal(project)
	if err := app.SaveRecovery(string(b), "example.tuxu.json"); err != nil {
		t.Fatal(err)
	}
	project.Diagrams[0].Model.Relations[0].Target = "missing"
	b, _ = json.Marshal(project)
	if err := app.SaveRecovery(string(b), "example.tuxu.json"); err == nil {
		t.Fatal("invalid recovery accepted")
	}
	session := app.GetSession()
	if !session.Recovered || session.Project.Name != "有效恢复副本" || session.Project.Diagrams[0].Model.Relations[0].Target == "missing" {
		t.Fatal("valid recovery overwritten")
	}
}
func TestCorruptRecoveryFallsBackToExamples(t *testing.T) {
	app := &App{dataDir: t.TempDir()}
	if err := os.WriteFile(filepath.Join(app.dataDir, "recovery.json"), []byte(`{"project":{"diagrams":[{"type":"unsupported"}]}}`), 0600); err != nil {
		t.Fatal(err)
	}
	session := app.GetSession()
	if session.Recovered || session.RecoveryError == "" || len(session.Project.Diagrams) != 8 {
		t.Fatalf("bad fallback: %+v", session)
	}
}

func TestWorkspaceMigratesLegacyRecoveryWithoutChangingOriginal(t *testing.T) {
	legacy := t.TempDir()
	data := filepath.Join(t.TempDir(), "data")
	project := core.Project{SchemaVersion: "1.0", Name: "旧项目", Diagrams: core.Templates()[:1]}
	project.Diagrams[0].Title = "尚未导出的旧图"
	contents, _ := json.Marshal(Session{Project: project})
	old := filepath.Join(legacy, "recovery.json")
	if e := os.WriteFile(old, contents, 0644); e != nil {
		t.Fatal(e)
	}
	app := &App{dataDir: data, legacyDataDir: legacy}
	tree, e := app.ListWorkspace()
	if e != nil {
		t.Fatal(e)
	}
	if len(tree.Entries) != 1 || tree.Entries[0].Name != "恢复的项目" || len(tree.Entries[0].Children) != 1 {
		t.Fatal("legacy recovery was not migrated", tree)
	}
	doc, e := app.ReadWorkspaceFile(tree.Entries[0].Children[0].Path)
	if e != nil || doc.Diagram.Title != "尚未导出的旧图" {
		t.Fatal("recovered content changed", e)
	}
	original, _ := os.ReadFile(old)
	if string(original) != string(contents) {
		t.Fatal("original recovery was modified")
	}
}

func TestProjectDataRootAndPortableDataRoot(t *testing.T) {
	for _, name := range []string{"EZ软图", "EZ毕图"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			os.WriteFile(filepath.Join(root, "wails.json"), []byte(`{"name":"`+name+`"}`), 0644)
			os.WriteFile(filepath.Join(root, "go.mod"), []byte("module tuxu"), 0644)
			if got := projectDataDirectory(filepath.Join(root, "dist", "release", name+".exe")); got != filepath.Join(root, "data") {
				t.Fatal("source build did not share project data", got)
			}
			portable := t.TempDir()
			if got := projectDataDirectory(filepath.Join(portable, name+".exe")); got != filepath.Join(portable, "data") {
				t.Fatal("portable data not beside executable", got)
			}
		})
	}
}
