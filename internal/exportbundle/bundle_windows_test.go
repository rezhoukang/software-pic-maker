package exportbundle

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

func TestWriteFolderWithOpenChildDirectoryHandle(t *testing.T) {
	directory := t.TempDir()
	child := filepath.Join(directory, "子目录")
	if err := os.Mkdir(child, 0755); err != nil {
		t.Fatal(err)
	}
	// Model the directory handles held by Windows file monitors. Even with
	// FILE_SHARE_DELETE, this can prevent renaming the populated parent tree.
	handle, err := windows.CreateFile(windows.StringToUTF16Ptr(child), 0,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(handle)
	files, err := decode(fixture())
	if err != nil {
		t.Fatal(err)
	}
	if err = writeFolder(directory, files); err != nil {
		t.Fatal("export failed while a child-directory handle remained open", err)
	}
	data, err := os.ReadFile(filepath.Join(child, "用户.json"))
	if err != nil || string(data) != `{"title":"另一个用户"}` {
		t.Fatal("nested export content was lost", err)
	}
}
