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

func TestBug04_DecisionCannotCrossBatchVersion(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "bug4.db")); if err != nil { t.Fatal(err) }; defer st.DB.Close(); h:=NewServer(service.New(st)).Routes()
	var b1,b2 model.Batch; call2444(t,h,http.MethodPost,"/api/batches",map[string]any{"name":"one"},&b1); call2444(t,h,http.MethodPost,"/api/batches",map[string]any{"name":"two"},&b2)
	var e1,e2 model.Entry; call2444(t,h,http.MethodPost,"/api/batches/"+b2.ID+"/entries",map[string]any{"lang_code":"en","headword":"bank"},&e1); call2444(t,h,http.MethodPost,"/api/batches/"+b2.ID+"/entries",map[string]any{"lang_code":"zh","headword":"岸"},&e2)
	var s1,s2 model.Sense; call2444(t,h,http.MethodPost,"/api/entries/"+e1.ID+"/senses",map[string]any{"definition":"river bank"},&s1); call2444(t,h,http.MethodPost,"/api/entries/"+e2.ID+"/senses",map[string]any{"definition":"河岸"},&s2)
	var v model.MappingVersion; call2444(t,h,http.MethodPost,"/api/versions",map[string]any{"batch_id":b1.ID,"name":"v1"},&v)
	status,_:=call2444(t,h,http.MethodPost,"/api/alignments/decide",map[string]any{"source_sense_id":s1.ID,"target_sense_id":s2.ID,"relation":"confirmed","version_id":v.ID},nil); if status == http.StatusOK { t.Fatal("cross-batch version decision accepted") }
	var aligns []model.Alignment; call2444(t,h,http.MethodGet,"/api/batches/"+b2.ID+"/alignments",nil,&aligns); if len(aligns)!=0 { t.Fatalf("cross-batch decision persisted: %d",len(aligns)) }
	var got model.MappingVersion; call2444(t,h,http.MethodGet,"/api/versions/"+v.ID,nil,&got); if got.Status!=model.VersionDraft { t.Fatalf("version changed: %s",got.Status) }
}
func call2444(t *testing.T,h http.Handler,m,p string,b any,o any)(int,[]byte){t.Helper();var raw []byte;if b!=nil{raw,_=json.Marshal(b)};r:=httptest.NewRecorder();q:=httptest.NewRequest(m,p,bytes.NewReader(raw));if b!=nil{q.Header.Set("Content-Type","application/json")};h.ServeHTTP(r,q);if o!=nil{if err:=json.Unmarshal(r.Body.Bytes(),o);err!=nil{t.Fatalf("decode: %v body=%s",err,r.Body.Bytes())}};return r.Code,r.Body.Bytes()}
