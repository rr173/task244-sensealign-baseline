package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"task244-sensealign/internal/model"
	"task244-sensealign/internal/service"
	"task244-sensealign/internal/store"
)

func TestBug02_BatchTransitionsAreSequential(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "bug2.db")); if err != nil { t.Fatal(err) }; defer st.DB.Close(); h := NewServer(service.New(st)).Routes()
	var b model.Batch; call2442(t, h, http.MethodPost, "/api/batches", map[string]any{"name":"b"}, &b)
	status, _ := call2442(t, h, http.MethodPut, "/api/batches/"+b.ID, map[string]any{"status":"published"}, nil)
	if status != http.StatusConflict { t.Fatalf("illegal jump status=%d, want 409", status) }
	var got model.Batch; call2442(t, h, http.MethodGet, "/api/batches/"+b.ID, nil, &got); if got.Status != model.BatchOrganizing { t.Fatalf("status changed after rejected jump: %s", got.Status) }
	for _, next := range []string{"aligning", "published", "sealed"} { status, _ = call2442(t, h, http.MethodPut, "/api/batches/"+b.ID, map[string]any{"status":next}, nil); if status != http.StatusOK { t.Fatalf("valid transition to %s status=%d", next, status) } }
}

func call2442(t *testing.T, h http.Handler, method, path string, body any, out any) (int, []byte) { t.Helper(); var raw []byte; if body != nil { raw, _ = json.Marshal(body) }; rec:=httptest.NewRecorder(); req:=httptest.NewRequest(method,path,bytes.NewReader(raw)); if body!=nil { req.Header.Set("Content-Type","application/json") }; h.ServeHTTP(rec,req); if out!=nil { if err:=json.Unmarshal(rec.Body.Bytes(),out); err!=nil { t.Fatalf("decode: %v body=%s",err,rec.Body.Bytes()) } }; return rec.Code,rec.Body.Bytes() }
