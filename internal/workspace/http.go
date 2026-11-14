package workspace

import (
	"encoding/json"
	"ezruantu/internal/core"
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// Browser preview uses the exact same filesystem store as the native bridge.
func Handler(root string, isolated bool) http.Handler {
	stores := map[string]*Store{}
	var mu sync.Mutex
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", 405)
			return
		}
		media, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if media != "application/json" {
			http.Error(w, "application/json required", 415)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" && origin != "http://127.0.0.1:5173" && origin != "http://localhost:5173" && origin != "http://127.0.0.1:34116" {
			http.Error(w, "origin not allowed", 403)
			return
		}
		var args struct {
			Path     string `json:"path"`
			Parent   string `json:"parent"`
			Name     string `json:"name"`
			Raw      string `json:"raw"`
			Revision string `json:"revision"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, 64<<20)
		defer r.Body.Close()
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&args); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		key := ""
		if isolated {
			key = r.Header.Get("X-EZ-Workspace")
			if !regexp.MustCompile(`^[a-f0-9-]{36}$`).MatchString(key) {
				http.Error(w, "isolated workspace key required", 400)
				return
			}
		}
		mu.Lock()
		s := stores[key]
		var err error
		if s == nil {
			directory := root
			if isolated {
				directory = filepath.Join(root, key)
			}
			s = New(directory)
			err = s.Initialize(core.Project{SchemaVersion: "1.0", Name: "示例图", Diagrams: core.Templates()}, false)
			if err == nil {
				stores[key] = s
			}
		}
		mu.Unlock()
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		var result any
		switch strings.TrimPrefix(r.URL.Path, "/api/workspace/") {
		case "list":
			result, err = s.List()
		case "read":
			result, err = s.Read(args.Path)
		case "create-folder":
			result, err = s.CreateFolder(args.Parent, args.Name)
		case "create-file":
			result, err = s.CreateFile(args.Parent, args.Name, args.Raw)
		case "write":
			result, err = s.Write(args.Path, args.Raw, args.Revision)
		case "rename":
			result, err = s.Rename(args.Path, args.Name)
		case "delete":
			err = s.Delete(args.Path)
			result = true
		case "import":
			result, err = s.ImportProject(args.Raw)
		default:
			err = fmt.Errorf("unknown workspace operation")
		}
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(result)
	})
}
