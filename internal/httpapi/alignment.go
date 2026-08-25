package httpapi

import (
	"net/http"
	"strings"

	"task244-sensealign/internal/model"
)

type candidatesReq struct {
	SourceEntryID string `json:"source_entry_id"`
	TargetEntryID string `json:"target_entry_id"`
}

type decideReq struct {
	SourceSenseID string `json:"source_sense_id"`
	TargetSenseID string `json:"target_sense_id"`
	Relation      string `json:"relation"`
	Reason        string `json:"reason"`
	VersionID     string `json:"version_id"`
}

// handleAlignCandidates 处理 POST /api/align/candidates，生成并落盘候选对齐。
func (s *Server) handleAlignCandidates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, model.ErrBadRelation)
		return
	}
	var req candidatesReq
	if err := decodeBody(r, &req); err != nil {
		writeError(w, model.ErrEmptyDefinition)
		return
	}
	cands, err := s.svc.GenerateCandidates(req.SourceEntryID, req.TargetEntryID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cands)
}

// handleAlignDecide 处理 POST /api/alignments/decide，对义项配对做出裁决。
func (s *Server) handleAlignDecide(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, model.ErrBadRelation)
		return
	}
	var req decideReq
	if err := decodeBody(r, &req); err != nil {
		writeError(w, model.ErrEmptyDefinition)
		return
	}
	a, err := s.svc.DecideAlignment(req.SourceSenseID, req.TargetSenseID, model.AlignRelation(req.Relation), req.Reason, req.VersionID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

// handleAlignmentSub 处理 /api/alignments/{id}（读取单条对齐）。
func (s *Server) handleAlignmentSub(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/alignments/")
	id := rest
	if id == "" || strings.Contains(id, "/") {
		writeError(w, model.ErrNotFound)
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, model.ErrBadRelation)
		return
	}
	a, err := s.svc.GetAlignment(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, a)
}
