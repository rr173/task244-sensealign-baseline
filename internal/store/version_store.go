package store

import (
	"database/sql"
	"time"

	"task244-sensealign/internal/model"
)

// CreateVersion 在批次下新建映射版本，初始状态草稿。
func (s *Store) CreateVersion(batchID, name string) (*model.MappingVersion, error) {
	if name == "" {
		return nil, model.ErrEmptyDefinition
	}
	v := &model.MappingVersion{
		ID:        NewID("v"),
		BatchID:   batchID,
		Name:      name,
		Status:    model.VersionDraft,
		CreatedAt: time.Now().UTC(),
	}
	_, err := s.DB.Exec(
		`INSERT INTO mapping_versions(id, batch_id, name, status, snapshot, frozen_at, created_at)
		 VALUES(?,?,?,?,?,?,?)`,
		v.ID, v.BatchID, v.Name, string(v.Status), "", "", NowRFC(),
	)
	if err != nil {
		return nil, err
	}
	return v, nil
}

// GetVersion 按 ID 读取版本。
func (s *Store) GetVersion(id string) (*model.MappingVersion, error) {
	row := s.DB.QueryRow(
		`SELECT id, batch_id, name, status, snapshot, frozen_at, created_at FROM mapping_versions WHERE id=?`, id,
	)
	return scanVersion(row)
}

// ListVersions 列出批次全部版本（按创建时间倒序）。
func (s *Store) ListVersions(batchID string) ([]*model.MappingVersion, error) {
	rows, err := s.DB.Query(
		`SELECT id, batch_id, name, status, snapshot, frozen_at, created_at FROM mapping_versions WHERE batch_id=? ORDER BY created_at DESC`,
		batchID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.MappingVersion
	for rows.Next() {
		v, err := scanVersion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// FreezeVersion 冻结版本：写入不可变快照与冻结时间。
func (s *Store) FreezeVersion(id, snapshot string) error {
	res, err := s.DB.Exec(
		`UPDATE mapping_versions SET status=?, snapshot=?, frozen_at=? WHERE id=?`,
		string(model.VersionFrozen), snapshot, NowRFC(), id,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return model.ErrNotFound
	}
	return nil
}

// ShareVersion 将草稿版本标记为共享（仍可变，区别于冻结）。
// 冻结版本不可变，拒绝改写（不更新任何行）。
func (s *Store) ShareVersion(id string) error {
	res, err := s.DB.Exec(
		`UPDATE mapping_versions SET status=? WHERE id=? AND status<>?`,
		string(model.VersionShared), id, string(model.VersionFrozen),
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		// 区分"不存在"与"已冻结"：先确认版本是否存在。
		if _, err := s.GetVersion(id); err != nil {
			return err
		}
		return model.ErrVersionFrozen
	}
	return nil
}

// SupersedeVersion 将某版本标记为替代（被新版取代）。冻结版本不可变，
// 拒绝改写（不更新任何行），以保护已发布快照的不可变性。
func (s *Store) SupersedeVersion(id string) error {
	res, err := s.DB.Exec(
		`UPDATE mapping_versions SET status=? WHERE id=? AND status<>?`,
		string(model.VersionSuperseded), id, string(model.VersionFrozen),
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		// 区分"不存在"与"已冻结"：先确认版本是否存在。
		if _, err := s.GetVersion(id); err != nil {
			return err
		}
		return model.ErrVersionFrozen
	}
	return nil
}

func scanVersion(scanner interface {
	Scan(dest ...any) error
}) (*model.MappingVersion, error) {
	var id, bid, name, status, snap, fa, ca string
	if err := scanner.Scan(&id, &bid, &name, &status, &snap, &fa, &ca); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	return &model.MappingVersion{
		ID:        id,
		BatchID:   bid,
		Name:      name,
		Status:    model.VersionStatus(status),
		Snapshot:  snap,
		FrozenAt:  ParseTime(fa),
		CreatedAt: ParseTime(ca),
	}, nil
}
