// Package adjudicate 负责跨语言义项对齐的冲突裁决：多义词拆分、反例归并、
// 映射版本冻结与替代。状态机约束（已拆分/已冻结）在此强制。
package adjudicate

import (
	"time"

	"task244-sensealign/internal/model"
	"task244-sensealign/internal/store"
)

// SplitSense 拆分多义词：将原义项标记为已拆分，并为每个目标义项复制出一个独立新义项
// （沿用原定义与语域标签，状态置为可对齐），同时建立新义项到目标的候选对齐。
// targetSenseIDs 至少需 2 个，否则不构成拆分。
func SplitSense(s *store.Store, senseID string, targetSenseIDs []string) ([]*model.Sense, error) {
	if len(targetSenseIDs) < 2 {
		return nil, model.ErrEmptyDefinition
	}
	orig, err := s.GetSense(senseID)
	if err != nil {
		return nil, err
	}
	if orig.Status == model.SenseSplit {
		return nil, model.ErrSenseSplit
	}
	seenTargets := make(map[string]struct{}, len(targetSenseIDs))
	for _, t := range targetSenseIDs {
		if _, exists := seenTargets[t]; exists {
			return nil, model.ErrDuplicateTarget
		}
		seenTargets[t] = struct{}{}
		if t == senseID {
			return nil, model.ErrSelfAlign
		}
		if _, err := s.GetSense(t); err != nil {
			return nil, err
		}
	}
	var created []*model.Sense
	for _, t := range targetSenseIDs {
		ns, err := s.CreateSense(orig.EntryID, orig.Definition, orig.RegisterTags)
		if err != nil {
			return nil, err
		}
		if err := s.SetSenseStatus(ns.ID, model.SenseAlignable); err != nil {
			return nil, err
		}
		// 建立新义项到目标义项的候选对齐，使拆分结果可直接进入裁决。
		a := &model.Alignment{
			ID:            store.NewID("a"),
			SourceSenseID: ns.ID,
			TargetSenseID: t,
			Relation:      model.AlignCandidate,
			CovScore:      0,
			CreatedAt:     time.Now().UTC(),
		}
		if err := s.SaveAlignment(a); err != nil {
			return nil, err
		}
		created = append(created, ns)
	}
	if err := s.SetSenseStatus(orig.ID, model.SenseSplit); err != nil {
		return nil, err
	}
	return created, nil
}

// FreezeVersion 冻结映射版本：写入不可变快照。
func FreezeVersion(s *store.Store, versionID, snapshot string) error {
	v, err := s.GetVersion(versionID)
	if err != nil {
		return err
	}
	if v.Status == model.VersionFrozen {
		return model.ErrVersionFrozen
	}
	return s.FreezeVersion(versionID, snapshot)
}
