package mcpserver

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"ezruantu/internal/core"
	"ezruantu/internal/exportbundle"
	"ezruantu/internal/raster"
	"ezruantu/internal/settings"
	"ezruantu/internal/workspace"
)

var diagramTypes = []string{"er", "table", "usecase", "flowchart", "sequence", "class", "architecture", "structure"}

type projectService struct {
	root         string
	store        *workspace.Store
	settingsRoot string
}

func (s *projectService) applyGlobalStyle(diagram *core.Diagram) error {
	p, err := settings.Load(s.settingsRoot)
	if err != nil {
		return err
	}
	diagram.View.Theme = p.Theme
	return nil
}

func newProjectService(root string) (*projectService, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err = os.MkdirAll(abs, 0755); err != nil {
		return nil, err
	}
	s := &projectService{root: abs, store: workspace.New(abs), settingsRoot: abs}
	if _, err = s.internalDir(""); err != nil {
		return nil, err
	}
	// An empty workspace stays empty; example diagrams are never presented as
	// completed design work for the user's actual software project.
	err = s.store.Initialize(core.Project{}, false)
	return s, err
}

func (s *projectService) internalDir(name string) (string, error) {
	dir := s.root
	parts := []string{workspace.Internal}
	if name != "" {
		parts = append(parts, name)
	}
	for _, part := range parts {
		dir = filepath.Join(dir, part)
		info, err := os.Lstat(dir)
		if err == nil && (info.Mode()&os.ModeSymlink != 0 || !info.IsDir()) {
			return "", fmt.Errorf("内部目录不能是文件或链接")
		}
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}
		if err = os.MkdirAll(dir, 0755); err != nil {
			return "", err
		}
	}
	return dir, nil
}

func flattenEntries(entries []workspace.Entry) []workspace.Entry {
	result := []workspace.Entry{}
	for _, entry := range entries {
		result = append(result, entry)
		result = append(result, flattenEntries(entry.Children)...)
	}
	return result
}

