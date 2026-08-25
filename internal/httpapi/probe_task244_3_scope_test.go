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

func TestBug03_CandidatesStayInsideBatch(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "bug3.db")); if err != nil { t.Fatal(err) }; defer st.DB.Close(); h:=NewServer(service.New(st)).Routes()
	var b1,b2 model.Batch; call2443(t,h,http.MethodPost,"/api/batches",map[string]any{"name":"one"},&b1); call2443(t,h,http.MethodPost,"/api/batches",map[string]any{"name":"two"},&b2)
	var e1,e2 model.Entry; call2443(t,h,http.MethodPost,"/api/batches/"+b1.ID+"/entries",map[string]any{"lang_code":"en","headword":"bank"},&e1); call2443(t,h,http.MethodPost,"/api/batches/"+b2.ID+"/entries",map[string]any{"lang_code":"zh","headword":"岸"},&e2)
	status,_:=call2443(t,h,http.MethodPost,"/api/align/candidates",map[string]any{"source_entry_id":e1.ID,"target_entry_id":e2.ID},nil); if status == http.StatusOK { t.Fatal("cross-batch candidates were accepted") }
	var aligns []model.Alignment; call2443(t,h,http.MethodGet,"/api/batches/"+b1.ID+"/alignments",nil,&aligns); if len(aligns)!=0 { t.Fatalf("cross-batch alignment persisted: %d",len(aligns)) }
}
func call2443(t *testing.T,h http.Handler,m,p string,b any,o any)(int,[]byte){t.Helper();var raw []byte;if b!=nil{raw,_=json.Marshal(b)};r:=httptest.NewRecorder();q:=httptest.NewRequest(m,p,bytes.NewReader(raw));if b!=nil{q.Header.Set("Content-Type","application/json")};h.ServeHTTP(r,q);if o!=nil{if err:=json.Unmarshal(r.Body.Bytes(),o);err!=nil{t.Fatalf("decode: %v body=%s",err,r.Body.Bytes())}};return r.Code,r.Body.Bytes()}
