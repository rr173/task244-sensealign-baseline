package service

import (
	"time"

	"task244-sensealign/internal/adjudicate"
	"task244-sensealign/internal/matcher"
	"task244-sensealign/internal/model"
	"task244-sensealign/internal/store"
)

// GenerateCandidates 为两个词条生成候选对齐，并落盘为 candidate 关系以便后续裁决。
func (svc *Service) GenerateCandidates(entryA, entryB string) ([]model.Candidate, error) {
	if entryA == entryB {
		return nil, model.ErrSameEntry
	}
	srcEntry, err := svc.Store.GetEntry(entryA)
	if err != nil {
		return nil, model.ErrUnknownEntry
	}
	tgtEntry, err := svc.Store.GetEntry(entryB)
	if err != nil {
		return nil, model.ErrUnknownEntry
	}
	// 候选只能来自同一批次：跨批次配对不产出任何候选与对齐记录。
	if srcEntry.BatchID != tgtEntry.BatchID {
		return nil, model.ErrCrossBatch
	}
	cands, err := matcher.GenerateCandidates(svc.Store, entryA, entryB)
	if err != nil {
		return nil, err
	}
	for _, c := range cands {
		a := &model.Alignment{
			ID:               store.NewID("a"),
			SourceSenseID:    c.SourceSenseID,
			TargetSenseID:    c.TargetSenseID,
			Relation:         model.AlignCandidate,
			CovScore:         c.CovScore,
			Hypernym:         c.Hypernym,
			Hyponym:          c.Hyponym,
			RegisterConflict: c.RegisterConflict,
			CreatedAt:        time.Now().UTC(),
		}
		if err := svc.Store.SaveAlignment(a); err != nil {
			return nil, err
		}
	}
	conflicts := matcher.DetectOneToMany(cands, 0.5)
	seen := make(map[string]struct{})
	for _, c := range cands {
		seen[c.SourceSenseID] = struct{}{}
		seen[c.TargetSenseID] = struct{}{}
	}
	for senseID := range seen {
		sn, err := svc.Store.GetSense(senseID)
		if err != nil {
			return nil, err
		}
		if sn.Status == model.SenseSplit {
			continue
		}
		status := model.SenseAlignable
		if _, ok := conflicts[senseID]; ok {
			status = model.SenseConflict
		}
		if err := svc.Store.SetSenseStatus(senseID, status); err != nil {
			return nil, err
		}
	}
	return cands, nil
}

// DecideAlignment 对一条义项配对做出裁决（确认/部分重合/否决）。
// 若纳入某映射版本且该版本已冻结，则拒绝写入（ErrFrozenWrite）。
func (svc *Service) DecideAlignment(sourceID, targetID string, relation model.AlignRelation, reason, versionID string) (*model.Alignment, error) {
	if sourceID == targetID {
		return nil, model.ErrSelfAlign
	}
	if !model.ValidAlignRelation(string(relation)) {
		return nil, model.ErrBadRelation
	}
	source, err := svc.Store.GetSense(sourceID)
	if err != nil {
		return nil, model.ErrUnknownSense
	}
	target, err := svc.Store.GetSense(targetID)
	if err != nil {
		return nil, model.ErrUnknownSense
	}
	sourceEntry, err := svc.Store.GetEntry(source.EntryID)
	if err != nil {
		return nil, err
	}
	targetEntry, err := svc.Store.GetEntry(target.EntryID)
	if err != nil {
		return nil, err
	}
	if sourceEntry.BatchID != targetEntry.BatchID {
		return nil, model.ErrCrossBatch
	}
	if versionID != "" {
		v, err := svc.Store.GetVersion(versionID)
		if err != nil {
			return nil, err
		}
		if v.Status == model.VersionFrozen {
			return nil, model.ErrFrozenWrite
		}
		if v.BatchID != sourceEntry.BatchID {
			return nil, model.ErrCrossBatch
		}
	}
	if relation == model.AlignConfirmed {
		sourceCounterexamples, err := svc.Store.CountCounterexamples(sourceID)
		if err != nil {
			return nil, err
		}
		targetCounterexamples, err := svc.Store.CountCounterexamples(targetID)
		if err != nil {
			return nil, err
		}
		if sourceCounterexamples > 0 || targetCounterexamples > 0 {
			return nil, model.ErrCounterexampleConflict
		}
	}
	existing, err := svc.Store.GetAlignmentPair(sourceID, targetID)
	if err != nil {
		if err != model.ErrNotFound {
			return nil, err
		}
		existing = &model.Alignment{
			ID:            store.NewID("a"),
			SourceSenseID: sourceID,
			TargetSenseID: targetID,
			Relation:      model.AlignCandidate,
			CreatedAt:     time.Now().UTC(),
		}
	}
	existing.Relation = relation
	existing.Reason = reason
	existing.VersionID = versionID
	existing.DecidedAt = time.Now().UTC()
	if err := svc.Store.SaveAlignment(existing); err != nil {
		return nil, err
	}
	return existing, nil
}

// GetAlignment 读取对齐关系。
func (svc *Service) GetAlignment(id string) (*model.Alignment, error) {
	return svc.Store.GetAlignment(id)
}

// ListAlignments 列出批次内全部对齐。
func (svc *Service) ListAlignments(batchID string) ([]*model.Alignment, error) {
	return svc.Store.ListAlignmentsForBatch(batchID)
}

// SplitSense 拆分多义词（委托 adjudicate）。
func (svc *Service) SplitSense(senseID string, targetIDs []string) ([]*model.Sense, error) {
	return adjudicate.SplitSense(svc.Store, senseID, targetIDs)
}
