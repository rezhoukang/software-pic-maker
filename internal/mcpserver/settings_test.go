package mcpserver

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ezruantu/internal/core"
	"ezruantu/internal/settings"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPUsesLiveGlobalStyleFromInstallationForScopedWorkspace(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project")
	server, err := newServerWithSettings(project, root)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	ss, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ss.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "global-style-test", Version: "1"}, nil)
	cs, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()
	d := core.Templates()[0]
	data, _ := json.Marshal(d)
	file := filepath.Join(project, "diagram.json")
	if err := os.WriteFile(file, data, 0600); err != nil {
		t.Fatal(err)
	}
	for _, theme := range []string{"mono", "sage"} {
		if err := settings.SetTheme(root, theme); err != nil {
			t.Fatal(err)
		}
		r, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "render_diagram", Arguments: map[string]any{"diagram": d}})
		if err != nil || r.IsError {
			t.Fatalf("render: %v %+v", err, r)
		}
		encoded, _ := json.Marshal(r.StructuredContent)
		var rendered struct {
			Scene core.Scene `json:"scene"`
		}
		if err := json.Unmarshal(encoded, &rendered); err != nil || rendered.Scene.Theme != theme {
			t.Fatal(string(encoded), err)
		}
		r, err = cs.CallTool(ctx, &mcp.CallToolParams{Name: "export_project", Arguments: map[string]any{"format": "json"}})
		if err != nil || r.IsError {
			t.Fatalf("export: %v %+v", err, r)
		}
		encoded, _ = json.Marshal(r.StructuredContent)
		var exported struct {
			Paths []string `json:"paths"`
		}
		if err := json.Unmarshal(encoded, &exported); err != nil || len(exported.Paths) != 1 {
			t.Fatal(string(encoded), err)
		}
		output, err := os.ReadFile(exported.Paths[0])
		if err != nil || !strings.Contains(string(output), `"theme": "`+theme+`"`) {
			t.Fatal(string(output), err)
		}
	}
	unchanged, err := os.ReadFile(file)
	if err != nil || string(unchanged) != string(data) {
		t.Fatal("global styling modified source data", err)
	}
}
