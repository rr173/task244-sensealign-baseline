package adjudicate

import (
	"path/filepath"
	"testing"

	"task244-sensealign/internal/model"
	"task244-sensealign/internal/store"
)

// newStore 构造临时库用于冲突报告测试。
func newStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "conflict.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return st
}

// seedOneToMany 构造一对多场景：源义项 src 与两个高覆盖目标义项配对。
func seedOneToMany(t *testing.T, st *store.Store) (batchID, src string) {
	t.Helper()
	b, err := st.CreateBatch("b", "d")
	if err != nil {
		t.Fatal(err)
	}
	en, err := st.CreateEntry(b.ID, "en", "bank", "poly")
	if err != nil {
		t.Fatal(err)
	}
	zh, err := st.CreateEntry(b.ID, "zh", "岸/银行", "tgt")
	if err != nil {
		t.Fatal(err)
	}
	srcS, err := st.CreateSense(en.ID, "polysemous source", nil)
	if err != nil {
		t.Fatal(err)
	}
	t1, err := st.CreateSense(zh.ID, "金融机构", nil)
	if err != nil {
		t.Fatal(err)
	}
	t2, err := st.CreateSense(zh.ID, "河岸", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, pair := range []string{t1.ID, t2.ID} {
		a := &model.Alignment{
			ID:            store.NewID("a"),
			SourceSenseID: srcS.ID,
			TargetSenseID: pair,
			Relation:      model.AlignCandidate,
			CovScore:      0.8,
		}
		if err := st.SaveAlignment(a); err != nil {
			t.Fatal(err)
		}
	}
	return b.ID, srcS.ID
}

// TestConflictReportCountsActiveCandidates 校验：否决前一对多冲突计入统计，
// 否决其中一条候选后，该冲突不再出现在报告中（统计只反映当前仍有效的候选）。
func TestConflictReportCountsActiveCandidates(t *testing.T) {
	st := newStore(t)
	defer st.DB.Close()
	batchID, srcID := seedOneToMany(t, st)

	rep, err := BuildConflictReport(st, batchID)
	if err != nil {
		t.Fatal(err)
	}
	sum := rep.Summarize()
	if sum["one_to_many"] != 1 {
		t.Fatalf("before reject: one_to_many = %d, want 1", sum["one_to_many"])
	}
	if len(rep.OneToMany[srcID]) != 2 {
		t.Fatalf("before reject: targets = %v, want 2", rep.OneToMany[srcID])
	}

	// 取消任一一条高覆盖候选：将其裁决为否决。
	tgts := rep.OneToMany[srcID]
	rejected := &model.Alignment{
		ID:            store.NewID("a"),
		SourceSenseID: srcID,
		TargetSenseID: tgts[0],
		Relation:      model.AlignRejected,
		CovScore:      0.8,
	}
	if err := st.SaveAlignment(rejected); err != nil {
		t.Fatal(err)
	}

	rep2, err := BuildConflictReport(st, batchID)
	if err != nil {
		t.Fatal(err)
	}
	sum2 := rep2.Summarize()
	if sum2["one_to_many"] != 0 {
		t.Fatalf("after reject: one_to_many = %d, want 0", sum2["one_to_many"])
	}
	if _, still := rep2.OneToMany[srcID]; still {
		t.Fatalf("after reject: src %s still listed as one-to-many: %v", srcID, rep2.OneToMany[srcID])
	}
}

// TestConflictReportIgnoresRejectedRegisterAndHypernym 校验语域冲突与上下位提示
// 同样只统计当前有效候选，否决后不再计入。
func TestConflictReportIgnoresRejectedRegisterAndHypernym(t *testing.T) {
	st := newStore(t)
	defer st.DB.Close()
	b, err := st.CreateBatch("b", "d")
	if err != nil {
		t.Fatal(err)
	}
	en, err := st.CreateEntry(b.ID, "en", "w", "")
	if err != nil {
		t.Fatal(err)
	}
	zh, err := st.CreateEntry(b.ID, "zh", "词", "")
	if err != nil {
		t.Fatal(err)
	}
	srcS, err := st.CreateSense(en.ID, "src", nil)
	if err != nil {
		t.Fatal(err)
	}
	tgt, err := st.CreateSense(zh.ID, "tgt", nil)
	if err != nil {
		t.Fatal(err)
	}
	a := &model.Alignment{
		ID:               store.NewID("a"),
		SourceSenseID:    srcS.ID,
		TargetSenseID:    tgt.ID,
		Relation:         model.AlignCandidate,
		CovScore:         0.6,
		RegisterConflict: true,
		Hypernym:         true,
	}
	if err := st.SaveAlignment(a); err != nil {
		t.Fatal(err)
	}

	rep, err := BuildConflictReport(st, b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.RegisterConflicts) != 1 || len(rep.HypernymPairs) != 1 {
		t.Fatalf("before reject: register=%d hypernym=%d, want 1/1",
			len(rep.RegisterConflicts), len(rep.HypernymPairs))
	}

	// 否决该对齐后，两类冲突均不再计入。
	a.Relation = model.AlignRejected
	if err := st.SaveAlignment(a); err != nil {
		t.Fatal(err)
	}
	rep2, err := BuildConflictReport(st, b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep2.RegisterConflicts) != 0 || len(rep2.HypernymPairs) != 0 {
		t.Fatalf("after reject: register=%d hypernym=%d, want 0/0",
			len(rep2.RegisterConflicts), len(rep2.HypernymPairs))
	}
}
