// Package service 是跨语言义项对齐复核台的业务编排层，串接 store / evidence /
// matcher / adjudicate，向 HTTP 层暴露高层能力，并内置端到端自检场景。
package service

import (
	"task244-sensealign/internal/evidence"
	"task244-sensealign/internal/model"
	"task244-sensealign/internal/store"
)

// Service 聚合存储与领域规则。
type Service struct {
	Store *store.Store
}

// New 构造服务。
func New(s *store.Store) *Service {
	return &Service{Store: s}
}

// CreateBatch 新建词典批次。
func (svc *Service) CreateBatch(name, desc string) (*model.Batch, error) {
	return svc.Store.CreateBatch(name, desc)
}

// GetBatch 读取批次。
func (svc *Service) GetBatch(id string) (*model.Batch, error) {
	return svc.Store.GetBatch(id)
}

// ListBatches 列出批次。
func (svc *Service) ListBatches() ([]*model.Batch, error) {
	return svc.Store.ListBatches()
}

// SetBatchStatus 更新批次状态（封存为终态，后续写入由各写操作拦截）。
func (svc *Service) SetBatchStatus(id string, status model.BatchStatus) error {
	if !model.ValidBatchStatus(string(status)) {
		return model.ErrEmptyDefinition
	}
	b, err := svc.Store.GetBatch(id)
	if err != nil {
		return err
	}
	if b.Status == model.BatchSealed && status != model.BatchSealed {
		return model.ErrBatchSealed
	}
	if !model.ValidBatchTransition(b.Status, status) {
		return model.ErrInvalidBatchTransition
	}
	return svc.Store.SetBatchStatus(id, status)
}

// AddEntry 在批次下新增词条（批次已封存则拒绝）。
func (svc *Service) AddEntry(batchID, lang, headword, gloss string) (*model.Entry, error) {
	b, err := svc.Store.GetBatch(batchID)
	if err != nil {
		return nil, err
	}
	if b.Status == model.BatchSealed {
		return nil, model.ErrBatchSealed
	}
	return svc.Store.CreateEntry(batchID, lang, headword, gloss)
}

// GetEntry 读取词条。
func (svc *Service) GetEntry(id string) (*model.Entry, error) {
	return svc.Store.GetEntry(id)
}

// ListEntries 列出批次词条。
func (svc *Service) ListEntries(batchID string) ([]*model.Entry, error) {
	return svc.Store.ListEntries(batchID)
}

// AddSense 在词条下新增义项，语域标签按受控词表规范化。批次已封存则拒绝。
func (svc *Service) AddSense(entryID, definition string, registers []string) (*model.Sense, error) {
	e, err := svc.Store.GetEntry(entryID)
	if err != nil {
		return nil, model.ErrUnknownEntry
	}
	if err := svc.ensureBatchWritable(e.BatchID); err != nil {
		return nil, err
	}
	tags := evidence.NormalizeRegisters(registers)
	return svc.Store.CreateSense(entryID, definition, tags)
}

// GetSense 读取义项。
func (svc *Service) GetSense(id string) (*model.Sense, error) {
	return svc.Store.GetSense(id)
}

// ListSenses 列出词条义项。
func (svc *Service) ListSenses(entryID string) ([]*model.Sense, error) {
	return svc.Store.ListSenses(entryID)
}

// SetRegisters 覆盖义项语域标签（规范化）。
func (svc *Service) SetRegisters(senseID string, registers []string) error {
	sn, err := svc.Store.GetSense(senseID)
	if err != nil {
		return err
	}
	if _, err := svc.Store.GetEntry(sn.EntryID); err != nil {
		return err
	}
	if err := svc.Store.SetSenseRegisters(senseID, evidence.NormalizeRegisters(registers)); err != nil {
		return err
	}
	return svc.refreshAlignmentEvidence(senseID)
}

// AddExample 为义项新增例句。批次已封存则拒绝。
func (svc *Service) AddExample(senseID, text, lang, trans, register string) (*model.Example, error) {
	sn, err := svc.Store.GetSense(senseID)
	if err != nil {
		return nil, model.ErrUnknownSense
	}
	e, err := svc.Store.GetEntry(sn.EntryID)
	if err != nil {
		return nil, err
	}
	if err := svc.ensureBatchWritable(e.BatchID); err != nil {
		return nil, err
	}
	return svc.Store.CreateExample(senseID, text, lang, trans, register)
}

// ListExamples 列出义项例句。
func (svc *Service) ListExamples(senseID string) ([]*model.Example, error) {
	return svc.Store.ListExamples(senseID)
}

// AddCounterexample 为义项新增反例证据。批次已封存则拒绝。
func (svc *Service) AddCounterexample(senseID, text, note string) (*model.Counterexample, error) {
	sn, err := svc.Store.GetSense(senseID)
	if err != nil {
		return nil, model.ErrUnknownSense
	}
	e, err := svc.Store.GetEntry(sn.EntryID)
	if err != nil {
		return nil, err
	}
	if err := svc.ensureBatchWritable(e.BatchID); err != nil {
		return nil, err
	}
	return svc.Store.CreateCounterexample(senseID, text, note)
}

func (svc *Service) ensureBatchWritable(batchID string) error {
	b, err := svc.Store.GetBatch(batchID)
	if err != nil {
		return err
	}
	if b.Status == model.BatchSealed {
		return model.ErrBatchSealed
	}
	return nil
}

func (svc *Service) refreshAlignmentEvidence(senseID string) error {
	alignments, err := svc.Store.ListAlignmentsBySense(senseID)
	if err != nil {
		return err
	}
	for _, a := range alignments {
		src, err := svc.Store.GetSense(a.SourceSenseID)
		if err != nil {
			return err
		}
		tgt, err := svc.Store.GetSense(a.TargetSenseID)
		if err != nil {
			return err
		}
		_, a.RegisterConflict = evidence.RegisterCompatibility(src.RegisterTags, tgt.RegisterTags)
		if err := svc.Store.SaveAlignment(a); err != nil {
			return err
		}
	}
	return nil
}

// ListCounterexamples 列出义项反例。
func (svc *Service) ListCounterexamples(senseID string) ([]*model.Counterexample, error) {
	return svc.Store.ListCounterexamples(senseID)
}
