package store

import (
	"path/filepath"
	"testing"

	"task244-sensealign/internal/model"
)

func TestBatchAndEntrySurviveReopen(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "roundtrip.db")
	st, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	b, err := st.CreateBatch("roundtrip", "persisted")
	if err != nil {
		t.Fatal(err)
	}
	e, err := st.CreateEntry(b.ID, "en", "bank", "gloss")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.DB.Close(); err != nil {
		t.Fatal(err)
	}

	st2, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st2.DB.Close()
	gotBatch, err := st2.GetBatch(b.ID)
	if err != nil {
		t.Fatal(err)
	}
	gotEntry, err := st2.GetEntry(e.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotBatch.Name != b.Name || gotEntry.Headword != e.Headword {
		t.Fatalf("roundtrip mismatch: batch=%+v entry=%+v", gotBatch, gotEntry)
	}
}

func TestDuplicateEntryIsDomainError(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "duplicate.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.DB.Close()
	b, err := st.CreateBatch("duplicate", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateEntry(b.ID, "en", "bank", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateEntry(b.ID, "en", "bank", ""); err != model.ErrDuplicate {
		t.Fatalf("duplicate error = %v, want %v", err, model.ErrDuplicate)
	}
}
