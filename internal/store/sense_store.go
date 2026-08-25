package store

import (
	"database/sql"
	"encoding/json"
	"time"

	"task244-sensealign/internal/model"
)

// CreateSense 在词条下新建义项，初始状态待校验，语域标签为空。
func (s *Store) CreateSense(entryID, definition string, registerTags []string) (*model.Sense, error) {
	if definition == "" {
		return nil, model.ErrEmptyDefinition
	}
	if registerTags == nil {
		registerTags = []string{}
	}
	tags, err := json.Marshal(registerTags)
	if err != nil {
		return nil, err
	}
	sn := &model.Sense{
		ID:          NewID("s"),
		EntryID:     entryID,
		Definition:  definition,
		Status:      model.SensePending,
		RegisterTags: registerTags,
		CreatedAt:   time.Now().UTC(),
	}
	_, err = s.DB.Exec(
		`INSERT INTO senses(id, entry_id, definition, status, register_tags, created_at)
		 VALUES(?,?,?,?,?,?)`,
		sn.ID, sn.EntryID, sn.Definition, string(sn.Status), string(tags), NowRFC(),
	)
	if err != nil {
		return nil, err
	}
	return sn, nil
}

// GetSense 按 ID 读取义项。
func (s *Store) GetSense(id string) (*model.Sense, error) {
	row := s.DB.QueryRow(
		`SELECT id, entry_id, definition, status, register_tags, created_at FROM senses WHERE id=?`, id,
	)
	return scanSense(row)
}

// ListSenses 列出词条下全部义项。
func (s *Store) ListSenses(entryID string) ([]*model.Sense, error) {
	rows, err := s.DB.Query(
		`SELECT id, entry_id, definition, status, register_tags, created_at FROM senses WHERE entry_id=? ORDER BY created_at`,
		entryID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Sense
	for rows.Next() {
		sn, err := scanSense(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sn)
	}
	return out, rows.Err()
}

// SetSenseStatus 更新义项状态。
func (s *Store) SetSenseStatus(id string, status model.SenseStatus) error {
	res, err := s.DB.Exec(
		`UPDATE senses SET status=? WHERE id=?`, string(status), id,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return model.ErrNotFound
	}
	return nil
}

// SetSenseRegisters 覆盖义项语域标签。
func (s *Store) SetSenseRegisters(id string, tags []string) error {
	if tags == nil {
		tags = []string{}
	}
	b, err := json.Marshal(tags)
	if err != nil {
		return err
	}
	res, err := s.DB.Exec(`UPDATE senses SET register_tags=? WHERE id=?`, string(b), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return model.ErrNotFound
	}
	return nil
}

func scanSense(scanner interface {
	Scan(dest ...any) error
}) (*model.Sense, error) {
	var id, eid, def, status, tags, ca string
	if err := scanner.Scan(&id, &eid, &def, &status, &tags, &ca); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	var rt []string
	if tags != "" {
		if err := json.Unmarshal([]byte(tags), &rt); err != nil {
			rt = []string{}
		}
	}
	if rt == nil {
		rt = []string{}
	}
	return &model.Sense{
		ID:          id,
		EntryID:     eid,
		Definition:  def,
		Status:      model.SenseStatus(status),
		RegisterTags: rt,
		CreatedAt:   ParseTime(ca),
	}, nil
}
