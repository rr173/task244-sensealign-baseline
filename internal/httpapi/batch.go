package httpapi

import (
	"net/http"
	"strings"

	"task244-sensealign/internal/model"
)

type createBatchReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type setStatusReq struct {
	Status string `json:"status"`
}

type addEntryReq struct {
	LangCode string `json:"lang_code"`
	Headword string `json:"headword"`
	Gloss    string `json:"gloss"`
}

// handleBatches 处理 POST /api/batches（新建）与 GET /api/batches（列表）。
func (s *Server) handleBatches(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req createBatchReq
		if err := decodeBody(r, &req); err != nil {
			writeError(w, model.ErrEmptyDefinition)
			return
		}
		b, err := s.svc.CreateBatch(req.Name, req.Description)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, b)
	case http.MethodGet:
		bs, err := s.svc.ListBatches()
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, bs)
	default:
		writeError(w, model.ErrBadRelation)
	}
}

// handleBatchSub 处理 /api/batches/{id} 及其子资源。
func (s *Server) handleBatchSub(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/batches/")
	parts := strings.Split(rest, "/")
	id := parts[0]
	if id == "" {
		writeError(w, model.ErrUnknownBatch)
		return
	}
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			b, err := s.svc.GetBatch(id)
			if err != nil {
				writeError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, b)
		case http.MethodPut:
			var req setStatusReq
			if err := decodeBody(r, &req); err != nil {
				writeError(w, model.ErrEmptyDefinition)
				return
			}
			if err := s.svc.SetBatchStatus(id, model.BatchStatus(req.Status)); err != nil {
				writeError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"id": id, "status": req.Status})
		default:
			writeError(w, model.ErrBadRelation)
		}
		return
	}
	switch parts[1] {
	case "entries":
		switch r.Method {
		case http.MethodPost:
			var req addEntryReq
			if err := decodeBody(r, &req); err != nil {
				writeError(w, model.ErrEmptyDefinition)
				return
			}
			e, err := s.svc.AddEntry(id, req.LangCode, req.Headword, req.Gloss)
			if err != nil {
				writeError(w, err)
				return
			}
			writeJSON(w, http.StatusCreated, e)
		case http.MethodGet:
			es, err := s.svc.ListEntries(id)
			if err != nil {
				writeError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, es)
		default:
			writeError(w, model.ErrBadRelation)
		}
	case "stats":
		if r.Method != http.MethodGet {
			writeError(w, model.ErrBadRelation)
			return
		}
		st, err := s.svc.BatchStats(id)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, st)
	case "alignments":
		if r.Method != http.MethodGet {
			writeError(w, model.ErrBadRelation)
			return
		}
		as, err := s.svc.ListAlignments(id)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, as)
	case "versions":
		if r.Method != http.MethodGet {
			writeError(w, model.ErrBadRelation)
			return
		}
		vs, err := s.svc.ListVersions(id)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, vs)
	default:
		writeError(w, model.ErrUnknownBatch)
	}
}
