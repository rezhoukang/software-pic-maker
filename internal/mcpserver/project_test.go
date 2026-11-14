package mcpserver

import (
	"archive/zip"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ezruantu/internal/core"
	"ezruantu/internal/raster"
	"ezruantu/internal/workspace"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func mcpClient(t *testing.T, root string) (context.Context, *mcp.ClientSession) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	t.Cleanup(cancel)
	server, err := newServer(root)
	if err != nil {
		t.Fatal(err)
	}
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	session, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })
	client := mcp.NewClient(&mcp.Implementation{Name: "project-workflow-test", Version: "1"}, nil)
	cs, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	return ctx, cs
}
func callData(t *testing.T, ctx context.Context, client *mcp.ClientSession, name string, args any, result any) {
	t.Helper()
	r, err := client.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil || r.IsError {
		t.Fatalf("%s failed: %v %+v", name, err, r)
	}
	if result != nil {
		if err = json.Unmarshal(jsonBytes(r.StructuredContent), result); err != nil {
			t.Fatal(err)
		}
	}
}
func expectToolError(t *testing.T, ctx context.Context, client *mcp.ClientSession, name string, args any) {
	t.Helper()
	r, err := client.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	if !r.IsError {
		t.Fatalf("%s accepted invalid request", name)
	}
}

func TestProjectGenerationEditAndAllDiagramExport(t *testing.T) {
	root := t.TempDir()
	ctx, client := mcpClient(t, root)
	var empty struct {
		Count int `json:"count"`
	}
	callData(t, ctx, client, "inspect_project", map[string]any{}, &empty)
	if empty.Count != 0 {
		t.Fatal("empty project should not be seeded with finished-looking templates")
	}
	callData(t, ctx, client, "create_folder", map[string]any{"name": "图书系统"}, nil)
	callData(t, ctx, client, "create_folder", map[string]any{"parent": "图书系统", "name": "数据库"}, nil)
	for _, d := range core.Templates() {
		p := "图书系统/" + d.Type + ".json"
		if d.Type == "table" {
			p = "图书系统/数据库/table.json"
		}
		callData(t, ctx, client, "save_diagram", map[string]any{"path": p, "diagram": d}, nil)
	}
	var inspection struct {
		Count   int      `json:"count"`
		Missing []string `json:"types_not_present"`
	}
	callData(t, ctx, client, "inspect_project", map[string]any{"path": "图书系统"}, &inspection)
	if inspection.Count != 8 || len(inspection.Missing) != 0 {
		t.Fatal(inspection)
	}
	var doc workspace.Document
	callData(t, ctx, client, "read_diagram", map[string]any{"path": "图书系统/usecase.json"}, &doc)
	oldRevision := doc.Revision
	var changed struct {
		Revision string `json:"revision"`
		Backup   string `json:"backup_path"`
	}
	callData(t, ctx, client, "patch_diagram", map[string]any{"path": doc.Path, "revision": oldRevision, "operations": []any{
		map[string]any{"op": "test", "path": "/model/actors/0/id", "value": "reader"},
		map[string]any{"op": "replace", "path": "/model/actors/0/name", "value": "注册读者"},
		map[string]any{"op": "add", "path": "/model/actors/-", "value": map[string]any{"id": "guest", "name": "访客"}},
	}}, &changed)
	if changed.Revision == oldRevision || changed.Backup == "" {
		t.Fatal("revision or backup missing", changed)
	}
	backup, err := os.ReadFile(changed.Backup)
	if err != nil {
		t.Fatal(err)
	}
	if _, v := core.ValidateJSON(backup); !v.Valid {
		t.Fatal("backup is not editable", v)
	}
	expectToolError(t, ctx, client, "patch_diagram", map[string]any{"path": doc.Path, "revision": oldRevision, "operations": []any{map[string]any{"op": "replace", "path": "/title", "value": "旧版本不能覆盖"}}})
	expectToolError(t, ctx, client, "save_diagram", map[string]any{"path": doc.Path, "diagram": doc.Diagram})
	callData(t, ctx, client, "read_diagram", map[string]any{"path": doc.Path}, &doc)
	if doc.Diagram.Model.Actors[0].Name != "注册读者" || len(doc.Diagram.Model.Actors) != 3 {
		t.Fatal("content edit missing", doc)
	}
	before, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(doc.Path)))
	expectToolError(t, ctx, client, "patch_diagram", map[string]any{"path": doc.Path, "revision": doc.Revision, "operations": []any{
		map[string]any{"op": "replace", "path": "/title", "value": "不应部分保存"}, map[string]any{"op": "remove", "path": "/model/actors/0"},
	}})
	after, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(doc.Path)))
	if string(before) != string(after) {
		t.Fatal("invalid references partially changed the file")
	}
	var exported struct {
		Count int      `json:"count"`
		Paths []string `json:"paths"`
	}
	expectToolError(t, ctx, client, "export_project", map[string]any{"path": "图书系统", "format": "svg", "outputs": []string{}})
	callData(t, ctx, client, "export_project", map[string]any{"path": "图书系统", "format": "svg", "outputs": []string{"folder", "zip"}}, &exported)
	if exported.Count != 8 || len(exported.Paths) != 2 {
		t.Fatal(exported)
	}
	var zipPath, directory string
	for _, p := range exported.Paths {
		if strings.HasSuffix(p, ".zip") {
			zipPath = p
		} else {
			directory = p
		}
	}
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	if len(archive.File) != 8 {
		t.Fatal("not all project diagrams exported")
	}
	if _, err = os.Stat(filepath.Join(directory, "数据库", "table.svg")); err != nil {
		t.Fatal("nested hierarchy lost", err)
	}
	callData(t, ctx, client, "inspect_project", map[string]any{}, &inspection)
	if inspection.Count != 8 {
		t.Fatal("outputs or backups re-entered source scan", inspection.Count)
	}
	firstPaths := append([]string{}, exported.Paths...)
	callData(t, ctx, client, "export_project", map[string]any{"path": "图书系统", "format": "svg", "outputs": []string{"folder", "zip"}}, &exported)
	if firstPaths[0] == exported.Paths[0] {
		t.Fatal("existing export was overwritten")
	}
}

