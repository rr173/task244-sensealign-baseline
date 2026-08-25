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

func TestBug05_CounterexampleBlocksConfirmation(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "bug5.db")); if err != nil { t.Fatal(err) }; defer st.DB.Close(); h:=NewServer(service.New(st)).Routes()
	var b model.Batch; call2445(t,h,http.MethodPost,"/api/batches",map[string]any{"name":"b"},&b); var e1,e2 model.Entry; call2445(t,h,http.MethodPost,"/api/batches/"+b.ID+"/entries",map[string]any{"lang_code":"en","headword":"bank"},&e1); call2445(t,h,http.MethodPost,"/api/batches/"+b.ID+"/entries",map[string]any{"lang_code":"zh","headword":"岸"},&e2)
	var s1,s2 model.Sense; call2445(t,h,http.MethodPost,"/api/entries/"+e1.ID+"/senses",map[string]any{"definition":"river bank"},&s1); call2445(t,h,http.MethodPost,"/api/entries/"+e2.ID+"/senses",map[string]any{"definition":"河岸"},&s2)
	call2445(t,h,http.MethodPost,"/api/align/candidates",map[string]any{"source_entry_id":e1.ID,"target_entry_id":e2.ID},nil); call2445(t,h,http.MethodPost,"/api/senses/"+s1.ID+"/counterexamples",map[string]any{"text":"this usage means a financial institution"},nil)
	status,_:=call2445(t,h,http.MethodPost,"/api/alignments/decide",map[string]any{"source_sense_id":s1.ID,"target_sense_id":s2.ID,"relation":"confirmed"},nil); if status == http.StatusOK { t.Fatal("confirmed alignment ignored counterexample") }
	var aligns []model.Alignment; call2445(t,h,http.MethodGet,"/api/batches/"+b.ID+"/alignments",nil,&aligns); if len(aligns)!=1 || aligns[0].Relation!=model.AlignCandidate { t.Fatalf("alignment changed after blocked confirmation: %+v",aligns) }
}
func call2445(t *testing.T,h http.Handler,m,p string,b any,o any)(int,[]byte){t.Helper();var raw []byte;if b!=nil{raw,_=json.Marshal(b)};r:=httptest.NewRecorder();q:=httptest.NewRequest(m,p,bytes.NewReader(raw));if b!=nil{q.Header.Set("Content-Type","application/json")};h.ServeHTTP(r,q);if o!=nil{if err:=json.Unmarshal(r.Body.Bytes(),o);err!=nil{t.Fatalf("decode: %v body=%s",err,r.Body.Bytes())}};return r.Code,r.Body.Bytes()}
