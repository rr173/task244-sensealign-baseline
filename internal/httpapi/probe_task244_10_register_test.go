package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"task244-sensealign/internal/model"
	"task244-sensealign/internal/service"
	"task244-sensealign/internal/store"
)

func TestBug10_RegisterChangeRefreshesAlignmentEvidence(t *testing.T) {
	dbPath:=filepath.Join(t.TempDir(),"bug10.db"); st,err:=store.Open(dbPath); if err!=nil{t.Fatal(err)}; h:=NewServer(service.New(st)).Routes()
	var b model.Batch; call24410(t,h,http.MethodPost,"/api/batches",map[string]any{"name":"b"},&b); var e1,e2 model.Entry; call24410(t,h,http.MethodPost,"/api/batches/"+b.ID+"/entries",map[string]any{"lang_code":"en","headword":"bank"},&e1); call24410(t,h,http.MethodPost,"/api/batches/"+b.ID+"/entries",map[string]any{"lang_code":"zh","headword":"岸"},&e2)
	var s1,s2 model.Sense; call24410(t,h,http.MethodPost,"/api/entries/"+e1.ID+"/senses",map[string]any{"definition":"river bank","register_tags":[]string{"formal"}},&s1); call24410(t,h,http.MethodPost,"/api/entries/"+e2.ID+"/senses",map[string]any{"definition":"河岸","register_tags":[]string{"slang"}},&s2); call24410(t,h,http.MethodPost,"/api/align/candidates",map[string]any{"source_entry_id":e1.ID,"target_entry_id":e2.ID},nil)
	var stats map[string]any; call24410(t,h,http.MethodGet,"/api/batches/"+b.ID+"/stats",nil,&stats); if stats["register_conflicts"] != float64(1) { t.Fatalf("initial conflict=%v",stats["register_conflicts"]) }
	call24410(t,h,http.MethodPut,"/api/senses/"+s2.ID+"/registers",map[string]any{"register_tags":[]string{"formal"}},nil); call24410(t,h,http.MethodGet,"/api/batches/"+b.ID+"/stats",nil,&stats); if stats["register_conflicts"] != float64(0) { t.Fatalf("stale conflict after update=%v",stats["register_conflicts"]) }
	if err:=st.DB.Close();err!=nil{t.Fatal(err)}; st2,err:=store.Open(dbPath);if err!=nil{t.Fatal(err)};defer st2.DB.Close(); h2:=NewServer(service.New(st2)).Routes(); call24410(t,h2,http.MethodGet,"/api/batches/"+b.ID+"/stats",nil,&stats); if stats["register_conflicts"] != float64(0) { t.Fatalf("conflict returned after reopen=%v",stats["register_conflicts"]) }
}
func call24410(t *testing.T,h http.Handler,m,p string,b any,o any)(int,[]byte){t.Helper();var raw []byte;if b!=nil{raw,_=json.Marshal(b)};r:=httptest.NewRecorder();q:=httptest.NewRequest(m,p,bytes.NewReader(raw));if b!=nil{q.Header.Set("Content-Type","application/json")};h.ServeHTTP(r,q);if o!=nil{if err:=json.Unmarshal(r.Body.Bytes(),o);err!=nil{t.Fatalf("decode: %v body=%s",err,r.Body.Bytes())}};return r.Code,r.Body.Bytes()}
