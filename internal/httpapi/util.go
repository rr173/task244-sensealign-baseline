package httpapi

import (
	"encoding/json"
	"net/http"
	"os"
)

// tempDBPath 创建用于自检的临时 SQLite 文件路径。
func tempDBPath() (string, error) {
	f, err := os.CreateTemp("", "sensealign-selfcheck-*.db")
	if err != nil {
		return "", err
	}
	path := f.Name()
	_ = f.Close()
	_ = os.Remove(path) // 让 store.Open 自行创建
	return path, nil
}

// decodeBody 解析请求体为 v，失败返回错误。
func decodeBody(r *http.Request, v any) error {
	defer func() { _ = r.Body.Close() }()
	dec := json.NewDecoder(r.Body)
	return dec.Decode(v)
}
