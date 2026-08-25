package service

import (
	"errors"
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

// TestDecideAlignmentRejectsCrossBatchVersion 验证：把一个批次的义项裁决
// 绑定到另一个批次的映射版本时被范围校验拦截（返回 ErrCrossBatch），
// 且失败时两个批次的状态都不改变，也不产生串批的对齐写入。
func TestDecideAlignmentRejectsCrossBatchVersion(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	// 批次 A：承载被裁决的义项。
	bA, err := svc.CreateBatch("A", "batch A")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.SetBatchStatus(bA.ID, model.BatchAligning); err != nil {
		t.Fatal(err)
	}
	enA, err := svc.AddEntry(bA.ID, "en", "bank", "m")
	if err != nil {
		t.Fatal(err)
	}
	zhA, err := svc.AddEntry(bA.ID, "zh", "银行", "对应")
	if err != nil {
		t.Fatal(err)
	}
	sA1, err := svc.AddSense(enA.ID, "financial institution", nil)
	if err != nil {
		t.Fatal(err)
	}
	sA2, err := svc.AddSense(zhA.ID, "金融机构", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GenerateCandidates(enA.ID, zhA.ID); err != nil {
		t.Fatal(err)
	}

	// 批次 B：拥有一个草稿映射版本。
	bB, err := svc.CreateBatch("B", "batch B")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.SetBatchStatus(bB.ID, model.BatchAligning); err != nil {
		t.Fatal(err)
	}
	vB, err := svc.CreateVersion(bB.ID, "vB")
	if err != nil {
		t.Fatal(err)
	}

	// 试图把批次 A 的对齐裁决绑定到批次 B 的版本——必须被拒绝。
	_, err = svc.DecideAlignment(sA1.ID, sA2.ID, model.AlignConfirmed, "cross", vB.ID)
	if !errors.Is(err, model.ErrCrossBatch) {
		t.Fatalf("expected ErrCrossBatch, got %v", err)
	}

	// 失败不应改动两个批次的状态。
	bA2, err := svc.GetBatch(bA.ID)
	if err != nil {
		t.Fatal(err)
	}
	if bA2.Status != model.BatchAligning {
		t.Fatalf("batch A status changed to %s", bA2.Status)
	}
	bB2, err := svc.GetBatch(bB.ID)
	if err != nil {
		t.Fatal(err)
	}
	if bB2.Status != model.BatchAligning {
		t.Fatalf("batch B status changed to %s", bB2.Status)
	}

	// 失败不应把版本写入任何对齐行——避免串批。
	alignsA, err := svc.ListAlignments(bA.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range alignsA {
		if a.VersionID == vB.ID {
			t.Fatalf("cross-batch version id leaked into batch A alignment: %+v", a)
		}
	}
	alignsB, err := svc.ListAlignments(bB.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(alignsB) != 0 {
		t.Fatalf("batch B unexpectedly gained alignments: %+v", alignsB)
	}
	// 版本仍为草稿，未被改写。
	vB2, err := svc.GetVersion(vB.ID)
	if err != nil {
		t.Fatal(err)
	}
	if vB2.Status != model.VersionDraft {
		t.Fatalf("version B status changed to %s", vB2.Status)
	}
}