func normalizedPath(p string) string { return strings.ReplaceAll(p, `\`, "/") }

func (s *projectService) selectedFiles(folder string, paths []string) ([]workspace.Entry, error) {
	tree, err := s.store.List()
	if err != nil {
		return nil, err
	}
	entries := flattenEntries(tree.Entries)
	folder = normalizedPath(folder)
	if len(paths) > 0 && folder != "" {
		return nil, fmt.Errorf("path（项目文件夹）与 paths（指定图表）只能选择一个")
	}
	if len(paths) > 0 {
		index := map[string]workspace.Entry{}
		for _, entry := range entries {
			index[entry.Path] = entry
		}
		result, seen := []workspace.Entry{}, map[string]bool{}
		for _, name := range paths {
			name = normalizedPath(name)
			entry, ok := index[name]
			if !ok || entry.Kind != "file" || entry.DiagramType == "" {
				return nil, fmt.Errorf("找不到可编辑图表：%s", name)
			}
			if !seen[name] {
				seen[name] = true
				result = append(result, entry)
			}
		}
		return result, nil
	}
	if folder != "" {
		found := false
		for _, entry := range entries {
			if entry.Path == folder && entry.Kind == "folder" {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("项目文件夹不存在：%s", folder)
		}
	}
	result := []workspace.Entry{}
	for _, entry := range entries {
		if entry.Kind == "file" && entry.DiagramType != "" && (folder == "" || strings.HasPrefix(entry.Path, folder+"/")) {
			result = append(result, entry)
		}
	}
	return result, nil
}

func (s *projectService) inspect(folder string) (any, error) {
	files, err := s.selectedFiles(folder, nil)
	if err != nil {
		return nil, err
	}
	tree, err := s.store.List()
	if err != nil {
		return nil, err
	}
	folders, summaries, errors := []string{}, []any{}, []string{}
	found := map[string]bool{}
	folder = normalizedPath(folder)
	for _, entry := range flattenEntries(tree.Entries) {
		if entry.Kind == "folder" && (folder == "" || entry.Path == folder || strings.HasPrefix(entry.Path, folder+"/")) {
			folders = append(folders, entry.Path)
		}
	}
	for _, file := range files {
		doc, readErr := s.store.Read(file.Path)
		if readErr != nil {
			errors = append(errors, file.Path+"："+readErr.Error())
			continue
		}
		found[doc.Diagram.Type] = true
		_, validation := core.ValidateJSON(jsonBytes(doc.Diagram))
		summaries = append(summaries, map[string]any{"path": doc.Path, "title": doc.Diagram.Title, "type": doc.Diagram.Type, "revision": doc.Revision, "items": len(doc.Diagram.Items()), "issues": validation.Issues})
	}
	missing := []string{}
	for _, kind := range diagramTypes {
		if !found[kind] {
			missing = append(missing, kind)
		}
	}
	return map[string]any{"workspace": s.root, "path": folder, "folders": folders, "diagrams": summaries, "count": len(files), "errors": errors, "types_not_present": missing, "supported_types": diagramTypes, "png_available": raster.Browser() != "", "note": "未出现的图种仅供参考，按实际项目需要生成；不要把示例模板直接当成项目成品。"}, nil
}

type SaveDiagramArgs struct {
	Path     string         `json:"path" jsonschema:"Workspace-relative .json path, e.g. 图书系统/用例图.json. Parent folder must exist."`
	Diagram  map[string]any `json:"diagram" jsonschema:"Complete valid diagram JSON object. Preserve existing IDs when editing."`
	Revision string         `json:"revision,omitempty" jsonschema:"Required when updating an existing diagram: use the revision returned by read_diagram. Omit for a new file."`
}

func (s *projectService) save(args SaveDiagramArgs) (any, error) {
	args.Path = normalizedPath(args.Path)
	if !strings.HasSuffix(strings.ToLower(args.Path), ".json") {
		return nil, fmt.Errorf("图表路径必须以 .json 结尾")
	}
	raw := jsonBytes(args.Diagram)
	_, validation := core.ValidateJSON(raw)
	if !validation.Valid {
		return nil, fmt.Errorf("图表无效：%s", validation.Issues[0].Message)
	}
	previous, err := s.store.Read(args.Path)
	backup := ""
	var doc workspace.Document
	if os.IsNotExist(err) {
		if args.Revision != "" {
			return nil, fmt.Errorf("目标文件不存在，不能用旧版本创建文件")
		}
		parent := path.Dir(args.Path)
		if parent == "." {
			parent = ""
		}
		doc, err = s.store.CreateFile(parent, path.Base(args.Path), string(raw))
	} else if err == nil {
		if args.Revision == "" || args.Revision != previous.Revision {
			return nil, fmt.Errorf("文件版本已变更或缺少 revision；请重新 read_diagram 后再修改，现有内容未覆盖")
		}
		backups, mkdirErr := s.internalDir("mcp-backups")
		if mkdirErr != nil {
			return nil, mkdirErr
		}
		backup = filepath.Join(backups, previous.Revision+".json")
		if err = core.AtomicWrite(backup, jsonBytes(previous.Diagram)); err != nil {
			return nil, err
		}
		doc, err = s.store.Write(args.Path, string(raw), args.Revision)
	}
	if err != nil {
		return nil, err
	}
	return map[string]any{"path": doc.Path, "revision": doc.Revision, "backup_path": backup, "validation": validation}, nil
}

type ExportProjectArgs struct {
	Path    string   `json:"path,omitempty" jsonschema:"Project folder relative to workspace. Empty means all editable diagrams, recursively. Do not combine with paths."`
	Paths   []string `json:"paths,omitempty" jsonschema:"Optional exact diagram paths to export instead of a whole folder."`
	Format  string   `json:"format" jsonschema:"png, svg, drawio, or json"`
	Outputs []string `json:"outputs,omitempty" jsonschema:"For multiple diagrams: folder and/or zip. Omit to create a folder. A single diagram always produces one file."`
}

func exportBytes(ctx context.Context, d core.Diagram, format string) ([]byte, error) {
	if format == "png" {
		r := core.Render(string(jsonBytes(d)))
		if !r.Validation.Valid {
			return nil, fmt.Errorf("%s", r.Validation.Issues[0].Message)
		}
		return raster.PNG(ctx, r.SVG, r.Scene.Width, r.Scene.Height)
	}
	text, err := core.Export(string(jsonBytes(d)), format)
	return []byte(text), err
}

func writeUnique(dir, name string, data []byte) (string, error) {
	base, ext := strings.TrimSuffix(name, filepath.Ext(name)), filepath.Ext(name)
	for i := 1; i < 10000; i++ {
		candidate := name
		if i > 1 {
			candidate = fmt.Sprintf("%s (%d)%s", base, i, ext)
		}
		p := filepath.Join(dir, candidate)
		f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if os.IsExist(err) {
			continue
		}
		if err != nil {
			return "", err
		}
		_, err = f.Write(data)
		closeErr := f.Close()
		if err != nil || closeErr != nil {
			os.Remove(p)
			return "", fmt.Errorf("导出写入失败：%v %v", err, closeErr)
		}
		return p, nil
	}
	return "", fmt.Errorf("同名导出结果过多")
}

func (s *projectService) export(ctx context.Context, args ExportProjectArgs, progress func(int, int)) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	switch args.Format {
	case "png", "svg", "drawio", "json":
	default:
		return nil, fmt.Errorf("format 必须是 png、svg、drawio 或 json")
	}
	if args.Format == "png" && raster.Browser() == "" {
		return nil, fmt.Errorf("本机没有可用的 Edge/Chrome，PNG 不可用；请改用 SVG")
	}
	files, err := s.selectedFiles(args.Path, args.Paths)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("所选范围没有可导出的图表")
	}
	folder, zipped := true, false
	if args.Outputs != nil && len(args.Outputs) == 0 && len(files) > 1 {
		return nil, fmt.Errorf("请至少选择一种输出方式：folder 或 zip")
	}
	if len(args.Outputs) > 0 {
		folder = false
		for _, mode := range args.Outputs {
			switch mode {
			case "folder":
				folder = true
			case "zip":
				zipped = true
			default:
				return nil, fmt.Errorf("outputs 只支持 folder、zip")
			}
		}
	}
	// Read and validate every document before starting image rendering or output.
	preferences, err := settings.Load(s.settingsRoot)
	if err != nil {
		return nil, err
	}
	docs := make([]workspace.Document, len(files))
	for i, f := range files {
		docs[i], err = s.store.Read(f.Path)
		if err != nil {
			return nil, fmt.Errorf("%s：%w", f.Path, err)
		}
		docs[i].Diagram.View.Theme = preferences.Theme
	}
	prepared, total := []exportbundle.File{}, 0
	var single []byte
	for i, doc := range docs {
		if err = ctx.Err(); err != nil {
			return nil, err
		}
		data, renderErr := exportBytes(ctx, doc.Diagram, args.Format)
		if renderErr != nil {
			return nil, fmt.Errorf("%s：%w", doc.Path, renderErr)
		}
		total += len(data)
		if total > exportbundle.MaxBytes {
			return nil, fmt.Errorf("导出超过 128 MB，请缩小范围")
		}
		relative := doc.Path
		if args.Path != "" {
			relative = strings.TrimPrefix(relative, normalizedPath(args.Path)+"/")
		}
		relative = strings.TrimSuffix(relative, path.Ext(relative)) + "." + args.Format
		prepared = append(prepared, exportbundle.File{Path: relative, Data: base64.StdEncoding.EncodeToString(data)})
		if len(docs) == 1 {
			single = data
		}
		if progress != nil {
			progress(i+1, len(docs))
		}
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	destination, err := s.internalDir("exports")
	if err != nil {
		return nil, err
	}
	results := []string{}
	if len(docs) == 1 {
		p, writeErr := writeUnique(destination, path.Base(prepared[0].Path), single)
		if writeErr != nil {
			return nil, writeErr
		}
		results = append(results, p)
	} else {
		name := "全部图表"
		if args.Path != "" {
			name = path.Base(normalizedPath(args.Path))
		} else if len(args.Paths) > 0 {
			name = "所选图表"
		}
		results, err = exportbundle.Write(destination, name+"-"+args.Format, prepared, folder, zipped)
		if err != nil {
			return nil, err
		}
	}
	sources := []string{}
	for _, doc := range docs {
		sources = append(sources, doc.Path)
	}
	return map[string]any{"count": len(docs), "format": args.Format, "sources": sources, "paths": results, "single_file": len(docs) == 1}, nil
}
