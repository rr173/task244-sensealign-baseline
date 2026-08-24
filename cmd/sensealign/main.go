// Command sensealign 是跨语言词典义项对齐复核台的可执行入口。
//
// 常用调用：
//   sensealign --smoke-test            运行端到端自检（建库→对齐→冻结→关闭重开验证）后以 0 退出
//   sensealign --addr :8080 --db app.db 启动长驻 HTTP 服务
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"task244-sensealign/internal/httpapi"
	"task244-sensealign/internal/service"
	"task244-sensealign/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP 监听地址")
	dbPath := flag.String("db", "sensealign.db", "SQLite 数据库文件路径")
	smoke := flag.Bool("smoke-test", false, "运行端到端自检后退出（不启动长驻服务）")
	flag.Parse()

	if *smoke {
		if err := service.RunSelfCheck(*dbPath); err != nil {
			fmt.Fprintf(os.Stderr, "smoke-test FAILED: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("smoke-test OK")
		os.Exit(0)
	}

	st, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer func() { _ = st.DB.Close() }()

	svc := service.New(st)
	srv := httpapi.NewServer(svc)
	fmt.Printf("listening on %s (db=%s)\n", *addr, *dbPath)
	if err := http.ListenAndServe(*addr, srv.Routes()); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
