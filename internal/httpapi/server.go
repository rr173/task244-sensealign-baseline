// Package httpapi 提供 /api 前缀的 JSON HTTP 接口与轻量复核页面，
// 消费 service 层能力，不做业务规则判断（规则在 service / store 内）。
package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"task244-sensealign/internal/model"
	"task244-sensealign/internal/service"
)

// Server 持有业务服务并注册路由。
type Server struct {
	svc *service.Service
}

// NewServer 构造 HTTP 服务。
func NewServer(svc *service.Service) *Server {
	return &Server{svc: svc}
}

// Routes 返回已注册全部路由的 mux。
func (s *Server) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/selfcheck", s.handleSelfCheck)

	mux.HandleFunc("/api/batches", s.handleBatches)
	mux.HandleFunc("/api/batches/", s.handleBatchSub)
	// Register concrete subresource patterns as well as the compatibility
	// prefix handler above. ServeMux chooses the longest matching pattern.
	mux.HandleFunc("/api/batches/{id}", s.handleBatchSub)
	mux.HandleFunc("/api/batches/{id}/entries", s.handleBatchSub)
	mux.HandleFunc("/api/batches/{id}/stats", s.handleBatchSub)
	mux.HandleFunc("/api/batches/{id}/alignments", s.handleBatchSub)
	mux.HandleFunc("/api/batches/{id}/versions", s.handleBatchSub)

	mux.HandleFunc("/api/entries/", s.handleEntrySub)
	mux.HandleFunc("/api/entries/{id}", s.handleEntrySub)
	mux.HandleFunc("/api/entries/{id}/senses", s.handleEntrySub)

	mux.HandleFunc("/api/senses/", s.handleSenseSub)
	mux.HandleFunc("/api/senses/{id}", s.handleSenseSub)
	mux.HandleFunc("/api/senses/{id}/registers", s.handleSenseSub)
	mux.HandleFunc("/api/senses/{id}/split", s.handleSenseSub)
	mux.HandleFunc("/api/senses/{id}/examples", s.handleSenseSub)
	mux.HandleFunc("/api/senses/{id}/counterexamples", s.handleSenseSub)

	mux.HandleFunc("/api/align/candidates", s.handleAlignCandidates)
	mux.HandleFunc("/api/alignments/decide", s.handleAlignDecide)
	mux.HandleFunc("/api/alignments/", s.handleAlignmentSub)
	mux.HandleFunc("/api/alignments/{id}", s.handleAlignmentSub)

	mux.HandleFunc("/api/versions", s.handleVersionsCreate)
	mux.HandleFunc("/api/versions/", s.handleVersionSub)
	mux.HandleFunc("/api/versions/{id}", s.handleVersionSub)
	mux.HandleFunc("/api/versions/{id}/freeze", s.handleVersionSub)
	mux.HandleFunc("/api/versions/{id}/share", s.handleVersionSub)
	mux.HandleFunc("/api/versions/{id}/supersede", s.handleVersionSub)

	mux.HandleFunc("/", s.handleRoot)
	return mux
}

// writeJSON 写出 JSON 响应。
func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError 将领域错误映射为 HTTP 状态码并写出。
func writeError(w http.ResponseWriter, err error) {
	code := http.StatusBadRequest
	switch {
	case errors.Is(err, model.ErrNotFound):
		code = http.StatusNotFound
	case errors.Is(err, model.ErrBatchSealed),
		errors.Is(err, model.ErrFrozenWrite),
		errors.Is(err, model.ErrSenseSplit),
		errors.Is(err, model.ErrVersionFrozen):
		code = http.StatusConflict
	case errors.Is(err, model.ErrInvalidLang),
		errors.Is(err, model.ErrEmptyDefinition),
		errors.Is(err, model.ErrSelfAlign),
		errors.Is(err, model.ErrSameEntry),
		errors.Is(err, model.ErrBadRelation),
		errors.Is(err, model.ErrDuplicate),
		errors.Is(err, model.ErrUnknownSense),
		errors.Is(err, model.ErrUnknownEntry),
		errors.Is(err, model.ErrUnknownBatch):
		code = http.StatusBadRequest
	}
	writeJSON(w, code, map[string]any{"error": err.Error()})
}

// handleHealth 健康检查。
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "service": "sensealign"})
}

// handleSelfCheck 运行端到端自检（内存临时库），返回是否通过。
func (s *Server) handleSelfCheck(w http.ResponseWriter, r *http.Request) {
	tmp, err := tempDBPath()
	if err != nil {
		writeError(w, err)
		return
	}
	if err := service.RunSelfCheck(tmp); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
