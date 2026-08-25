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

func TestBug08_CandidateGenerationUpdatesSenseStates(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "bug8.db")); if err != nil { t.Fatal(err) }; defer st.DB.Close(); h:=NewServer(service.New(st)).Routes()
	var b model.Batch; call2448(t,h,http.MethodPost,"/api/batches",map[string]any{"name":"b"},&b); var e1,e2 model.Entry; call2448(t,h,http.MethodPost,"/api/batches/"+b.ID+"/entries",map[string]any{"lang_code":"en","headword":"bank"},&e1); call2448(t,h,http.MethodPost,"/api/batches/"+b.ID+"/entries",map[string]any{"lang_code":"zh","headword":"岸"},&e2)
	var s1,s2,s3 model.Sense; call2448(t,h,http.MethodPost,"/api/entries/"+e1.ID+"/senses",map[string]any{"definition":"river bank"},&s1); call2448(t,h,http.MethodPost,"/api/entries/"+e2.ID+"/senses",map[string]any{"definition":"river bank"},&s2); call2448(t,h,http.MethodPost,"/api/entries/"+e2.ID+"/senses",map[string]any{"definition":"river bank"},&s3)
	call2448(t,h,http.MethodPost,"/api/align/candidates",map[string]any{"source_entry_id":e1.ID,"target_entry_id":e2.ID},nil); var got1,got2,got3 model.Sense; call2448(t,h,http.MethodGet,"/api/senses/"+s1.ID,nil,&got1); call2448(t,h,http.MethodGet,"/api/senses/"+s2.ID,nil,&got2); call2448(t,h,http.MethodGet,"/api/senses/"+s3.ID,nil,&got3)
	if got1.Status!=model.SenseConflict || got2.Status!=model.SenseAlignable || got3.Status!=model.SenseAlignable { t.Fatalf("states source=%s target1=%s target2=%s",got1.Status,got2.Status,got3.Status) }
}
func call2448(t *testing.T,h http.Handler,m,p string,b any,o any)(int,[]byte){t.Helper();var raw []byte;if b!=nil{raw,_=json.Marshal(b)};r:=httptest.NewRecorder();q:=httptest.NewRequest(m,p,bytes.NewReader(raw));if b!=nil{q.Header.Set("Content-Type","application/json")};h.ServeHTTP(r,q);if o!=nil{if err:=json.Unmarshal(r.Body.Bytes(),o);err!=nil{t.Fatalf("decode: %v body=%s",err,r.Body.Bytes())}};return r.Code,r.Body.Bytes()}
