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

func TestBug07_RejectedAlignmentLeavesNoConflict(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "bug7.db")); if err != nil { t.Fatal(err) }; defer st.DB.Close(); h:=NewServer(service.New(st)).Routes()
	var b model.Batch; call2447(t,h,http.MethodPost,"/api/batches",map[string]any{"name":"b"},&b); var e1,e2 model.Entry; call2447(t,h,http.MethodPost,"/api/batches/"+b.ID+"/entries",map[string]any{"lang_code":"en","headword":"bank"},&e1); call2447(t,h,http.MethodPost,"/api/batches/"+b.ID+"/entries",map[string]any{"lang_code":"zh","headword":"岸"},&e2)
	var s1,s2,s3 model.Sense; call2447(t,h,http.MethodPost,"/api/entries/"+e1.ID+"/senses",map[string]any{"definition":"river bank"},&s1); call2447(t,h,http.MethodPost,"/api/entries/"+e2.ID+"/senses",map[string]any{"definition":"river bank"},&s2); call2447(t,h,http.MethodPost,"/api/entries/"+e2.ID+"/senses",map[string]any{"definition":"river bank"},&s3)
	call2447(t,h,http.MethodPost,"/api/align/candidates",map[string]any{"source_entry_id":e1.ID,"target_entry_id":e2.ID},nil); for _, target:=range []string{s2.ID,s3.ID} { call2447(t,h,http.MethodPost,"/api/alignments/decide",map[string]any{"source_sense_id":s1.ID,"target_sense_id":target,"relation":"rejected"},nil) }
	var stats map[string]any; call2447(t,h,http.MethodGet,"/api/batches/"+b.ID+"/stats",nil,&stats); if n,ok:=stats["one_to_many_conflicts"].(float64); !ok || n!=0 { t.Fatalf("one-to-many conflict count=%v",stats["one_to_many_conflicts"]) }
}
func call2447(t *testing.T,h http.Handler,m,p string,b any,o any)(int,[]byte){t.Helper();var raw []byte;if b!=nil{raw,_=json.Marshal(b)};r:=httptest.NewRecorder();q:=httptest.NewRequest(m,p,bytes.NewReader(raw));if b!=nil{q.Header.Set("Content-Type","application/json")};h.ServeHTTP(r,q);if o!=nil{if err:=json.Unmarshal(r.Body.Bytes(),o);err!=nil{t.Fatalf("decode: %v body=%s",err,r.Body.Bytes())}};return r.Code,r.Body.Bytes()}
