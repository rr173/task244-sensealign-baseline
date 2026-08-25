package httpapi

import (
	"net/http"
	"strings"

	"task244-sensealign/internal/model"
)

type createVersionReq struct {
	BatchID string `json:"batch_id"`
	Name    string `json:"name"`
}

// handleVersionsCreate 处理 POST /api/versions（新建映射版本）。
func (s *Server) handleVersionsCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, model.ErrBadRelation)
		return
	}
	var req createVersionReq
	if err := decodeBody(r, &req); err != nil {
		writeError(w, model.ErrEmptyDefinition)
		return
	}
	v, err := s.svc.CreateVersion(req.BatchID, req.Name)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, v)
}

// handleVersionSub 处理 /api/versions/{id} 及其子操作（freeze/share/supersede）。
func (s *Server) handleVersionSub(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/versions/")
	parts := strings.Split(rest, "/")
	id := parts[0]
	if id == "" {
		writeError(w, model.ErrNotFound)
		return
	}
	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			writeError(w, model.ErrBadRelation)
			return
		}
		v, err := s.svc.GetVersion(id)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, v)
		return
	}
	action := parts[1]
	if r.Method != http.MethodPost {
		writeError(w, model.ErrBadRelation)
		return
	}
	var err error
	switch action {
	case "freeze":
		err = s.svc.FreezeVersion(id)
	case "share":
		err = s.svc.ShareVersion(id)
	case "supersede":
		err = s.svc.SupersedeVersion(id)
	default:
		writeError(w, model.ErrNotFound)
		return
	}
	if err != nil {
		writeError(w, err)
		return
	}
	v, err := s.svc.GetVersion(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}
