package httpapi

import (
	"net/http"
	"strings"

	"task244-sensealign/internal/model"
)

type addSenseReq struct {
	Definition   string   `json:"definition"`
	RegisterTags []string `json:"register_tags"`
}

// handleEntrySub 处理 /api/entries/{id} 与 /api/entries/{id}/senses。
func (s *Server) handleEntrySub(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/entries/")
	parts := strings.Split(rest, "/")
	id := parts[0]
	if id == "" {
		writeError(w, model.ErrUnknownEntry)
		return
	}
	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			writeError(w, model.ErrBadRelation)
			return
		}
		e, err := s.svc.GetEntry(id)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, e)
		return
	}
	switch parts[1] {
	case "senses":
		switch r.Method {
		case http.MethodPost:
			var req addSenseReq
			if err := decodeBody(r, &req); err != nil {
				writeError(w, model.ErrEmptyDefinition)
				return
			}
			sn, err := s.svc.AddSense(id, req.Definition, req.RegisterTags)
			if err != nil {
				writeError(w, err)
				return
			}
			writeJSON(w, http.StatusCreated, sn)
		case http.MethodGet:
			sns, err := s.svc.ListSenses(id)
			if err != nil {
				writeError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, sns)
		default:
			writeError(w, model.ErrBadRelation)
		}
	default:
		writeError(w, model.ErrUnknownEntry)
	}
}