func TestProjectPNGAndSingleFileOutput(t *testing.T) {
	if raster.Browser() == "" {
		t.Skip("PNG requires installed browser")
	}
	root := t.TempDir()
	ctx, client := mcpClient(t, root)
	callData(t, ctx, client, "save_diagram", map[string]any{"path": "时序图.json", "diagram": core.Templates()[4]}, nil)
	var exported struct {
		Count  int      `json:"count"`
		Paths  []string `json:"paths"`
		Single bool     `json:"single_file"`
	}
	callData(t, ctx, client, "export_project", map[string]any{"paths": []string{"时序图.json"}, "format": "png", "outputs": []string{"zip"}}, &exported)
	if !exported.Single || len(exported.Paths) != 1 || !strings.HasSuffix(exported.Paths[0], ".png") {
		t.Fatal(exported)
	}
	data, err := os.ReadFile(exported.Paths[0])
	if err != nil || len(data) < 8 || string(data[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatal("invalid PNG", err)
	}
}

func TestPatchPointersArraysAndInvalidOperations(t *testing.T) {
	root := t.TempDir()
	s, err := newProjectService(root)
	if err != nil {
		t.Fatal(err)
	}
	var diagram map[string]any
	json.Unmarshal(jsonBytes(core.Templates()[1]), &diagram)
	if _, err = s.save(SaveDiagramArgs{Path: "table.json", Diagram: diagram}); err != nil {
		t.Fatal(err)
	}
	doc, _ := s.store.Read("table.json")
	if _, err = s.patch(PatchDiagramArgs{Path: doc.Path, Revision: doc.Revision, Operations: []PatchOperation{
		{Op: "replace", Path: "/model/entities/0/fields/1/type", Value: "varchar(200)"},
		{Op: "add", Path: "/model/entities/0/fields/1", Value: map[string]any{"name": "备注", "type": "text"}},
		{Op: "remove", Path: "/model/entities/0/fields/0"},
	}}); err != nil {
		t.Fatal(err)
	}
	doc, _ = s.store.Read("table.json")
	fields := doc.Diagram.Model.Entities[0].Fields
	if fields[0].Name != "备注" || fields[1].Type != "varchar(200)" {
		t.Fatal(fields)
	}
	for _, p := range []string{"/model/entities/00", "/title/~2", "/model/entities/-/name"} {
		if _, err = s.patch(PatchDiagramArgs{Path: doc.Path, Revision: doc.Revision, Operations: []PatchOperation{{Op: "replace", Path: p, Value: "bad"}}}); err == nil {
			t.Fatal("invalid pointer accepted", p)
		}
	}
	for _, p := range []string{"../escape.json", ".ezbitu/secret.json", "C:/outside.json"} {
		if _, err = s.save(SaveDiagramArgs{Path: p, Diagram: diagram}); err == nil {
			t.Fatal("outside/reserved path accepted", p)
		}
	}
}
