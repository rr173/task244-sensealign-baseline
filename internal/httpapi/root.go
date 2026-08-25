package httpapi

import (
	"net/http"
)

// handleRoot 提供轻量复核页面：双栏展示批次与对齐，消费本服务 /api。
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		writeError(w, errNotFoundPath)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(rootHTML))
}

// errNotFoundPath 用于非根路径的 404。
var errNotFoundPath = &simpleErr{"not found"}

type simpleErr struct{ msg string }

func (e *simpleErr) Error() string { return e.msg }

const rootHTML = `<!doctype html>
<html lang="zh">
<head>
<meta charset="utf-8">
<title>跨语言词典义项对齐复核台</title>
<style>
body{font-family:system-ui,"PingFang SC","Microsoft YaHei",sans-serif;margin:0;background:#f5f6f8;color:#1f2933}
header{background:#1f3a5f;color:#fff;padding:14px 22px}
header h1{margin:0;font-size:18px}
main{padding:18px 22px;display:flex;gap:18px;flex-wrap:wrap}
.col{background:#fff;border:1px solid #e2e6ea;border-radius:8px;padding:14px 16px;min-width:300px;flex:1}
h2{font-size:15px;margin:0 0 10px}
pre{white-space:pre-wrap;word-break:break-word;font-size:12px;background:#f8fafc;padding:8px;border-radius:6px}
button{background:#1f3a5f;color:#fff;border:0;border-radius:6px;padding:6px 12px;cursor:pointer;margin:2px}
input{padding:5px 8px;border:1px solid #cbd2d9;border-radius:6px;width:200px}
</style>
</head>
<body>
<header><h1>跨语言词典义项对齐复核台</h1></header>
<main>
  <section class="col">
    <h2>批次</h2>
    <div><input id="bname" placeholder="批次名称"><button onclick="createBatch()">新建批次</button></div>
    <pre id="batches">加载中…</pre>
  </section>
  <section class="col">
    <h2>对齐与统计</h2>
    <div><input id="bid" placeholder="批次ID"><button onclick="loadBatch()">加载</button></div>
    <pre id="detail">选择批次后加载对齐与统计。</pre>
  </section>
</main>
<script>
async function api(path, opts){const r=await fetch(path, opts);return r.json();}
async function refreshBatches(){const bs=await api('/api/batches');document.getElementById('batches').textContent=JSON.stringify(bs,null,2);}
async function createBatch(){const name=document.getElementById('bname').value||'未命名';await api('/api/batches',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({name,description:''})});refreshBatches();}
async function loadBatch(){const id=document.getElementById('bid').value;if(!id)return;const [als,st]=await Promise.all([api('/api/batches/'+id+'/alignments'),api('/api/batches/'+id+'/stats')]);document.getElementById('detail').textContent=JSON.stringify({stats:st,alignments:als},null,2);}
refreshBatches();
</script>
</body>
</html>`
