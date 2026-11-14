package exportbundle

import (
	"encoding/json"
	"net/http"
)

// ZIPHandler supports the browser development preview; it never writes files.
func ZIPHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()
	var request struct {
		Files []File `json:"files"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 180<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	data, err := ZIP(request.Files)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Write(data)
}
