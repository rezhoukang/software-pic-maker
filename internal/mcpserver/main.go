package mcpserver

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ezruantu/internal/core"
	"ezruantu/shared"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Empty struct{}
type DiagramArgs struct {
	Diagram map[string]any `json:"diagram" jsonschema:"A diagram object following the shared v1 JSON Schema. Get get_diagram_schema first."`
}
type ExportArgs struct {
	Diagram   map[string]any `json:"diagram"`
	Format    string         `json:"format" jsonschema:"png, svg, drawio, or json"`
	Filename  string         `json:"filename" jsonschema:"A filename within the configured workspace, no directories"`
	Overwrite bool           `json:"overwrite,omitempty" jsonschema:"Explicitly replace an existing export; default false"`
}
type TemplateArgs struct {
	Type string `json:"type" jsonschema:"er, table, usecase, flowchart, sequence, class, architecture, structure; empty lists all"`
}
type ProjectArgs struct {
	Project   map[string]any `json:"project"`
	Filename  string         `json:"filename"`
	Overwrite bool           `json:"overwrite,omitempty" jsonschema:"Explicitly replace a legacy project JSON; use save_diagram for normal diagram edits"`
}
type OpenArgs struct {
	Filename string `json:"filename"`
}

func jsonBytes(v any) []byte { b, _ := json.Marshal(v); return b }
func newServer(workspace string) (*mcp.Server, error) {
	service, err := newProjectService(workspace)
	if err != nil {
		return nil, err
	}
	s := mcp.NewServer(&mcp.Implementation{Name: "ezruantu", Version: "0.2.14"}, &mcp.ServerOptions{Instructions: projectInstructions})
	mcp.AddTool(s, &mcp.Tool{Name: "get_diagram_schema", Description: "Get the shared JSON Schema for all supported diagram types. All changes must preserve valid references and stable IDs.", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}}, func(ctx context.Context, req *mcp.CallToolRequest, args Empty) (*mcp.CallToolResult, any, error) {
		var schema any
		json.Unmarshal(shared.DiagramSchema, &schema)
		return nil, schema, nil
	})
	mcp.AddTool(s, &mcp.Tool{Name: "list_templates", Description: "Get editable example diagram JSON for eight diagram types.", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}}, func(ctx context.Context, req *mcp.CallToolRequest, args TemplateArgs) (*mcp.CallToolResult, any, error) {
		ds := []core.Diagram{}
		for _, d := range core.Templates() {
			if args.Type == "" || args.Type == d.Type {
				ds = append(ds, d)
			}
		}
		return nil, map[string]any{"templates": ds}, nil
	})
	mcp.AddTool(s, &mcp.Tool{Name: "validate_diagram", Description: "Validate structure, duplicate IDs, references, use case relations, and inheritance cycles. Returns errors and warnings.", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}}, func(ctx context.Context, req *mcp.CallToolRequest, args DiagramArgs) (*mcp.CallToolResult, any, error) {
		_, v := core.ValidateJSON(jsonBytes(args.Diagram))
		return nil, v, nil
	})
	mcp.AddTool(s, &mcp.Tool{Name: "render_diagram", Description: "Generate a deterministic layout and SVG without opening the desktop app. Returns SVG and scene geometry; no files are written.", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}}, func(ctx context.Context, req *mcp.CallToolRequest, args DiagramArgs) (*mcp.CallToolResult, any, error) {
		r := core.Render(string(jsonBytes(args.Diagram)))
		if !r.Validation.Valid {
			return nil, nil, fmt.Errorf("%s", r.Validation.Issues[0].Message)
		}
		return nil, map[string]any{"svg": r.SVG, "scene": r.Scene, "validation": r.Validation}, nil
	})
	mcp.AddTool(s, &mcp.Tool{Name: "export_diagram", Description: "Export one supplied diagram as PNG/SVG/drawio/JSON to a plain workspace filename. Existing files require overwrite=true. Prefer export_project for saved diagrams and folders."}, func(ctx context.Context, req *mcp.CallToolRequest, args ExportArgs) (*mcp.CallToolResult, any, error) {
		path, err := outputPath(workspace, args.Filename)
		if err != nil {
			return nil, nil, err
		}
		if filepath.Ext(path) != "."+args.Format {
			return nil, nil, fmt.Errorf("filename extension must match format")
		}
		if _, statErr := os.Lstat(path); statErr == nil && !args.Overwrite {
			return nil, nil, fmt.Errorf("目标已存在，请换文件名或明确指定 overwrite=true")
		}
		diagram, validation := core.ValidateJSON(jsonBytes(args.Diagram))
		if !validation.Valid {
			return nil, nil, fmt.Errorf("%s", validation.Issues[0].Message)
		}
		content, err := exportBytes(ctx, diagram, args.Format)
		if err != nil {
			return nil, nil, err
		}
		if err = core.AtomicWrite(path, content); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"path": path, "bytes": len(content), "format": args.Format}, nil
	})
	mcp.AddTool(s, &mcp.Tool{Name: "save_project", Description: "Legacy aggregate project JSON save. Prefer create_folder/save_diagram for normal projects. Existing files require explicit overwrite=true."}, func(ctx context.Context, req *mcp.CallToolRequest, args ProjectArgs) (*mcp.CallToolResult, any, error) {
		p, err := core.ParseProject(jsonBytes(args.Project))
		if err != nil {
			return nil, nil, err
		}
		path, err := outputPath(workspace, args.Filename)
		if err != nil {
			return nil, nil, err
		}
		if !strings.HasSuffix(path, ".json") {
			return nil, nil, fmt.Errorf("project filename must end in .json")
		}
		if _, statErr := os.Lstat(path); statErr == nil && !args.Overwrite {
			return nil, nil, fmt.Errorf("项目文件已存在；新图请用 save_diagram，覆盖需显式 overwrite=true")
		}
		b, _ := json.MarshalIndent(p, "", "  ")
		if err = core.AtomicWrite(path, b); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"path": path}, nil
	})
	mcp.AddTool(s, &mcp.Tool{Name: "open_project", Description: "Read a project from the configured workspace.", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}}, func(ctx context.Context, req *mcp.CallToolRequest, args OpenArgs) (*mcp.CallToolResult, any, error) {
		path, err := outputPath(workspace, args.Filename)
		if err != nil {
			return nil, nil, err
		}
		info, err := os.Stat(path)
		if err != nil {
			return nil, nil, err
		}
		if info.Size() > 32<<20 {
			return nil, nil, fmt.Errorf("project too large")
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, err
		}
		p, err := core.ParseProject(b)
		return nil, p, err
	})
	registerProjectTools(s, service)
	return s, nil
}
func outputPath(root, name string) (string, error) {
	if name == "" || name == "." || name == ".." || filepath.Base(name) != name || strings.ContainsAny(name, "/\\:") {
		return "", fmt.Errorf("filename must be a plain file name")
	}
	path := filepath.Join(root, name)
	if info, err := os.Lstat(path); err == nil && (info.Mode()&os.ModeSymlink != 0 || info.IsDir()) {
		return "", fmt.Errorf("target must be a regular file")
	}
	return path, nil
}

// Run serves MCP over inherited stdin/stdout without starting the desktop UI.
func Run(ctx context.Context, args []string, defaultWorkspace string) error {
	flags := flag.NewFlagSet("EZ软图 --mcp", flag.ContinueOnError)
	workspace := flags.String("workspace", defaultWorkspace, "Directory used for MCP file operations")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected MCP arguments: %v", flags.Args())
	}
	abs, err := filepath.Abs(*workspace)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(abs, 0755); err != nil {
		return err
	}
	server, err := newServer(abs)
	if err != nil {
		return err
	}
	return server.Run(ctx, &mcp.StdioTransport{})
}
