package workspace

import (
	"encoding/json"
	"ezruantu/internal/core"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	s := New(t.TempDir())
	if err := s.Initialize(core.Project{}, false); err != nil {
		t.Fatal(err)
	}
	return s
}
func diagramJSON(t *testing.T) string {
	t.Helper()
	b, e := json.Marshal(core.Templates()[0])
	if e != nil {
		t.Fatal(e)
	}
	return string(b)
}

func TestDiskFolderFileRenameAndRecoverableDelete(t *testing.T) {
	s := testStore(t)
	folder, e := s.CreateFolder("", "数据库")
	if e != nil {
		t.Fatal(e)
	}
	nested, e := s.CreateFolder(folder, "用户模块")
	if e != nil {
		t.Fatal(e)
	}
	doc, e := s.CreateFile(nested, "用户表", diagramJSON(t))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = os.Stat(filepath.Join(s.root, filepath.FromSlash(doc.Path))); e != nil {
		t.Fatal("file was not persisted", e)
	}
	newFolder, e := s.Rename(folder, "数据设计")
	if e != nil {
		t.Fatal(e)
	}
	path := strings.Replace(doc.Path, folder, newFolder, 1)
	newPath, e := s.Rename(path, "读者表")
	if e != nil {
		t.Fatal(e)
	}
	loaded, e := s.Read(newPath)
	if e != nil || loaded.Revision != doc.Revision {
		t.Fatal("rename changed document", e)
	}
	if e = s.Delete(newFolder); e != nil {
		t.Fatal(e)
	}
	if _, e = os.Stat(filepath.Join(s.root, newFolder)); !os.IsNotExist(e) {
		t.Fatal("deleted folder remains visible")
	}
	backups, _ := filepath.Glob(filepath.Join(s.root, Internal, "deleted-*", newFolder, "用户模块", "读者表.json"))
	if len(backups) != 1 {
		t.Fatal("deleted content is not recoverable", backups)
	}
	tree, e := s.List()
	if e != nil || len(tree.Entries) != 0 {
		t.Fatal("internal data leaked into tree", tree, e)
	}
}
func TestWriteRejectsInvalidContentAndExternalChanges(t *testing.T) {
	s := testStore(t)
	doc, e := s.CreateFile("", "图", diagramJSON(t))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.Write(doc.Path, `{}`, doc.Revision); e == nil {
		t.Fatal("invalid document accepted")
	}
	current, _ := s.Read(doc.Path)
	if current.Revision != doc.Revision {
		t.Fatal("invalid write damaged file")
	}
	d := doc.Diagram
	d.Title = "外部修改"
	external, _ := json.Marshal(d)
	if e = os.WriteFile(filepath.Join(s.root, doc.Path), external, 0644); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Write(doc.Path, diagramJSON(t), doc.Revision); e == nil || !strings.Contains(e.Error(), "其他程序修改") {
		t.Fatal("external change was overwritten", e)
	}
	current, _ = s.Read(doc.Path)
	if current.Diagram.Title != "外部修改" {
		t.Fatal("external content lost")
	}
}
func TestConcurrentSavesOnlyAcceptOneRevision(t *testing.T) {
	s := testStore(t)
	doc, e := s.CreateFile("", "图", diagramJSON(t))
	if e != nil {
		t.Fatal(e)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			d := doc.Diagram
			d.Title = []string{"保存A", "保存B"}[i]
			raw, _ := json.Marshal(d)
			_, err := s.Write(doc.Path, string(raw), doc.Revision)
			results <- err
		}(i)
	}
	wg.Wait()
	close(results)
	success := 0
	for e := range results {
		if e == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatal("stale revisions were accepted", success)
	}
}
func TestPathsNamesAndLinksCannotEscapeRoot(t *testing.T) {
	s := testStore(t)
	for _, path := range []string{"../outside", "a/../../outside", `C:\outside`, `.ezbitu/secret`, `a\..\outside`} {
		if _, e := s.CreateFolder(path, "folder"); e == nil {
			t.Fatal("escaped", path)
		}
		if e := s.Delete(path); e == nil {
			t.Fatal("unsafe delete accepted", path)
		}
	}
	for _, name := range []string{"../bad", "CON", "a/b", "trailing.", " space", "a:b", ".ezbitu"} {
		if _, e := s.CreateFolder("", name); e == nil {
			t.Fatal("unsafe name accepted", name)
		}
	}
	if e := s.Delete(""); e == nil {
		t.Fatal("root deletion accepted")
	}
	if _, e := s.Rename("", "renamed"); e == nil {
		t.Fatal("root rename accepted")
	}
	outside := t.TempDir()
	link := filepath.Join(s.root, "outside-link")
	if e := os.Symlink(outside, link); e == nil {
		if _, e = s.CreateFolder("outside-link", "must-not-exist"); e == nil {
			t.Fatal("followed outside symlink")
		}
		if _, e = os.Stat(filepath.Join(outside, "must-not-exist")); !os.IsNotExist(e) {
			t.Fatal("modified outside root")
		}
	}
}
func TestInitializeAndImportDoNotOverwriteUserFiles(t *testing.T) {
	s := New(t.TempDir())
	project := core.Project{SchemaVersion: "1.0", Name: "设计项目", Diagrams: core.Templates()}
	if e := s.Initialize(project, false); e != nil {
		t.Fatal(e)
	}
	first, e := s.List()
	if e != nil || len(first.Entries) != 1 || len(first.Entries[0].Children) != 8 {
		t.Fatal("missing initial files", first, e)
	}
	if e = s.Initialize(project, false); e != nil {
		t.Fatal(e)
	}
	data, _ := json.Marshal(project)
	path, e := s.ImportProject(string(data))
	if e != nil {
		t.Fatal(e)
	}
	path2, e := s.ImportProject(string(data))
	if e != nil || path == path2 {
		t.Fatal("import overwrote existing folder", path, path2, e)
	}
	project.Diagrams[0].Model.Relations[0].Target = "missing"
	data, _ = json.Marshal(project)
	before, _ := s.List()
	if _, e = s.ImportProject(string(data)); e == nil {
		t.Fatal("invalid project imported")
	}
	after, _ := s.List()
	if len(before.Entries) != len(after.Entries) {
		t.Fatal("invalid import left a visible partial folder")
	}
}
