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

func TestBug01_SealedBatchRejectsChildWrites(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "bug1.db")); if err != nil { t.Fatal(err) }; defer st.DB.Close()
	srv := NewServer(service.New(st)).Routes()
	var b model.Batch; call2441(t, srv, http.MethodPost, "/api/batches", map[string]any{"name":"b"}, &b)
	call2441(t, srv, http.MethodPut, "/api/batches/"+b.ID, map[string]any{"status":"aligning"}, nil)
	var e model.Entry; call2441(t, srv, http.MethodPost, "/api/batches/"+b.ID+"/entries", map[string]any{"lang_code":"en","headword":"bank"}, &e)
	var sn model.Sense; call2441(t, srv, http.MethodPost, "/api/entries/"+e.ID+"/senses", map[string]any{"definition":"river bank"}, &sn)
	call2441(t, srv, http.MethodPut, "/api/batches/"+b.ID, map[string]any{"status":"published"}, nil)
	call2441(t, srv, http.MethodPut, "/api/batches/"+b.ID, map[string]any{"status":"sealed"}, nil)
	for _, tc := range []struct{ path string; body any }{
		{"/api/entries/"+e.ID+"/senses", map[string]any{"definition":"second"}},
		{"/api/senses/"+sn.ID+"/examples", map[string]any{"text":"on the bank","lang_code":"en"}},
		{"/api/senses/"+sn.ID+"/counterexamples", map[string]any{"text":"not a river"}},
	} {
		status, _ := call2441(t, srv, http.MethodPost, tc.path, tc.body, nil)
		if status != http.StatusConflict { t.Fatalf("%s status=%d, want 409", tc.path, status) }
	}
	var senses []model.Sense; call2441(t, srv, http.MethodGet, "/api/entries/"+e.ID+"/senses", nil, &senses)
	var examples []model.Example; call2441(t, srv, http.MethodGet, "/api/senses/"+sn.ID+"/examples", nil, &examples)
	var counters []model.Counterexample; call2441(t, srv, http.MethodGet, "/api/senses/"+sn.ID+"/counterexamples", nil, &counters)
	if len(senses) != 1 || len(examples) != 0 || len(counters) != 0 { t.Fatalf("sealed writes persisted: senses=%d examples=%d counters=%d", len(senses), len(examples), len(counters)) }
}

func call2441(t *testing.T, h http.Handler, method, path string, body any, out any) (int, []byte) {
	t.Helper(); var raw []byte; if body != nil { raw, _ = json.Marshal(body) }
	rec := httptest.NewRecorder(); req := httptest.NewRequest(method, path, bytes.NewReader(raw)); if body != nil { req.Header.Set("Content-Type", "application/json") }; h.ServeHTTP(rec, req)
	if out != nil { if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil { t.Fatalf("decode %s %s: %v body=%s", method, path, err, rec.Body.Bytes()) } }
	return rec.Code, rec.Body.Bytes()
}
