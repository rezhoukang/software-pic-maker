package exportbundle

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func fixture() []File {
	return []File{{"用户.json", base64.StdEncoding.EncodeToString([]byte(`{"title":"用户"}`))}, {"子目录/用户.json", base64.StdEncoding.EncodeToString([]byte(`{"title":"另一个用户"}`))}}
}

func TestFolderAndZipPreserveHierarchyWithoutOverwriting(t *testing.T) {
	destination := t.TempDir()
	paths, err := Write(destination, "项目", fixture(), true, true)
	if err != nil || len(paths) != 2 {
		t.Fatal(paths, err)
	}
	data, err := os.ReadFile(filepath.Join(destination, "项目", "子目录", "用户.json"))
	if err != nil || !strings.Contains(string(data), "另一个用户") {
		t.Fatal("nested design was lost", err)
	}
	archive, err := os.ReadFile(filepath.Join(destination, "项目.zip"))
	if err != nil {
		t.Fatal(err)
	}
	r, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil || len(r.File) != 2 {
		t.Fatal("invalid ZIP", err)
	}
	for i, file := range r.File {
		if file.Name != fixture()[i].Path {
			t.Fatal("ZIP path changed", file.Name)
		}
		reader, _ := file.Open()
		b, readErr := io.ReadAll(reader)
		reader.Close()
		original, _ := base64.StdEncoding.DecodeString(fixture()[i].Data)
		if readErr != nil || !bytes.Equal(b, original) {
			t.Fatal("ZIP payload changed", file.Name, readErr)
		}
	}
	if _, err = Write(destination, "项目", fixture(), true, true); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(filepath.Join(destination, "项目 (2).zip")); err != nil {
		t.Fatal("existing result was not preserved", err)
	}
}

func TestUnsafeAndConflictingPathsWriteNothing(t *testing.T) {
	for _, name := range []string{"../escape.json", "/absolute.json", `C:\escape.json`, "a/../b.json", "CON.json", "folder//x.json", "folder/ x.json"} {
		dir := t.TempDir()
		if _, err := Write(dir, "项目", []File{{name, "e30="}}, true, true); err == nil {
			t.Fatal("unsafe path accepted", name)
		}
		entries, _ := os.ReadDir(dir)
		if len(entries) != 0 {
			t.Fatal("invalid export left partial output", name)
		}
	}
	for _, files := range [][]File{
		{{"A.json", "e30="}, {"a.json", "e30="}},
		{{"a.json", "e30="}, {"a.json/child.json", "e30="}},
		{{"a.json", "invalid base64"}},
	} {
		if _, err := ZIP(files); err == nil {
			t.Fatal("invalid batch accepted", files)
		}
	}
}

func TestOutputModesAndZIPHandler(t *testing.T) {
	for _, modes := range [][2]bool{{true, false}, {false, true}} {
		paths, err := Write(t.TempDir(), "导出", fixture(), modes[0], modes[1])
		if err != nil || len(paths) != 1 {
			t.Fatal("output mode failed", modes, paths, err)
		}
	}
	if _, err := Write(t.TempDir(), "导出", fixture(), false, false); err == nil {
		t.Fatal("empty output modes accepted")
	}
	recorder := httptest.NewRecorder()
	ZIPHandler(recorder, httptest.NewRequest(http.MethodPost, "/api/export/zip", strings.NewReader(`{"files":[{"path":"设计.json","data":"e30="}]}`)))
	if recorder.Code != 200 || recorder.Header().Get("Content-Type") != "application/zip" {
		t.Fatal("preview ZIP endpoint failed", recorder.Code)
	}
}

func TestFailedFolderWriteRollsBackOnlyNewOutputs(t *testing.T) {
	destination := t.TempDir()
	existing := filepath.Join(destination, "项目")
	if err := os.Mkdir(existing, 0755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(existing, "保留.json")
	if err := os.WriteFile(marker, []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	// A component longer than the filesystem limit fails during disk writing,
	// after output reservation, rather than during model/path validation.
	files := append(fixture(), File{strings.Repeat("x", 300) + ".json", "e30="})
	if _, err := Write(destination, "项目", files, true, true); err == nil {
		t.Fatal("expected a filesystem write failure")
	}
	entries, err := os.ReadDir(destination)
	if err != nil || len(entries) != 1 || entries[0].Name() != "项目" {
		t.Fatal("failed export left partial outputs", entries, err)
	}
	data, err := os.ReadFile(marker)
	if err != nil || string(data) != "keep" {
		t.Fatal("existing export was changed", err)
	}
}

func TestConcurrentExportsReserveSeparateOutputs(t *testing.T) {
	destination := t.TempDir()
	var workers sync.WaitGroup
	for range 8 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			if _, err := Write(destination, "项目", fixture(), true, true); err != nil {
				t.Error(err)
			}
		}()
	}
	workers.Wait()
	entries, err := os.ReadDir(destination)
	if err != nil || len(entries) != 16 {
		t.Fatal("concurrent exports did not retain eight folder/ZIP pairs", len(entries), err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			data, err := os.ReadFile(filepath.Join(destination, entry.Name(), "子目录", "用户.json"))
			if err != nil || !strings.Contains(string(data), "另一个用户") {
				t.Error("incomplete concurrent export", entry.Name(), err)
			}
			if _, err := os.Stat(filepath.Join(destination, entry.Name()+".zip")); err != nil {
				t.Error("missing paired ZIP", entry.Name(), err)
			}
		}
	}
}
