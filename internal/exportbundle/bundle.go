package exportbundle

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode"
)

const MaxBytes = 128 << 20

type File struct {
	Path string `json:"path"`
	Data string `json:"data"`
}

type decodedFile struct {
	path string
	data []byte
}

func validPart(part string) bool {
	if part == "" || part == "." || part == ".." || strings.TrimSpace(part) != part || strings.TrimRight(part, ". ") != part || strings.ContainsAny(part, `\/:*?"<>|`) || strings.ContainsFunc(part, unicode.IsControl) {
		return false
	}
	base := strings.ToUpper(strings.SplitN(part, ".", 2)[0])
	return base != "CON" && base != "PRN" && base != "AUX" && base != "NUL" && !(len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '0' && base[3] <= '9')
}

func decode(files []File) ([]decodedFile, error) {
	if len(files) == 0 || len(files) > 5000 {
		return nil, fmt.Errorf("导出需包含 1～5000 个文件")
	}
	result := make([]decodedFile, 0, len(files))
	seen, total := map[string]bool{}, 0
	for _, f := range files {
		if path.Clean(f.Path) != f.Path || strings.HasPrefix(f.Path, "/") {
			return nil, fmt.Errorf("无效导出路径：%s", f.Path)
		}
		for _, part := range strings.Split(f.Path, "/") {
			if !validPart(part) {
				return nil, fmt.Errorf("无效导出路径：%s", f.Path)
			}
		}
		switch strings.ToLower(path.Ext(f.Path)) {
		case ".png", ".svg", ".drawio", ".json":
		default:
			return nil, fmt.Errorf("不支持的导出文件：%s", f.Path)
		}
		key := strings.ToLower(f.Path)
		if seen[key] {
			return nil, fmt.Errorf("导出文件重名：%s", f.Path)
		}
		seen[key] = true
		if len(f.Data) > (MaxBytes-total)*4/3+4 {
			return nil, fmt.Errorf("批量导出内容超过 128 MB，请缩小范围")
		}
		data, err := base64.StdEncoding.DecodeString(f.Data)
		if err != nil {
			return nil, fmt.Errorf("%s：无效文件数据", f.Path)
		}
		total += len(data)
		if total > MaxBytes {
			return nil, fmt.Errorf("批量导出内容超过 128 MB，请缩小范围")
		}
		result = append(result, decodedFile{f.Path, data})
	}
	// A file must never also be the parent directory of another file.
	for _, f := range result {
		for parent := path.Dir(f.path); parent != "."; parent = path.Dir(parent) {
			if seen[strings.ToLower(parent)] {
				return nil, fmt.Errorf("导出路径冲突：%s", f.path)
			}
		}
	}
	return result, nil
}

func zipFiles(files []decodedFile) ([]byte, error) {
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	for _, f := range files {
		entry, err := w.Create(f.path)
		if err != nil {
			return nil, err
		}
		if _, err = entry.Write(f.data); err != nil {
			return nil, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func ZIP(files []File) ([]byte, error) {
	decoded, err := decode(files)
	if err != nil {
		return nil, err
	}
	return zipFiles(decoded)
}

// Write creates new output names, preserving existing exports and user files.
// Output names are reserved exclusively before writing. Do not rename a populated
// directory: on Windows, an open child-directory handle can block that operation.
// Cleanup is restricted to outputs created by this invocation.
func Write(destination, name string, files []File, folder, zipped bool) ([]string, error) {
	if !folder && !zipped {
		return nil, fmt.Errorf("请至少选择一种输出方式")
	}
	if !validPart(name) {
		return nil, fmt.Errorf("无效导出名称")
	}
	decoded, err := decode(files)
	if err != nil {
		return nil, err
	}
	destination, err = filepath.Abs(destination)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(destination)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("导出位置不是可用的文件夹")
	}
	var archive []byte
	if zipped {
		archive, err = zipFiles(decoded)
		if err != nil {
			return nil, err
		}
	}
	for index := 1; index <= 10000; index++ {
		candidate := name
		if index > 1 {
			candidate = fmt.Sprintf("%s (%d)", name, index)
		}
		directory, archivePath := filepath.Join(destination, candidate), filepath.Join(destination, candidate+".zip")
		occupied := false
		for _, p := range []string{directory, archivePath} {
			if _, statErr := os.Lstat(p); statErr == nil {
				occupied = true
			} else if !os.IsNotExist(statErr) {
				return nil, statErr
			}
		}
		if occupied {
			continue
		}
		createdFolder, createdArchive := false, false
		cleanup := func() error {
			var cleanupErr error
			if createdFolder {
				cleanupErr = os.RemoveAll(directory)
			}
			if createdArchive {
				cleanupErr = errors.Join(cleanupErr, os.Remove(archivePath))
			}
			return cleanupErr
		}
		if folder {
			if err = os.Mkdir(directory, 0755); os.IsExist(err) {
				continue
			} else if err != nil {
				return nil, err
			}
			createdFolder = true
		}
		paths := []string{}
		if zipped {
			file, createErr := os.OpenFile(archivePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
			if os.IsExist(createErr) {
				if err = cleanup(); err != nil {
					return nil, err
				}
				continue
			}
			if createErr != nil {
				return nil, errors.Join(createErr, cleanup())
			}
			createdArchive = true
			_, writeErr := file.Write(archive)
			closeErr := file.Close()
			if writeErr != nil || closeErr != nil {
				return nil, errors.Join(fmt.Errorf("写入压缩包失败：%v %v", writeErr, closeErr), cleanup())
			}
			paths = append(paths, archivePath)
		}
		if folder {
			if err = writeFolder(directory, decoded); err != nil {
				return nil, errors.Join(err, cleanup())
			}
			paths = append(paths, directory)
		}
		return paths, nil
	}
	return nil, fmt.Errorf("导出位置存在过多同名结果，请选择其他文件夹")
}

func writeFolder(directory string, files []decodedFile) error {
	for _, f := range files {
		p := filepath.Join(directory, filepath.FromSlash(f.path))
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(p, f.data, 0644); err != nil {
			return err
		}
	}
	return nil
}
