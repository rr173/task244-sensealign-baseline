package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"task244-sensealign/internal/service"
	"task244-sensealign/internal/store"
)

func TestHealthAndRootRoutes(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "http.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.DB.Close()
	srv := NewServer(service.New(st)).Routes()

	health := httptest.NewRecorder()
	srv.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("health status = %d, want 200", health.Code)
	}
	root := httptest.NewRecorder()
	srv.ServeHTTP(root, httptest.NewRequest(http.MethodGet, "/", nil))
	if root.Code != http.StatusOK || root.Body.Len() == 0 {
		t.Fatalf("root response = status %d, bytes %d", root.Code, root.Body.Len())
	}
}

func TestUnknownAPIPathReturnsJSONNotFound(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "http-404.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.DB.Close()
	srv := NewServer(service.New(st)).Routes()
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/batches/missing/stats", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

// TestConfirmBlockedByCounterexampleReturns409 验证：登记反例后，确认对齐的
// 请求被拒并映射为 409 Conflict，带反例的关系不会被直接确认发布。
func TestConfirmBlockedByCounterexampleReturns409(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "cx.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.DB.Close()
	srv := NewServer(service.New(st)).Routes()

	mux := srv
	post := func(path string, body any) *httptest.ResponseRecorder {
		b, _ := json.Marshal(body)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b)))
		return rec
	}
	put := func(path string, body any) *httptest.ResponseRecorder {
		b, _ := json.Marshal(body)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, path, bytes.NewReader(b)))
		return rec
	}

	b := post("/api/batches", map[string]string{"name": "t"}).Body
	var batch struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(b.Bytes(), &batch); err != nil {
		t.Fatal(err)
	}
	if put("/api/batches/"+batch.ID, map[string]string{"status": "aligning"}).Code != http.StatusOK {
		t.Fatal("set aligning failed")
	}
	en := post("/api/batches/"+batch.ID+"/entries", map[string]string{"lang_code": "en", "headword": "bank"}).Body
	var entryA struct {
		ID string `json:"id"`
	}
	json.Unmarshal(en.Bytes(), &entryA)
	zh := post("/api/batches/"+batch.ID+"/entries", map[string]string{"lang_code": "zh", "headword": "银行"}).Body
	var entryB struct {
		ID string `json:"id"`
	}
	json.Unmarshal(zh.Bytes(), &entryB)
	es := post("/api/entries/"+entryA.ID+"/senses", map[string]string{"definition": "financial institution"}).Body
	var senseA struct {
		ID string `json:"id"`
	}
	json.Unmarshal(es.Bytes(), &senseA)
	zs := post("/api/entries/"+entryB.ID+"/senses", map[string]string{"definition": "金融机构"}).Body
	var senseB struct {
		ID string `json:"id"`
	}
	json.Unmarshal(zs.Bytes(), &senseB)
	post("/api/align/candidates", map[string]string{"source_entry_id": entryA.ID, "target_entry_id": entryB.ID})
	// 登记反例后尝试确认。
	if post("/api/senses/"+senseA.ID+"/counterexamples", map[string]string{"text": "反例"}).Code != http.StatusCreated {
		t.Fatal("add counterexample failed")
	}
	dec := post("/api/alignments/decide", map[string]string{
		"source_sense_id": senseA.ID, "target_sense_id": senseB.ID, "relation": "confirmed",
	})
	if dec.Code != http.StatusConflict {
		t.Fatalf("confirm status = %d, want 409 Conflict", dec.Code)
	}
}
