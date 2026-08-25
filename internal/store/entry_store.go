package store

import (
	"database/sql"
	"time"

	"task244-sensealign/internal/model"
)

// CreateEntry 在批次下新建词条。lang_code 必填；(batch_id, lang_code, headword) 唯一。
func (s *Store) CreateEntry(batchID, langCode, headword, gloss string) (*model.Entry, error) {
	if langCode == "" {
		return nil, model.ErrInvalidLang
	}
	if headword == "" {
		return nil, model.ErrEmptyDefinition
	}
	e := &model.Entry{
		ID:        NewID("e"),
		BatchID:   batchID,
		LangCode:  langCode,
		Headword:  headword,
		Gloss:     gloss,
		CreatedAt: time.Now().UTC(),
	}
	_, err := s.DB.Exec(
		`INSERT INTO entries(id, batch_id, lang_code, headword, gloss, created_at)
		 VALUES(?,?,?,?,?,?)`,
		e.ID, e.BatchID, e.LangCode, e.Headword, e.Gloss, NowRFC(),
	)
	if isUniqueErr(err) {
		return nil, model.ErrDuplicate
	}
	if err != nil {
		return nil, err
	}
	return e, nil
}

// GetEntry 按 ID 读取词条。
func (s *Store) GetEntry(id string) (*model.Entry, error) {
	row := s.DB.QueryRow(
		`SELECT id, batch_id, lang_code, headword, gloss, created_at FROM entries WHERE id=?`, id,
	)
	return scanEntry(row)
}

// ListEntries 列出批次下全部词条。
func (s *Store) ListEntries(batchID string) ([]*model.Entry, error) {
	rows, err := s.DB.Query(
		`SELECT id, batch_id, lang_code, headword, gloss, created_at FROM entries WHERE batch_id=? ORDER BY created_at`,
		batchID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Entry
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ListEntriesByIDs 按 ID 列表批量读取（保持传入顺序）。
func (s *Store) ListEntriesByIDs(ids []string) (map[string]*model.Entry, error) {
	out := make(map[string]*model.Entry, len(ids))
	for _, id := range ids {
		e, err := s.GetEntry(id)
		if err != nil {
			return nil, err
		}
		out[id] = e
	}
	return out, nil
}

func scanEntry(scanner interface {
	Scan(dest ...any) error
}) (*model.Entry, error) {
	var id, bid, lang, hw, gloss, ca string
	if err := scanner.Scan(&id, &bid, &lang, &hw, &gloss, &ca); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	return &model.Entry{
		ID:        id,
		BatchID:   bid,
		LangCode:  lang,
		Headword:  hw,
		Gloss:     gloss,
		CreatedAt: ParseTime(ca),
	}, nil
}

// isUniqueErr 粗略判断是否为唯一约束冲突（驱动名差异下用错误文本匹配）。
func isUniqueErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return containsAny(msg, "UNIQUE constraint failed", "duplicate key", "constraint failed")
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(sub) > 0 && indexOf(s, sub) >= 0 {
			return true
		}
	}
	return false
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
