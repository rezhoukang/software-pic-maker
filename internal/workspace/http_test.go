package workspace

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPreviewFilesystemIsolationAndOriginChecks(t *testing.T) {
	handler := Handler(t.TempDir(), true)
	request := func(operation, body, key, origin, contentType string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, "/api/workspace/"+operation, strings.NewReader(body))
		r.Header.Set("X-EZ-Workspace", key)
		r.Header.Set("Origin", origin)
		r.Header.Set("Content-Type", contentType)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	a, b := "11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222"
	if w := request("list", "{}", a, "https://unrelated.example", "application/json"); w.Code != 403 {
		t.Fatal("foreign origin permitted", w.Code)
	}
	if w := request("list", "{}", a, "", "text/plain"); w.Code != 415 {
		t.Fatal("simple cross-site request permitted", w.Code)
	}
	if w := request("list", "{}", "../escape", "", "application/json"); w.Code != 400 {
		t.Fatal("unsafe test scope accepted")
	}
	if w := request("create-folder", `{"name":"仅在A中","parent":""}`, a, "http://127.0.0.1:5173", "application/json"); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	w := request("list", "{}", b, "", "application/json")
	var tree Tree
	if e := json.Unmarshal(w.Body.Bytes(), &tree); e != nil {
		t.Fatal(e)
	}
	for _, entry := range tree.Entries {
		if entry.Name == "仅在A中" {
			t.Fatal("test workspaces leaked into each other")
		}
	}
}
