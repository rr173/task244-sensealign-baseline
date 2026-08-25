package service

import (
	"task244-sensealign/internal/adjudicate"
	"task244-sensealign/internal/model"
)

// Stats 汇总批次规模与冲突计数，供端到端验收与页面展示。
type Stats struct {
	BatchID          string `json:"batch_id"`
	Entries          int    `json:"entries"`
	Senses           int    `json:"senses"`
	Examples         int    `json:"examples"`
	Alignments       int    `json:"alignments"`
	Confirmed        int    `json:"confirmed"`
	Partial          int    `json:"partial"`
	Rejected         int    `json:"rejected"`
	Versions         int    `json:"versions"`
	OneToManyConflicts int  `json:"one_to_many_conflicts"`
	RegisterConflicts   int `json:"register_conflicts"`
	HypernymPairs        int `json:"hypernym_pairs"`
}

// BatchStats 计算批次统计。
func (svc *Service) BatchStats(batchID string) (*Stats, error) {
	if _, err := svc.Store.GetBatch(batchID); err != nil {
		return nil, err
	}
	st := &Stats{BatchID: batchID}
	entries, err := svc.Store.ListEntries(batchID)
	if err != nil {
		return nil, err
	}
	st.Entries = len(entries)
	for _, e := range entries {
		senses, err := svc.Store.ListSenses(e.ID)
		if err != nil {
			return nil, err
		}
		st.Senses += len(senses)
		for _, sn := range senses {
			exs, err := svc.Store.ListExamples(sn.ID)
			if err != nil {
				return nil, err
			}
			st.Examples += len(exs)
		}
	}
	aligns, err := svc.Store.ListAlignmentsForBatch(batchID)
	if err != nil {
		return nil, err
	}
	st.Alignments = len(aligns)
	for _, a := range aligns {
		switch a.Relation {
		case model.AlignConfirmed:
			st.Confirmed++
		case model.AlignPartial:
			st.Partial++
		case model.AlignRejected:
			st.Rejected++
		}
	}
	versions, err := svc.Store.ListVersions(batchID)
	if err != nil {
		return nil, err
	}
	st.Versions = len(versions)
	rep, err := adjudicate.BuildConflictReport(svc.Store, batchID)
	if err != nil {
		return nil, err
	}
	sum := rep.Summarize()
	st.OneToManyConflicts = sum["one_to_many"]
	st.RegisterConflicts = sum["register_conflict"]
	st.HypernymPairs = sum["hypernym_pairs"]
	return st, nil
}
