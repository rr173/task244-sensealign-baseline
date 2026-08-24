package store

import (
	"database/sql"
	"time"

	"task244-sensealign/internal/model"
)

// SaveAlignment 写入或覆盖一条对齐关系（按 source/target 唯一）。
func (s *Store) SaveAlignment(a *model.Alignment) error {
	decided := ""
	if !a.DecidedAt.IsZero() {
		decided = a.DecidedAt.UTC().Format(time.RFC3339)
	}
	_, err := s.DB.Exec(
		`INSERT INTO alignments(
			id, source_sense_id, target_sense_id, relation, coverage_score,
			hypernym, hyponym, register_conflict, reason, version_id, created_at, decided_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?,?,?)
		 ON CONFLICT(source_sense_id, target_sense_id) DO UPDATE SET
			id=excluded.id, relation=excluded.relation, coverage_score=excluded.coverage_score,
			hypernym=excluded.hypernym, hyponym=excluded.hyponym,
			register_conflict=excluded.register_conflict, reason=excluded.reason,
			version_id=excluded.version_id, decided_at=excluded.decided_at`,
		a.ID, a.SourceSenseID, a.TargetSenseID, string(a.Relation), a.CovScore,
		boolToInt(a.Hypernym), boolToInt(a.Hyponym), boolToInt(a.RegisterConflict),
		a.Reason, a.VersionID, a.CreatedAt.UTC().Format(time.RFC3339), decided,
	)
	return err
}

// GetAlignment 按 ID 读取对齐关系。
func (s *Store) GetAlignment(id string) (*model.Alignment, error) {
	row := s.DB.QueryRow(
		`SELECT id, source_sense_id, target_sense_id, relation, coverage_score,
		 hypernym, hyponym, register_conflict, reason, version_id, created_at, decided_at
		 FROM alignments WHERE id=?`, id,
	)
	return scanAlignment(row)
}

// GetAlignmentPair 按义项对读取对齐关系。
func (s *Store) GetAlignmentPair(sourceID, targetID string) (*model.Alignment, error) {
	row := s.DB.QueryRow(
		`SELECT id, source_sense_id, target_sense_id, relation, coverage_score,
		 hypernym, hyponym, register_conflict, reason, version_id, created_at, decided_at
		 FROM alignments WHERE source_sense_id=? AND target_sense_id=?`, sourceID, targetID,
	)
	return scanAlignment(row)
}

// ListAlignmentsBySource 列出某义项作为源的全部对齐。
func (s *Store) ListAlignmentsBySource(sourceID string) ([]*model.Alignment, error) {
	return s.listAlignments(
		`SELECT id, source_sense_id, target_sense_id, relation, coverage_score,
		 hypernym, hyponym, register_conflict, reason, version_id, created_at, decided_at
		 FROM alignments WHERE source_sense_id=? ORDER BY coverage_score DESC`, sourceID,
	)
}

// ListAlignmentsByVersion 列出某映射版本纳入的全部对齐。
func (s *Store) ListAlignmentsByVersion(versionID string) ([]*model.Alignment, error) {
	return s.listAlignments(
		`SELECT id, source_sense_id, target_sense_id, relation, coverage_score,
		 hypernym, hyponym, register_conflict, reason, version_id, created_at, decided_at
		 FROM alignments WHERE version_id=? ORDER BY created_at`, versionID,
	)
}

// ListAlignmentsForBatch 列出批次内（经 source 义项归属）的全部对齐。
func (s *Store) ListAlignmentsForBatch(batchID string) ([]*model.Alignment, error) {
	return s.listAlignments(
		`SELECT a.id, a.source_sense_id, a.target_sense_id, a.relation, a.coverage_score,
		 a.hypernym, a.hyponym, a.register_conflict, a.reason, a.version_id, a.created_at, a.decided_at
		 FROM alignments a
		 JOIN senses s ON a.source_sense_id = s.id
		 JOIN entries e ON s.entry_id = e.id
		 WHERE e.batch_id=? ORDER BY a.created_at`, batchID,
	)
}

func (s *Store) listAlignments(query string, args ...any) ([]*model.Alignment, error) {
	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Alignment
	for rows.Next() {
		a, err := scanAlignment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func scanAlignment(scanner interface {
	Scan(dest ...any) error
}) (*model.Alignment, error) {
	var (
		id, src, tgt, rel, reason, vid, ca, da string
		cov                                     float64
		hyper, hypo, regc                        int
	)
	if err := scanner.Scan(&id, &src, &tgt, &rel, &cov, &hyper, &hypo, &regc, &reason, &vid, &ca, &da); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	a := &model.Alignment{
		ID:              id,
		SourceSenseID:   src,
		TargetSenseID:   tgt,
		Relation:        model.AlignRelation(rel),
		CovScore:        cov,
		Hypernym:        hyper == 1,
		Hyponym:         hypo == 1,
		RegisterConflict: regc == 1,
		Reason:          reason,
		VersionID:       vid,
		CreatedAt:       ParseTime(ca),
		DecidedAt:       ParseTime(da),
	}
	return a, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
