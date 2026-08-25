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

// TestBatchStatusTransitionOrder 状态机只允许按业务顺序前进；
// 非法推进被拒绝且原状态不变（整理中直接跳到已发布或封存）。
func TestBatchStatusTransitionOrder(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	b, err := svc.CreateBatch("t", "d")
	if err != nil {
		t.Fatal(err)
	}

	// 整理中直接跳到已发布 / 封存：非法，状态保持整理中。
	if err := svc.SetBatchStatus(b.ID, model.BatchPublished); err != model.ErrInvalidBatchTransition {
		t.Fatalf("organizing→published: want ErrInvalidBatchTransition, got %v", err)
	}
	if err := svc.SetBatchStatus(b.ID, model.BatchSealed); err != model.ErrInvalidBatchTransition {
		t.Fatalf("organizing→sealed: want ErrInvalidBatchTransition, got %v", err)
	}
	got, err := svc.GetBatch(b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.BatchOrganizing {
		t.Fatalf("status mutated after illegal advance: %s", got.Status)
	}

	// 正序推进到已发布后再跳回待对齐：倒退非法。
	if err := svc.SetBatchStatus(b.ID, model.BatchAligning); err != nil {
		t.Fatalf("organizing→aligning: %v", err)
	}
	if err := svc.SetBatchStatus(b.ID, model.BatchPublished); err != nil {
		t.Fatalf("aligning→published: %v", err)
	}
	if err := svc.SetBatchStatus(b.ID, model.BatchAligning); err != model.ErrInvalidBatchTransition {
		t.Fatalf("published→aligning: want ErrInvalidBatchTransition, got %v", err)
	}
	got, err = svc.GetBatch(b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.BatchPublished {
		t.Fatalf("status mutated after illegal regress: %s", got.Status)
	}

	// 封存为终态：后续任何推进被拒绝。
	if err := svc.SetBatchStatus(b.ID, model.BatchSealed); err != nil {
		t.Fatalf("published→sealed: %v", err)
	}
	if err := svc.SetBatchStatus(b.ID, model.BatchOrganizing); err != model.ErrBatchSealed {
		t.Fatalf("sealed→organizing: want ErrBatchSealed, got %v", err)
	}
	got, err = svc.GetBatch(b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.BatchSealed {
		t.Fatalf("status mutated after sealing: %s", got.Status)
	}

	// 非法取值被拒绝。
	if err := svc.SetBatchStatus(b.ID, model.BatchStatus("nope")); err == nil {
		t.Fatalf("invalid status value accepted")
	}
}

func TestBatchStatusIdempotent(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	b, _ := svc.CreateBatch("t", "d")
	// 重复当前状态视为幂等成功。
	if err := svc.SetBatchStatus(b.ID, model.BatchOrganizing); err != nil {
		t.Fatalf("repeating organizing: %v", err)
	}
	if err := svc.SetBatchStatus(b.ID, model.BatchAligning); err != nil {
		t.Fatalf("organizing→aligning: %v", err)
	}
	if err := svc.SetBatchStatus(b.ID, model.BatchAligning); err != nil {
		t.Fatalf("repeating aligning: %v", err)
	}
}
