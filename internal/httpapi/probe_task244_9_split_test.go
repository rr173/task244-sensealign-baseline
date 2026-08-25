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

func TestBug09_SplitRejectsDuplicateTargets(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "bug9.db")); if err != nil { t.Fatal(err) }; defer st.DB.Close(); h:=NewServer(service.New(st)).Routes()
	var b model.Batch; call2449(t,h,http.MethodPost,"/api/batches",map[string]any{"name":"b"},&b); var e model.Entry; call2449(t,h,http.MethodPost,"/api/batches/"+b.ID+"/entries",map[string]any{"lang_code":"en","headword":"bank"},&e)
	var poly,target model.Sense; call2449(t,h,http.MethodPost,"/api/entries/"+e.ID+"/senses",map[string]any{"definition":"shared"},&poly); call2449(t,h,http.MethodPost,"/api/entries/"+e.ID+"/senses",map[string]any{"definition":"bank"},&target)
	status,_:=call2449(t,h,http.MethodPost,"/api/senses/"+poly.ID+"/split",map[string]any{"target_sense_ids":[]string{target.ID,target.ID}},nil); if status == http.StatusOK { t.Fatal("duplicate split targets accepted") }
	var senses []model.Sense; call2449(t,h,http.MethodGet,"/api/entries/"+e.ID+"/senses",nil,&senses); var got model.Sense; call2449(t,h,http.MethodGet,"/api/senses/"+poly.ID,nil,&got); if len(senses)!=2 || got.Status!=model.SensePending { t.Fatalf("partial split persisted: senses=%d original=%s",len(senses),got.Status) }
}
func call2449(t *testing.T,h http.Handler,m,p string,b any,o any)(int,[]byte){t.Helper();var raw []byte;if b!=nil{raw,_=json.Marshal(b)};r:=httptest.NewRecorder();q:=httptest.NewRequest(m,p,bytes.NewReader(raw));if b!=nil{q.Header.Set("Content-Type","application/json")};h.ServeHTTP(r,q);if o!=nil{if err:=json.Unmarshal(r.Body.Bytes(),o);err!=nil{t.Fatalf("decode: %v body=%s",err,r.Body.Bytes())}};return r.Code,r.Body.Bytes()}
