package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"tuxu/internal/core"
	"tuxu/internal/workspace"
)

type App struct {
	ctx            context.Context
	dataDir        string
	legacyDataDir  string
	files          *workspace.Store
	mu             sync.Mutex
	dirty          bool
	closeConfirmed bool
	requestClose   func(context.Context)
	quit           func(context.Context)
}
type Session struct {
	Project       core.Project `json:"project"`
	Path          string       `json:"path"`
	Recovered     bool         `json:"recovered"`
	RecoveryError string       `json:"recovery_error,omitempty"`
	Revision      string       `json:"revision,omitempty"`
}
type Environment struct {
	DataDir  string `json:"data_dir"`
	MCPPath  string `json:"mcp_path"`
	MCPReady bool   `json:"mcp_ready"`
	Version  string `json:"version"`
}

func NewApp() *App {
	exe, _ := os.Executable()
	return &App{
		dataDir:       projectDataDirectory(exe),
		legacyDataDir: filepath.Join(filepath.Dir(exe), "data"),
		requestClose:  func(ctx context.Context) { runtime.EventsEmit(ctx, "app:close-requested") },
		quit:          runtime.Quit,
	}
}
func (a *App) startup(ctx context.Context) { a.ctx = ctx }
func (a *App) beforeClose(ctx context.Context) bool {
	a.mu.Lock()
	preventClose := a.dirty && !a.closeConfirmed
	a.mu.Unlock()
	if !preventClose {
		return false
	}
	// Let the renderer confirm asynchronously; never block the native close callback.
	a.requestClose(ctx)
	return true
}
func (a *App) ConfirmQuit() {
	a.mu.Lock()
	a.closeConfirmed = true
	a.mu.Unlock()
	// Quit runs OnBeforeClose again, which must now allow the window to close.
	a.quit(a.ctx)
}
func (a *App) SetDirty(dirty bool) { a.mu.Lock(); a.dirty = dirty; a.mu.Unlock() }
func (a *App) GetEnvironment() Environment {
	exe, _ := os.Executable()
	mcpPath := filepath.Join(filepath.Dir(exe), "tuxu-mcp.exe")
	_, err := os.Stat(mcpPath)
	return Environment{a.dataDir, mcpPath, err == nil, "0.2.14"}
}
func (a *App) GetSession() Session {
	s := Session{Project: core.Project{SchemaVersion: "1.0", Name: "图书借阅系统", Diagrams: core.Templates()}}
	b, err := os.ReadFile(filepath.Join(a.dataDir, workspace.Internal, "recovery.json"))
	if os.IsNotExist(err) {
		b, err = os.ReadFile(filepath.Join(a.dataDir, "recovery.json"))
	}
	if os.IsNotExist(err) && a.legacyDataDir != "" {
		b, err = os.ReadFile(filepath.Join(a.legacyDataDir, "recovery.json"))
	}
	if os.IsNotExist(err) {
		if legacy := a.legacyRecoveryFile(); legacy != "" {
			b, err = os.ReadFile(legacy)
		}
	}
	if os.IsNotExist(err) {
		return s
	}
	if err != nil {
		s.RecoveryError = err.Error()
		return s
	}
	var saved Session
	if err = json.Unmarshal(b, &saved); err != nil {
		s.RecoveryError = err.Error()
		return s
	}
	p, _ := json.Marshal(saved.Project)
	if _, err = core.ParseProject(p); err != nil {
		s.RecoveryError = err.Error()
		return s
	}
	saved.Recovered = true
	return saved
}
func (a *App) SaveRecovery(raw, path string) error {
	return a.saveRecovery(raw, path, "")
}
func (a *App) SaveWorkspaceRecovery(raw, path, revision string) error {
	return a.saveRecovery(raw, path, revision)
}
func (a *App) saveRecovery(raw, path, revision string) error {
	p, err := core.ParseProject([]byte(raw))
	if err != nil {
		return err
	}
	b, err := json.Marshal(Session{Project: p, Path: path, Revision: revision})
	if err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return core.AtomicWrite(filepath.Join(a.dataDir, workspace.Internal, "recovery.json"), b)
}
func (a *App) RenderDiagram(raw string) core.RenderResult    { return core.Render(raw) }
func (a *App) ExportText(raw, format string) (string, error) { return core.Export(raw, format) }
func (a *App) SaveProject(raw, path string, saveAs bool) (string, error) {
	p, err := core.ParseProject([]byte(raw))
	if err != nil {
		return "", err
	}
	if saveAs || path == "" {
		path, err = runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{Title: "保存EZ软图项目", DefaultFilename: safeName(p.Name) + ".tuxu.json", Filters: []runtime.FileFilter{{DisplayName: "EZ软图项目", Pattern: "*.tuxu.json"}}})
		if err != nil || path == "" {
			return "", err
		}
		if !strings.HasSuffix(strings.ToLower(path), ".json") {
			path += ".tuxu.json"
		}
	}
	p.UpdatedAt = time.Now().Format(time.RFC3339)
	b, _ := json.MarshalIndent(p, "", "  ")
	if err = core.AtomicWrite(path, b); err != nil {
		return "", err
	}
	return path, nil
}
func (a *App) OpenProject() (Session, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{Title: "打开EZ软图项目", Filters: []runtime.FileFilter{{DisplayName: "EZ软图项目 / JSON", Pattern: "*.json"}}})
	if err != nil || path == "" {
		return Session{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return Session{}, err
	}
	if info.Size() > 32<<20 {
		return Session{}, fmt.Errorf("项目文件不能超过 32 MB")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return Session{}, err
	}
	p, err := core.ParseProject(b)
	if err != nil {
		return Session{}, err
	}
	return Session{Project: p, Path: path}, nil
}
func (a *App) ExportFile(raw, format, pngBase64 string) (string, error) {
	d, v := core.ValidateJSON([]byte(raw))
	if !v.Valid {
		return "", fmt.Errorf("%s", v.Issues[0].Message)
	}
	if format != "svg" && format != "drawio" && format != "json" && format != "png" {
		return "", fmt.Errorf("不支持的导出格式")
	}
	var data []byte
	var err error
	if format == "png" {
		if len(pngBase64) > 64<<20 {
			return "", fmt.Errorf("图片超过大小限制")
		}
		data, err = base64.StdEncoding.DecodeString(pngBase64)
		if err == nil && (len(data) < 8 || string(data[:8]) != "\x89PNG\r\n\x1a\n") {
			return "", fmt.Errorf("无效 PNG 数据")
		}
	} else {
		var content string
		content, err = core.Export(raw, format)
		data = []byte(content)
	}
	if err != nil {
		return "", err
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{Title: "导出图表", DefaultFilename: safeName(d.Title) + "." + format, Filters: []runtime.FileFilter{{DisplayName: strings.ToUpper(format), Pattern: "*." + format}}})
	if err != nil || path == "" {
		return "", err
	}
	if !strings.HasSuffix(strings.ToLower(path), "."+format) {
		path += "." + format
	}
	if err = core.AtomicWrite(path, data); err != nil {
		return "", err
	}
	return path, nil
}
func (a *App) CopyText(text string) error { return runtime.ClipboardSetText(a.ctx, text) }
func safeName(s string) string {
	r := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "-", "?", "-", "\"", "-", "<", "-", ">", "-", "|", "-")
	s = strings.Trim(r.Replace(s), " .")
	if s == "" {
		return "diagram"
	}
	return s
}
