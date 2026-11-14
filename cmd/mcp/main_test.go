package main

import (
	"context"
	"encoding/json"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"os"
	"path/filepath"
	"testing"
	"time"
	"tuxu/internal/core"
)

func TestMCPRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	root := t.TempDir()
	ct, st := mcp.NewInMemoryTransports()
	server, err := newServer(root)
	if err != nil {
		t.Fatal(err)
	}
	ss, err := server.Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ss.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()
	listed, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.Tools) != 13 {
		t.Fatal(len(listed.Tools))
	}
	call := func(name string, args any) *mcp.CallToolResult {
		t.Helper()
		r, e := cs.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		if e != nil {
			t.Fatal(e)
		}
		if r.IsError {
			t.Fatalf("%s: %+v", name, r.Content)
		}
		return r
	}
	call("get_diagram_schema", map[string]any{})
	call("list_templates", map[string]any{"type": ""})
	for _, d := range core.Templates() {
		t.Run(d.Type, func(t *testing.T) {
			call("validate_diagram", map[string]any{"diagram": d})
			call("render_diagram", map[string]any{"diagram": d})
			for _, f := range []string{"svg", "drawio", "json"} {
				name := d.Type + "." + f
				call("export_diagram", map[string]any{"diagram": d, "format": f, "filename": name})
				if info, e := os.Stat(filepath.Join(root, name)); e != nil || info.Size() == 0 {
					t.Fatalf("export missing %s %v", name, e)
				}
			}
		})
	}
	p := core.Project{SchemaVersion: "1.0", Name: "MCP 项目", Diagrams: core.Templates()}
	call("save_project", map[string]any{"project": p, "filename": "project.tuxu.json"})
	r := call("open_project", map[string]any{"filename": "project.tuxu.json"})
	b, _ := json.Marshal(r.StructuredContent)
	if _, err := core.ParseProject(b); err != nil {
		t.Fatal(err, string(b))
	}
	r, err = cs.CallTool(ctx, &mcp.CallToolParams{Name: "export_diagram", Arguments: map[string]any{"diagram": p.Diagrams[0], "format": "svg", "filename": "../escape.svg"}})
	if err != nil {
		t.Fatal(err)
	}
	if !r.IsError {
		t.Fatal("accepted traversal")
	}
}
func TestOutputPath(t *testing.T) {
	for _, name := range []string{"", "..", "../x", "a/b", "a\\b", "C:x"} {
		if _, e := outputPath(t.TempDir(), name); e == nil {
			t.Fatal(name)
		}
	}
}
