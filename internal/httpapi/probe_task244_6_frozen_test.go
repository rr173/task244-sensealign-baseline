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

func TestBug06_FrozenVersionCannotBeSuperseded(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "bug6.db")); if err != nil { t.Fatal(err) }; defer st.DB.Close(); h:=NewServer(service.New(st)).Routes()
	var b model.Batch; call2446(t,h,http.MethodPost,"/api/batches",map[string]any{"name":"b"},&b); var v model.MappingVersion; call2446(t,h,http.MethodPost,"/api/versions",map[string]any{"batch_id":b.ID,"name":"v1"},&v)
	status,_:=call2446(t,h,http.MethodPost,"/api/versions/"+v.ID+"/freeze",nil,nil); if status!=http.StatusOK { t.Fatalf("freeze status=%d",status) }
	status,_=call2446(t,h,http.MethodPost,"/api/versions/"+v.ID+"/supersede",nil,nil); if status==http.StatusOK { t.Fatal("frozen version was superseded") }
	var got model.MappingVersion; call2446(t,h,http.MethodGet,"/api/versions/"+v.ID,nil,&got); if got.Status!=model.VersionFrozen { t.Fatalf("frozen status changed to %s",got.Status) }
}
func call2446(t *testing.T,h http.Handler,m,p string,b any,o any)(int,[]byte){t.Helper();var raw []byte;if b!=nil{raw,_=json.Marshal(b)};r:=httptest.NewRecorder();q:=httptest.NewRequest(m,p,bytes.NewReader(raw));if b!=nil{q.Header.Set("Content-Type","application/json")};h.ServeHTTP(r,q);if o!=nil{if err:=json.Unmarshal(r.Body.Bytes(),o);err!=nil{t.Fatalf("decode: %v body=%s",err,r.Body.Bytes())}};return r.Code,r.Body.Bytes()}
