package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"ezruantu/internal/core"
)

const Internal = ".ezbitu"
const maxFileSize = 32 << 20

type Entry struct {
	Name        string  `json:"name"`
	Path        string  `json:"path"`
	Kind        string  `json:"kind"`
	DiagramType string  `json:"diagram_type,omitempty"`
	Children    []Entry `json:"children,omitempty"`
}
type Tree struct {
	Root    string  `json:"root"`
	Entries []Entry `json:"entries"`
}
type Document struct {
	Path     string       `json:"path"`
	Diagram  core.Diagram `json:"diagram"`
	Revision string       `json:"revision"`
}
type Store struct {
	root string
	mu   sync.Mutex
}

func (s *Store) hiddenRuntime(parent, name string) bool {
	if name == Internal {
		return true
	}
	if parent != s.root {
		return false
	}
	if name == "webview" {
		_, err := os.Stat(filepath.Join(parent, name, "EBWebView"))
		return err == nil
	}
	if name == "recovery.json" {
		data, err := os.ReadFile(filepath.Join(parent, name))
		if err == nil {
			var legacy struct {
				Project json.RawMessage `json:"project"`
			}
			return json.Unmarshal(data, &legacy) == nil && len(legacy.Project) > 0
		}
	}
	return false
}

func New(root string) *Store {
	absolute, _ := filepath.Abs(root)
	return &Store{root: filepath.Clean(absolute)}
}
func revision(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

func validName(name string) error {
	if name == "" || name == "." || name == ".." || strings.TrimSpace(name) != name || strings.TrimRight(name, ". ") != name || len([]rune(name)) > 120 {
		return fmt.Errorf("名称需为 1～120 个字符，不能以空格或点结尾")
	}
	if strings.ContainsAny(name, "/\\:*?\"<>|") || strings.ContainsFunc(name, unicode.IsControl) {
		return fmt.Errorf("名称包含不支持的字符")
	}
	base := strings.ToUpper(strings.SplitN(name, ".", 2)[0])
	if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" || (len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '0' && base[3] <= '9') {
		return fmt.Errorf("该名称是 Windows 保留名称")
	}
	if strings.EqualFold(name, Internal) {
		return fmt.Errorf("该名称由应用保留")
	}
	return nil
}

func (s *Store) resolve(relative string, allowRoot bool) (string, error) {
	if relative == "" {
		if allowRoot {
			return s.root, nil
		}
		return "", fmt.Errorf("不能修改主目录本身")
	}
	local := filepath.FromSlash(relative)
	if !filepath.IsLocal(local) {
		return "", fmt.Errorf("路径必须位于主目录内")
	}
	parts := strings.Split(strings.ReplaceAll(relative, "\\", "/"), "/")
	current := s.root
	for _, part := range parts {
		if err := validName(part); err != nil {
			return "", err
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("不支持通过链接访问目录外的文件")
		}
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}
	}
	return current, nil
}
func (s *Store) rel(path string) string {
	relative, _ := filepath.Rel(s.root, path)
	return filepath.ToSlash(relative)
}
func (s *Store) existing(relative string, directory bool) (string, error) {
	p, err := s.resolve(relative, directory)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(p)
	if err != nil {
		return "", err
	}
	if info.IsDir() != directory {
		return "", fmt.Errorf("文件类型与操作不匹配")
	}
	return p, nil
}
func (s *Store) unused(parent, name string) (string, error) {
	if err := validName(name); err != nil {
		return "", err
	}
	p := filepath.Join(parent, name)
	if _, err := os.Lstat(p); !os.IsNotExist(err) {
		if err != nil {
			return "", err
		}
		return "", fmt.Errorf("同名文件或文件夹已存在")
	}
	return p, nil
}
func fileName(name string) string {
	name = strings.Trim(name, " .")
	r := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "-", "?", "-", "\"", "-", "<", "-", ">", "-", "|", "-")
	name = r.Replace(name)
	if len([]rune(name)) > 90 {
		name = string([]rune(name)[:90])
	}
	if name == "" {
		name = "未命名图表"
	}
	if err := validName(name); err != nil {
		name = "图表-" + name
	}
	return name
}

