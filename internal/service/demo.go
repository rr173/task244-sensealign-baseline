package service

import (
	"fmt"

	"task244-sensealign/internal/model"
	"task244-sensealign/internal/store"
)

// RunSelfCheck 端到端自检：在 dbPath 上构建一次真实的跨语言义项对齐闭环，
// 关闭并重新打开数据库验证持久化与重启恢复，全部通过返回 nil。
// 该函数在 main.go 的 --smoke-test 与 HTTP 自检接口中复用。
func RunSelfCheck(dbPath string) error {
	st, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	svc := New(st)

	b, err := svc.CreateBatch("selfcheck", "跨语言义项对齐端到端自检")
	if err != nil {
		return fmt.Errorf("create batch: %w", err)
	}
	if err := svc.SetBatchStatus(b.ID, model.BatchAligning); err != nil {
		return fmt.Errorf("set aligning: %w", err)
	}

	en, err := svc.AddEntry(b.ID, "en", "bank", "英语多义词：金融机构/河岸")
	if err != nil {
		return fmt.Errorf("add en entry: %w", err)
	}
	zh, err := svc.AddEntry(b.ID, "zh", "银行/岸", "汉语对应词目")
	if err != nil {
		return fmt.Errorf("add zh entry: %w", err)
	}

	// 英语两个义项
	eFin, err := svc.AddSense(en.ID, "financial institution that holds deposits", []string{"formal", "technical"})
	if err != nil {
		return fmt.Errorf("en sense1: %w", err)
	}
	if _, err := svc.AddExample(eFin.ID, "I withdrew cash from the bank", "en", "我在银行取了现金", "formal"); err != nil {
		return fmt.Errorf("en ex1: %w", err)
	}
	eRiv, err := svc.AddSense(en.ID, "side of a river or lake", []string{"formal"})
	if err != nil {
		return fmt.Errorf("en sense2: %w", err)
	}
	if _, err := svc.AddExample(eRiv.ID, "willows grow on the bank of the river", "en", "柳树生长在河岸边", "formal"); err != nil {
		return fmt.Errorf("en ex2: %w", err)
	}

	// 汉语两个义项
	zFin, err := svc.AddSense(zh.ID, "金融机构", []string{"formal", "technical"})
	if err != nil {
		return fmt.Errorf("zh sense1: %w", err)
	}
	if _, err := svc.AddExample(zFin.ID, "我在银行取钱", "zh", "I withdrew cash from the bank", "formal"); err != nil {
		return fmt.Errorf("zh ex1: %w", err)
	}
	zRiv, err := svc.AddSense(zh.ID, "河岸", []string{"formal"})
	if err != nil {
		return fmt.Errorf("zh sense2: %w", err)
	}
	if _, err := svc.AddExample(zRiv.ID, "河岸边长着柳树", "zh", "willows grow on the bank of the river", "formal"); err != nil {
		return fmt.Errorf("zh ex2: %w", err)
	}

	if _, err := svc.GenerateCandidates(en.ID, zh.ID); err != nil {
		return fmt.Errorf("generate candidates: %w", err)
	}

	if _, err := svc.DecideAlignment(eFin.ID, zFin.ID, model.AlignConfirmed, "例句与语域一致", ""); err != nil {
		return fmt.Errorf("decide fin: %w", err)
	}
	if _, err := svc.DecideAlignment(eRiv.ID, zRiv.ID, model.AlignConfirmed, "river bank 对应河岸", ""); err != nil {
		return fmt.Errorf("decide riv: %w", err)
	}

	v, err := svc.CreateVersion(b.ID, "v1")
	if err != nil {
		return fmt.Errorf("create version: %w", err)
	}
	if err := svc.FreezeVersion(v.ID); err != nil {
		return fmt.Errorf("freeze version: %w", err)
	}

	// 关闭并重新打开，验证持久化与重启恢复。
	if err := st.DB.Close(); err != nil {
		return fmt.Errorf("close: %w", err)
	}
	st2, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("reopen: %w", err)
	}
	svc2 := New(st2)
	aligns, err := svc2.ListAlignments(b.ID)
	if err != nil {
		return fmt.Errorf("list alignments after reopen: %w", err)
	}
	confirmed := 0
	for _, a := range aligns {
		if a.Relation == model.AlignConfirmed {
			confirmed++
		}
	}
	if confirmed != 2 {
		return fmt.Errorf("expected 2 confirmed alignments after reopen, got %d", confirmed)
	}
	v2, err := svc2.GetVersion(v.ID)
	if err != nil {
		return fmt.Errorf("get version after reopen: %w", err)
	}
	if v2.Status != model.VersionFrozen {
		return fmt.Errorf("version not frozen after reopen: %s", v2.Status)
	}
	if err := st2.DB.Close(); err != nil {
		return fmt.Errorf("close2: %w", err)
	}
	return nil
}
