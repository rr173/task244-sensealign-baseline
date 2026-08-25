package store

import (
	"database/sql"
	"time"

	"task244-sensealign/internal/model"
)

// CreateBatch 新建词典批次，初始状态为整理中。
func (s *Store) CreateBatch(name, description string) (*model.Batch, error) {
	if name == "" {
		return nil, model.ErrEmptyDefinition
	}
	b := &model.Batch{
		ID:          NewID("b"),
		Name:        name,
		Description: description,
		Status:      model.BatchOrganizing,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	_, err := s.DB.Exec(
		`INSERT INTO batches(id, name, description, status, created_at, updated_at)
		 VALUES(?,?,?,?,?,?)`,
		b.ID, b.Name, b.Description, string(b.Status), NowRFC(), NowRFC(),
	)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// GetBatch 按 ID 读取批次。
func (s *Store) GetBatch(id string) (*model.Batch, error) {
	row := s.DB.QueryRow(
		`SELECT id, name, description, status, created_at, updated_at FROM batches WHERE id=?`, id,
	)
	return scanBatch(row)
}

// ListBatches 列出全部批次（按创建时间倒序）。
func (s *Store) ListBatches() ([]*model.Batch, error) {
	rows, err := s.DB.Query(
		`SELECT id, name, description, status, created_at, updated_at FROM batches ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Batch
	for rows.Next() {
		b, err := scanBatch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// SetBatchStatus 更新批次状态（写入校验在 service 层完成）。
func (s *Store) SetBatchStatus(id string, status model.BatchStatus) error {
	res, err := s.DB.Exec(
		`UPDATE batches SET status=?, updated_at=? WHERE id=?`,
		string(status), NowRFC(), id,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return model.ErrNotFound
	}
	return nil
}

// CountEntries 统计批次内词条数。
func (s *Store) CountEntries(batchID string) (int, error) {
	var n int
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM entries WHERE batch_id=?`, batchID).Scan(&n)
	return n, err
}

func scanBatch(scanner interface {
	Scan(dest ...any) error
}) (*model.Batch, error) {
	var (
		id, name, desc, status, ca, ua string
	)
	if err := scanner.Scan(&id, &name, &desc, &status, &ca, &ua); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	return &model.Batch{
		ID:        id,
		Name:      name,
		Description: desc,
		Status:    model.BatchStatus(status),
		CreatedAt: ParseTime(ca),
		UpdatedAt: ParseTime(ua),
	}, nil
}
