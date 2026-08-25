package service

import (
	"encoding/json"
	"time"

	"task244-sensealign/internal/adjudicate"
	"task244-sensealign/internal/model"
)

// CreateVersion 新建映射版本（草稿）。
func (svc *Service) CreateVersion(batchID, name string) (*model.MappingVersion, error) {
	if _, err := svc.Store.GetBatch(batchID); err != nil {
		return nil, err
	}
	return svc.Store.CreateVersion(batchID, name)
}

// GetVersion 读取版本。
func (svc *Service) GetVersion(id string) (*model.MappingVersion, error) {
	return svc.Store.GetVersion(id)
}

// ListVersions 列出批次版本。
func (svc *Service) ListVersions(batchID string) ([]*model.MappingVersion, error) {
	return svc.Store.ListVersions(batchID)
}

// FreezeVersion 冻结版本：采集当前已裁决对齐为不可变快照后冻结。
func (svc *Service) FreezeVersion(versionID string) error {
	v, err := svc.Store.GetVersion(versionID)
	if err != nil {
		return err
	}
	if v.Status == model.VersionFrozen {
		return model.ErrVersionFrozen
	}
	snap, err := svc.BuildSnapshot(v.BatchID)
	if err != nil {
		return err
	}
	return adjudicate.FreezeVersion(svc.Store, versionID, snap)
}

// ShareVersion 将版本标记为共享。冻结版本不可变，拒绝改写。
func (svc *Service) ShareVersion(versionID string) error {
	v, err := svc.Store.GetVersion(versionID)
	if err != nil {
		return err
	}
	if v.Status == model.VersionFrozen {
		return model.ErrVersionFrozen
	}
	return svc.Store.ShareVersion(versionID)
}

// SupersedeVersion 将版本标记为替代。冻结版本不可变，拒绝改写，
// 否则会令已发布的不可变快照失去冻结标记、绕过裁决写入保护。
func (svc *Service) SupersedeVersion(versionID string) error {
	v, err := svc.Store.GetVersion(versionID)
	if err != nil {
		return err
	}
	if v.Status == model.VersionFrozen {
		return model.ErrVersionFrozen
	}
	return svc.Store.SupersedeVersion(versionID)
}

// Snapshot 映射版本快照载荷。
type Snapshot struct {
	BatchID     string            `json:"batch_id"`
	GeneratedAt string            `json:"generated_at"`
	Alignments  []model.Alignment `json:"alignments"`
}

// BuildSnapshot 采集批次内已裁决（确认/部分重合）对齐为快照 JSON。
func (svc *Service) BuildSnapshot(batchID string) (string, error) {
	all, err := svc.Store.ListAlignmentsForBatch(batchID)
	if err != nil {
		return "", err
	}
	var kept []model.Alignment
	for _, a := range all {
		if a.Relation == model.AlignConfirmed || a.Relation == model.AlignPartial {
			kept = append(kept, *a)
		}
	}
	snap := Snapshot{
		BatchID:     batchID,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Alignments:  kept,
	}
	b, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