// Initialize once, without replacing any user file. Runtime data stays hidden.
func (s *Store) Initialize(seed core.Project, recovered bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	internal := filepath.Join(s.root, Internal)
	if err := os.MkdirAll(internal, 0755); err != nil {
		return err
	}
	marker := filepath.Join(internal, "initialized")
	if _, err := os.Stat(marker); err == nil {
		return nil
	}
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return err
	}
	hasFiles := false
	for _, entry := range entries {
		if !s.hiddenRuntime(s.root, entry.Name()) {
			hasFiles = true
			break
		}
	}
	if !hasFiles && len(seed.Diagrams) > 0 {
		folder := "示例图"
		if recovered {
			folder = "恢复的项目"
		}
		if _, err = s.importLocked(seed, folder); err != nil {
			return err
		}
	}
	return core.AtomicWrite(marker, []byte("1\n"))
}

func (s *Store) List() (Tree, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	var walk func(string, int) ([]Entry, error)
	walk = func(parent string, depth int) ([]Entry, error) {
		if depth > 32 {
			return nil, fmt.Errorf("文件夹层级不能超过 32 层")
		}
		items, err := os.ReadDir(parent)
		if err != nil {
			return nil, err
		}
		out := []Entry{}
		for _, item := range items {
			if s.hiddenRuntime(parent, item.Name()) {
				continue
			}
			count++
			if count > 5000 {
				return nil, fmt.Errorf("主目录超过 5000 项，请缩小工作目录")
			}
			p := filepath.Join(parent, item.Name())
			info, err := os.Lstat(p)
			if err != nil {
				return nil, err
			}
			if info.Mode()&os.ModeSymlink != 0 {
				continue
			}
			e := Entry{Name: item.Name(), Path: s.rel(p), Kind: "file"}
			if item.IsDir() {
				e.Kind = "folder"
				e.Children, err = walk(p, depth+1)
				if err != nil {
					return nil, err
				}
			} else if strings.EqualFold(filepath.Ext(item.Name()), ".json") && info.Size() <= maxFileSize {
				if data, err := os.ReadFile(p); err == nil {
					var d core.Diagram
					if json.Unmarshal(data, &d) == nil && d.SchemaVersion == "1.0" {
						switch d.Type {
						case "er", "table", "usecase", "flowchart", "sequence", "class", "architecture", "structure":
							e.DiagramType = d.Type
						}
					}
				}
			}
			out = append(out, e)
		}
		sort.SliceStable(out, func(i, j int) bool {
			if out[i].Kind != out[j].Kind {
				return out[i].Kind == "folder"
			}
			return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
		})
		return out, nil
	}
	entries, err := walk(s.root, 0)
	return Tree{Root: s.root, Entries: entries}, err
}
func (s *Store) Read(relative string) (Document, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readLocked(relative)
}
func (s *Store) readLocked(relative string) (Document, error) {
	p, err := s.existing(relative, false)
	if err != nil {
		return Document{}, err
	}
	info, err := os.Stat(p)
	if err != nil {
		return Document{}, err
	}
	if info.Size() > maxFileSize {
		return Document{}, fmt.Errorf("文件不能超过 32 MB")
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return Document{}, err
	}
	d, v := core.ValidateJSON(data)
	if !v.Valid {
		return Document{}, fmt.Errorf("该文件不是有效图表：%s", v.Issues[0].Message)
	}
	return Document{Path: s.rel(p), Diagram: d, Revision: revision(data)}, nil
}
func (s *Store) CreateFolder(parent, name string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	directory, err := s.existing(parent, true)
	if err != nil {
		return "", err
	}
	p, err := s.unused(directory, name)
	if err != nil {
		return "", err
	}
	if err = os.Mkdir(p, 0755); err != nil {
		return "", err
	}
	return s.rel(p), nil
}
func (s *Store) CreateFile(parent, name, raw string) (Document, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	directory, err := s.existing(parent, true)
	if err != nil {
		return Document{}, err
	}
	if !strings.HasSuffix(strings.ToLower(name), ".json") {
		name += ".json"
	}
	p, err := s.unused(directory, name)
	if err != nil {
		return Document{}, err
	}
	d, v := core.ValidateJSON([]byte(raw))
	if !v.Valid {
		return Document{}, fmt.Errorf("图表无效：%s", v.Issues[0].Message)
	}
	data, _ := json.MarshalIndent(d, "", "  ")
	file, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return Document{}, err
	}
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	if writeErr != nil {
		os.Remove(p)
		return Document{}, writeErr
	}
	if closeErr != nil {
		return Document{}, closeErr
	}
	return Document{Path: s.rel(p), Diagram: d, Revision: revision(data)}, nil
}
func (s *Store) Write(relative, raw, expectedRevision string) (Document, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.existing(relative, false)
	if err != nil {
		return Document{}, err
	}
	if expectedRevision == "" {
		return Document{}, fmt.Errorf("缺少文件版本，请重新打开文件")
	}
	current, err := os.ReadFile(p)
	if err != nil {
		return Document{}, err
	}
	if revision(current) != expectedRevision {
		return Document{}, fmt.Errorf("文件已被其他程序修改，请先另存副本或重新打开，当前编辑内容已保留")
	}
	d, v := core.ValidateJSON([]byte(raw))
	if !v.Valid {
		return Document{}, fmt.Errorf("图表无效：%s", v.Issues[0].Message)
	}
	data, _ := json.MarshalIndent(d, "", "  ")
	if err = core.AtomicWrite(p, data); err != nil {
		return Document{}, err
	}
	return Document{Path: s.rel(p), Diagram: d, Revision: revision(data)}, nil
}
func (s *Store) Rename(relative, name string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	old, err := s.resolve(relative, false)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(old)
	if err != nil {
		return "", err
	}
	if !info.IsDir() && strings.EqualFold(filepath.Ext(old), ".json") && !strings.HasSuffix(strings.ToLower(name), ".json") {
		name += ".json"
	}
	if err = validName(name); err != nil {
		return "", err
	}
	next := filepath.Join(filepath.Dir(old), name)
	if old == next {
		return s.rel(old), nil
	}
	if existing, err := os.Stat(next); err == nil && !os.SameFile(info, existing) {
		return "", fmt.Errorf("同名文件或文件夹已存在")
	} else if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	if err = os.Rename(old, next); err != nil {
		return "", err
	}
	return s.rel(next), nil
}
func (s *Store) Delete(relative string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.resolve(relative, false)
	if err != nil {
		return err
	}
	if _, err = os.Lstat(p); err != nil {
		return err
	}
	trash, err := os.MkdirTemp(filepath.Join(s.root, Internal), "deleted-"+time.Now().Format("20060102-")+"*")
	if err != nil {
		return err
	}
	metadata, _ := json.Marshal(map[string]string{"path": relative, "deleted_at": time.Now().Format(time.RFC3339)})
	if err = core.AtomicWrite(filepath.Join(trash, "original.json"), metadata); err != nil {
		return err
	}
	return os.Rename(p, filepath.Join(trash, filepath.Base(p)))
}
func (s *Store) ImportProject(raw string) (string, error) {
	p, err := core.ParseProject([]byte(raw))
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.importLocked(p, fileName(p.Name))
}
func (s *Store) importLocked(project core.Project, folder string) (string, error) {
	for i := 1; ; i++ {
		name := folder
		if i > 1 {
			name = fmt.Sprintf("%s (%d)", folder, i)
		}
		p, err := s.unused(s.root, name)
		if err == nil {
			folder = filepath.Base(p)
			break
		}
		if !strings.Contains(err.Error(), "已存在") {
			return "", err
		}
	}
	stage, err := os.MkdirTemp(filepath.Join(s.root, Internal), "import-*")
	if err != nil {
		return "", err
	}
	used := map[string]bool{}
	first := ""
	for _, d := range project.Diagrams {
		raw, _ := json.Marshal(d)
		if _, v := core.ValidateJSON(raw); !v.Valid {
			return "", fmt.Errorf("图表无效：%s", v.Issues[0].Message)
		}
		base := fileName(d.Title)
		name := base + ".json"
		for i := 2; used[strings.ToLower(name)]; i++ {
			name = fmt.Sprintf("%s (%d).json", base, i)
		}
		used[strings.ToLower(name)] = true
		data, _ := json.MarshalIndent(d, "", "  ")
		if err = core.AtomicWrite(filepath.Join(stage, name), data); err != nil {
			return "", err
		}
		if first == "" {
			first = folder + "/" + name
		}
	}
	if err = os.Rename(stage, filepath.Join(s.root, folder)); err != nil {
		return "", err
	}
	return first, nil
}
