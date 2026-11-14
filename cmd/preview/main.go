// Local development API. It is not included or started by the desktop application.
package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"tuxu/internal/core"
	"tuxu/internal/exportbundle"
	"tuxu/internal/workspace"
)

func main() {
	data := flag.String("workspace", "data", "directory for real diagram files")
	isolated := flag.Bool("isolated-workspaces", false, "isolate browser test contexts into separate directories")
	flag.Parse()
	http.Handle("/api/workspace/", workspace.Handler(*data, *isolated))
	http.HandleFunc("/api/export/zip", exportbundle.ZIPHandler)
	http.HandleFunc("/api/project/validate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		defer r.Body.Close()
		b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 32<<20))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if _, err := core.ParseProject(b); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	http.HandleFunc("/api/render", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "POST required", 405)
			return
		}
		defer r.Body.Close()
		b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 2<<20))
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(core.Render(string(b)))
	})
	http.HandleFunc("/api/export", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "POST required", 405)
			return
		}
		defer r.Body.Close()
		b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 2<<20))
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		content, err := core.Export(string(b), r.URL.Query().Get("format"))
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte(content))
	})
	log.Println("Preview API http://127.0.0.1:34116")
	log.Fatal(http.ListenAndServe("127.0.0.1:34116", nil))
}
