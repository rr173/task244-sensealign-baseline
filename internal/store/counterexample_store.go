package store

import (
	"time"

	"task244-sensealign/internal/model"
)

// CreateCounterexample 为义项新增反例证据。
func (s *Store) CreateCounterexample(senseID, text, note string) (*model.Counterexample, error) {
	if text == "" {
		return nil, model.ErrEmptyDefinition
	}
	c := &model.Counterexample{
		ID:        NewID("cx"),
		SenseID:   senseID,
		Text:      text,
		Note:      note,
		CreatedAt: time.Now().UTC(),
	}
	_, err := s.DB.Exec(
		`INSERT INTO counterexamples(id, sense_id, text, note, created_at) VALUES(?,?,?,?,?)`,
		c.ID, c.SenseID, c.Text, c.Note, NowRFC(),
	)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// ListCounterexamples 列出义项全部反例。
func (s *Store) ListCounterexamples(senseID string) ([]*model.Counterexample, error) {
	rows, err := s.DB.Query(
		`SELECT id, sense_id, text, note, created_at FROM counterexamples WHERE sense_id=? ORDER BY created_at`,
		senseID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Counterexample
	for rows.Next() {
		var id, sid, text, note, ca string
		if err := rows.Scan(&id, &sid, &text, &note, &ca); err != nil {
			return nil, err
		}
		out = append(out, &model.Counterexample{
			ID:        id,
			SenseID:   sid,
			Text:      text,
			Note:      note,
			CreatedAt: ParseTime(ca),
		})
	}
	return out, rows.Err()
}

// CountCounterexamples 统计义项反例数（用于裁决提示）。
func (s *Store) CountCounterexamples(senseID string) (int, error) {
	var n int
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM counterexamples WHERE sense_id=?`, senseID).Scan(&n)
	return n, err
}
