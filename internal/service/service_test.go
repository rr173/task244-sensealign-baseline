package service

import (
	"os"
	"path/filepath"
	"testing"

	"task244-sensealign/internal/model"
	"task244-sensealign/internal/store"
)

func newTestService(t *testing.T) (*Service, func()) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return New(st), func() { _ = st.DB.Close() }
}

func TestEndToEndAlignAndFreeze(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	b, err := svc.CreateBatch("t", "demo")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.SetBatchStatus(b.ID, model.BatchAligning); err != nil {
		t.Fatal(err)
	}
	en, err := svc.AddEntry(b.ID, "en", "bank", "多义词")
	if err != nil {
		t.Fatal(err)
	}
	zh, err := svc.AddEntry(b.ID, "zh", "银行/岸", "对应")
	if err != nil {
		t.Fatal(err)
	}
	eFin, err := svc.AddSense(en.ID, "financial institution that holds deposits", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddExample(eFin.ID, "I withdrew cash from the bank", "en", "我在银行取了现金", "formal"); err != nil {
		t.Fatal(err)
	}
	eRiv, err := svc.AddSense(en.ID, "side of a river or lake", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddExample(eRiv.ID, "willows grow on the bank of the river", "en", "柳树生长在河岸边", "formal"); err != nil {
		t.Fatal(err)
	}
	zFin, err := svc.AddSense(zh.ID, "金融机构", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddExample(zFin.ID, "我在银行取钱", "zh", "I withdrew cash from the bank", "formal"); err != nil {
		t.Fatal(err)
	}
	zRiv, err := svc.AddSense(zh.ID, "河岸", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddExample(zRiv.ID, "河岸边长着柳树", "zh", "willows grow on the bank of the river", "formal"); err != nil {
		t.Fatal(err)
	}

	cands, err := svc.GenerateCandidates(en.ID, zh.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) != 4 {
		t.Fatalf("expected 4 candidate pairs, got %d", len(cands))
	}

	if _, err := svc.DecideAlignment(eFin.ID, zFin.ID, model.AlignConfirmed, "例句与语域一致", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.DecideAlignment(eRiv.ID, zRiv.ID, model.AlignConfirmed, "river bank 对应河岸", ""); err != nil {
		t.Fatal(err)
	}

	aligns, err := svc.ListAlignments(b.ID)
	if err != nil {
		t.Fatal(err)
	}
	confirmed := 0
	for _, a := range aligns {
		if a.Relation == model.AlignConfirmed {
			confirmed++
		}
	}
	if confirmed != 2 {
		t.Fatalf("expected 2 confirmed alignments, got %d", confirmed)
	}

	v, err := svc.CreateVersion(b.ID, "v1")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.FreezeVersion(v.ID); err != nil {
		t.Fatal(err)
	}
	v2, err := svc.GetVersion(v.ID)
	if err != nil {
		t.Fatal(err)
	}
	if v2.Status != model.VersionFrozen {
		t.Fatalf("expected frozen, got %s", v2.Status)
	}

	// 冻结版本拒绝改写。
	if _, err := svc.DecideAlignment(eFin.ID, zFin.ID, model.AlignRejected, "x", v.ID); err != model.ErrFrozenWrite {
		t.Fatalf("expected ErrFrozenWrite, got %v", err)
	}

	st, err := svc.BatchStats(b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if st.Senses != 4 || st.Examples != 4 || st.Confirmed != 2 {
		t.Fatalf("unexpected stats: %+v", st)
	}
}

// TestGenerateCandidatesRejectsCrossBatch 确保跨批次的词条不能用来生成候选，
// 请求失败且不落盘任何对齐记录（候选只能来自同一批次）。
func TestGenerateCandidatesRejectsCrossBatch(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	b1, _ := svc.CreateBatch("b1", "first batch")
	if err := svc.SetBatchStatus(b1.ID, model.BatchAligning); err != nil {
		t.Fatal(err)
	}
	en1, _ := svc.AddEntry(b1.ID, "en", "bank", "金融机构/河岸")
	s1, _ := svc.AddSense(en1.ID, "financial institution", nil)

	b2, _ := svc.CreateBatch("b2", "second batch")
	if err := svc.SetBatchStatus(b2.ID, model.BatchAligning); err != nil {
		t.Fatal(err)
	}
	zh2, _ := svc.AddEntry(b2.ID, "zh", "银行/岸", "对应")
	s2, _ := svc.AddSense(zh2.ID, "金融机构", nil)

	cands, err := svc.GenerateCandidates(en1.ID, zh2.ID)
	if err != model.ErrCrossBatch {
		t.Fatalf("expected ErrCrossBatch, got %v (cands=%v)", err, cands)
	}
	if cands != nil {
		t.Fatalf("expected no candidates on cross-batch request, got %v", cands)
	}
	// 两个批次都不应留有任何对齐记录。
	if as, _ := svc.ListAlignments(b1.ID); len(as) != 0 {
		t.Fatalf("batch1 should have no alignments, got %d", len(as))
	}
	if as, _ := svc.ListAlignments(b2.ID); len(as) != 0 {
		t.Fatalf("batch2 should have no alignments, got %d", len(as))
	}
	_ = s1
	_ = s2
}

func TestSplitPolysemousSense(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	b, _ := svc.CreateBatch("t", "d")
	en, _ := svc.AddEntry(b.ID, "en", "bank", "m")
	poly, _ := svc.AddSense(en.ID, "shared meaning", nil)
	z1, _ := svc.AddSense(en.ID, "金融机构", nil)
	z2, _ := svc.AddSense(en.ID, "河岸", nil)

	created, err := svc.SplitSense(poly.ID, []string{z1.ID, z2.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 2 {
		t.Fatalf("expected 2 split senses, got %d", len(created))
	}
	orig, err := svc.GetSense(poly.ID)
	if err != nil {
		t.Fatal(err)
	}
	if orig.Status != model.SenseSplit {
		t.Fatalf("expected original split, got %s", orig.Status)
	}
	// 已拆分不可再次拆分。
	if _, err := svc.SplitSense(poly.ID, []string{z1.ID, z2.ID}); err != model.ErrSenseSplit {
		t.Fatalf("expected ErrSenseSplit, got %v", err)
	}
}

func TestRunSelfCheckTemp(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "selfcheck.db")
	if err := RunSelfCheck(dbPath); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("selfcheck db not persisted: %v", err)
	}
}
