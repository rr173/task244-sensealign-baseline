package httpapi

import (
	"net/http"
	"strings"

	"task244-sensealign/internal/model"
)

type setRegistersReq struct {
	RegisterTags []string `json:"register_tags"`
}

type splitSenseReq struct {
	TargetSenseIDs []string `json:"target_sense_ids"`
}

type addExampleReq struct {
	Text        string `json:"text"`
	LangCode    string `json:"lang_code"`
	Translation string `json:"translation"`
	Register    string `json:"register"`
}

type addCounterReq struct {
	Text string `json:"text"`
	Note string `json:"note"`
}

// handleSenseSub 处理 /api/senses/{id} 及其子资源（语域、拆分、例句、反例）。
func (s *Server) handleSenseSub(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/senses/")
	parts := strings.Split(rest, "/")
	id := parts[0]
	if id == "" {
		writeError(w, model.ErrUnknownSense)
		return
	}
	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			writeError(w, model.ErrBadRelation)
			return
		}
		sn, err := s.svc.GetSense(id)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, sn)
		return
	}
	switch parts[1] {
	case "registers":
		if r.Method != http.MethodPut {
			writeError(w, model.ErrBadRelation)
			return
		}
		var req setRegistersReq
		if err := decodeBody(r, &req); err != nil {
			writeError(w, model.ErrEmptyDefinition)
			return
		}
		if err := s.svc.SetRegisters(id, req.RegisterTags); err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"id": id, "register_tags": req.RegisterTags})
	case "split":
		if r.Method != http.MethodPost {
			writeError(w, model.ErrBadRelation)
			return
		}
		var req splitSenseReq
		if err := decodeBody(r, &req); err != nil {
			writeError(w, model.ErrEmptyDefinition)
			return
		}
		created, err := s.svc.SplitSense(id, req.TargetSenseIDs)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, created)
	case "examples":
		switch r.Method {
		case http.MethodPost:
			var req addExampleReq
			if err := decodeBody(r, &req); err != nil {
				writeError(w, model.ErrEmptyDefinition)
				return
			}
			ex, err := s.svc.AddExample(id, req.Text, req.LangCode, req.Translation, req.Register)
			if err != nil {
				writeError(w, err)
				return
			}
			writeJSON(w, http.StatusCreated, ex)
		case http.MethodGet:
			exs, err := s.svc.ListExamples(id)
			if err != nil {
				writeError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, exs)
		default:
			writeError(w, model.ErrBadRelation)
		}
	case "counterexamples":
		switch r.Method {
		case http.MethodPost:
			var req addCounterReq
			if err := decodeBody(r, &req); err != nil {
				writeError(w, model.ErrEmptyDefinition)
				return
			}
			c, err := s.svc.AddCounterexample(id, req.Text, req.Note)
			if err != nil {
				writeError(w, err)
				return
			}
			writeJSON(w, http.StatusCreated, c)
		case http.MethodGet:
			cs, err := s.svc.ListCounterexamples(id)
			if err != nil {
				writeError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, cs)
		default:
			writeError(w, model.ErrBadRelation)
		}
	default:
		writeError(w, model.ErrUnknownSense)
	}
}
