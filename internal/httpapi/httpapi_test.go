package httpapi

import (
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
