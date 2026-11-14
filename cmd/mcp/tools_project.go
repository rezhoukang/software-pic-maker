package main

import (
	"context"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const projectInstructions = `EZ软图 is a local software-design diagram service. Work in project folders inside the configured workspace.
For "export all existing software design diagrams": inspect_project to identify the project folder, then export_project. Do not silently export unrelated projects.
For a broad "output all design diagrams for this software project" request: inspect existing work first. If diagrams still need designing, or the workspace is empty, use the actual source/requirements available to the host client, author the appropriate project-specific diagram set, save it, then export. Do not stop at an empty file list when the user requested creation; ask only for genuinely missing project requirements.
For "design a diagram" or "create all diagrams for this software": the host client reads the actual source code/requirements with its own tools, inspects existing diagrams, fetches get_diagram_schema and relevant list_templates, authors real project-specific content, validates it, creates folders as needed, and calls save_diagram. Templates are examples, never finished project designs.
For edits: read_diagram first; preserve stable IDs. Use patch_diagram for small changes or save_diagram for complete replacements, always supplying the read revision. Invalid or stale edits do not write. Successful edits have backups.
export_project supports png/svg/drawio/json, a project folder or selected paths, recursive subfolders, and folder/zip output. One diagram produces a file; multiple diagrams default to a folder. Exports use unique names under .ezbitu/exports and never re-enter source scans.
PNG uses installed local Edge/Chrome in hidden headless mode; inspect_project reports png_available. No live desktop synchronization is required. Report actual output paths and validation issues. Legacy save_project/open_project are only for aggregate project JSON.`

type ProjectPathArgs struct {
	Path string `json:"path,omitempty" jsonschema:"Project folder relative to workspace; empty means root"`
}
type DiagramPathArgs struct {
	Path string `json:"path" jsonschema:"Exact workspace-relative .json diagram path"`
}
type CreateFolderArgs struct {
	Parent string `json:"parent,omitempty" jsonschema:"Existing parent folder; empty means workspace root"`
	Name   string `json:"name" jsonschema:"New project or subfolder name, without path separators"`
}

func projectTool(name, description string, readOnly bool) *mcp.Tool {
	closed, destructive := false, !readOnly
	return &mcp.Tool{Name: name, Description: description, Annotations: &mcp.ToolAnnotations{ReadOnlyHint: readOnly, DestructiveHint: &destructive, OpenWorldHint: &closed}}
}
func registerProjectTools(server *mcp.Server, service *projectService) {
	mcp.AddTool(server, projectTool("inspect_project", "Discover project folders, diagram paths/titles/types/revisions, validation issues and export capabilities. Use first for all-diagram requests. Does not inspect software source code or invent business logic.", true), func(ctx context.Context, req *mcp.CallToolRequest, args ProjectPathArgs) (*mcp.CallToolResult, any, error) {
		result, err := service.inspect(args.Path)
		return nil, result, err
	})
	additive := false
	tool := projectTool("create_folder", "Create a project folder or subfolder. Never replaces an existing directory. Use before saving in a new project.", false)
	tool.Annotations.DestructiveHint = &additive
	mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, args CreateFolderArgs) (*mcp.CallToolResult, any, error) {
		p, err := service.store.CreateFolder(args.Parent, args.Name)
		return nil, map[string]any{"path": p}, err
	})
	mcp.AddTool(server, projectTool("read_diagram", "Read editable diagram JSON and revision. Read before modifying, then pass revision to save_diagram or patch_diagram.", true), func(ctx context.Context, req *mcp.CallToolRequest, args DiagramPathArgs) (*mcp.CallToolResult, any, error) {
		doc, err := service.store.Read(args.Path)
		return nil, doc, err
	})
	mcp.AddTool(server, projectTool("save_diagram", "Validate and create/update a diagram in project subfolders. Omit revision only for NEW files. Updates require read_diagram revision and preserve stable IDs. Backs up edits; rejects conflicts.", false), func(ctx context.Context, req *mcp.CallToolRequest, args SaveDiagramArgs) (*mcp.CallToolResult, any, error) {
		result, err := service.save(args)
		return nil, result, err
	})
	mcp.AddTool(server, projectTool("patch_diagram", "Atomic RFC 6902 add/replace/remove/test content edits. Read first and supply revision. Validate the complete result before writing. Invalid operations/references or stale revisions leave the file unchanged.", false), func(ctx context.Context, req *mcp.CallToolRequest, args PatchDiagramArgs) (*mcp.CallToolResult, any, error) {
		result, err := service.patch(args)
		return nil, result, err
	})
	export := projectTool("export_project", "Export all diagrams of a project folder recursively, all workspace diagrams, or selected paths. Formats png/svg/drawio/json. Single diagram produces a file; multiple diagrams produce folder and/or ZIP. Preserves hierarchy and existing outputs. Returns actual paths.", false)
	export.Annotations.DestructiveHint = &additive
	mcp.AddTool(server, export, func(ctx context.Context, req *mcp.CallToolRequest, args ExportProjectArgs) (*mcp.CallToolResult, any, error) {
		result, err := service.export(ctx, args, func(done, total int) {
			if token := req.Params.GetProgressToken(); token != nil {
				_ = req.Session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{ProgressToken: token, Progress: float64(done), Total: float64(total), Message: "正在导出软件设计图"})
			}
		})
		return nil, result, err
	})
}
