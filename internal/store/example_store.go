package store

import (
	"database/sql"
	"time"

	"task244-sensealign/internal/model"
)

// CreateExample 为义项新增例句。
func (s *Store) CreateExample(senseID, text, langCode, translation, register string) (*model.Example, error) {
	if text == "" {
		return nil, model.ErrEmptyDefinition
	}
	ex := &model.Example{
		ID:          NewID("x"),
		SenseID:     senseID,
		Text:        text,
		LangCode:    langCode,
		Translation: translation,
		Register:    register,
		CreatedAt:   time.Now().UTC(),
	}
	_, err := s.DB.Exec(
		`INSERT INTO examples(id, sense_id, text, lang_code, translation, register, created_at)
		 VALUES(?,?,?,?,?,?,?)`,
		ex.ID, ex.SenseID, ex.Text, ex.LangCode, ex.Translation, ex.Register, NowRFC(),
	)
	if err != nil {
		return nil, err
	}
	return ex, nil
}

// GetExample 按 ID 读取例句。
func (s *Store) GetExample(id string) (*model.Example, error) {
	row := s.DB.QueryRow(
		`SELECT id, sense_id, text, lang_code, translation, register, created_at FROM examples WHERE id=?`, id,
	)
	return scanExample(row)
}

// ListExamples 列出义项下全部例句。
func (s *Store) ListExamples(senseID string) ([]*model.Example, error) {
	rows, err := s.DB.Query(
		`SELECT id, sense_id, text, lang_code, translation, register, created_at FROM examples WHERE sense_id=? ORDER BY created_at`,
		senseID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Example
	for rows.Next() {
		ex, err := scanExample(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ex)
	}
	return out, rows.Err()
}

// ListExamplesBySenseIDs 按义项 ID 批量读取，返回 senseID -> 例句列表。
func (s *Store) ListExamplesBySenseIDs(senseIDs []string) (map[string][]*model.Example, error) {
	out := make(map[string][]*model.Example, len(senseIDs))
	for _, sid := range senseIDs {
		exs, err := s.ListExamples(sid)
		if err != nil {
			return nil, err
		}
		out[sid] = exs
	}
	return out, nil
}

func scanExample(scanner interface {
	Scan(dest ...any) error
}) (*model.Example, error) {
	var id, sid, text, lang, trans, reg, ca string
	if err := scanner.Scan(&id, &sid, &text, &lang, &trans, &reg, &ca); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	return &model.Example{
		ID:          id,
		SenseID:     sid,
		Text:        text,
		LangCode:    lang,
		Translation: trans,
		Register:    reg,
		CreatedAt:   ParseTime(ca),
	}, nil
}
