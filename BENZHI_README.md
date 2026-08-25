基于 Go 实现的跨语言词典义项对齐复核 Web 项目，一款后端服务，完成义项候选对齐生成、语域与上位/下位冲突裁决与映射版本冻结发布。

# BENZHI 评测说明

本说明供评测构建与复验使用。项目类型为「跨语言词典义项对齐复核台」，词典编纂者导入双语词条与义项、
例句、语域标签，由系统依据例句覆盖度、语域相容性与上位/下位关系生成候选对齐，人工裁决后冻结为不可变映射版本。

## 环境

- Go：1.26.3（`GOTOOLCHAIN=local`，`CGO_ENABLED=0`）
- SQLite：`modernc.org/sqlite`（纯 Go、CGO 无关），版本 v1.52.0，组件版本 3.46.1
- 代理：`GOPROXY=https://goproxy.cn,direct`，`GOSUMDB=sum.golang.google.cn`

## 构建与运行

```bash
cd env
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go vet ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go test ./...
./cmd/sensealign/sensealign --smoke-test      # 端到端自检，成功以 0 退出
./cmd/sensealign/sensealign --addr :8080 --db app.db   # 长驻服务
```

## 门禁契约（--smoke-test）

`--smoke-test` 在临时 SQLite 上执行一次完整闭环：新建批次 → 导入双语多义词条与例句 →
生成候选对齐 → 裁决两条确认对齐 → 创建并冻结映射版本 → **关闭并重新打开数据库**验证持久化与重启恢复，
全部通过打印 `smoke-test OK` 并以退出码 0 结束；任一断言失败以非零退出码结束。

## HTTP API（前缀 /api）

- `GET /api/health` 健康检查
- `POST/GET /api/batches` 新建 / 列表批次
- `GET/PUT /api/batches/{id}` 读取批次 / 更新状态
- `POST/GET /api/batches/{id}/entries` 新增 / 列表词条
- `GET /api/batches/{id}/stats` 批次统计与冲突计数
- `GET /api/batches/{id}/alignments` 批次对齐列表
- `GET /api/batches/{id}/versions` 批次版本列表
- `GET /api/entries/{id}` 读取词条
- `POST/GET /api/entries/{id}/senses` 新增 / 列表义项
- `GET/PUT /api/senses/{id}` / `registers` 读取义项 / 设置语域标签
- `POST /api/senses/{id}/split` 拆分多义词
- `POST/GET /api/senses/{id}/examples` 新增 / 列表例句
- `POST/GET /api/senses/{id}/counterexamples` 新增 / 列表反例
- `POST /api/align/candidates` 生成候选对齐
- `POST /api/alignments/decide` 裁决对齐（确认/部分重合/否决）
- `GET /api/alignments/{id}` 读取单条对齐
- `POST /api/versions` 新建映射版本
- `GET /api/versions/{id}` 读取版本
- `POST /api/versions/{id}/freeze|share|supersede` 冻结 / 共享 / 替代版本
- `GET /api/selfcheck` 在线端到端自检
- `GET /` 轻量复核页面（双栏展示批次与对齐）

## 持久化与重启恢复

所有实体写入 SQLite（WAL），单写者串行化。服务重启后从既有行恢复批次、义项、对齐与版本；
冻结版本内容不可变，已裁决对齐纳入版本后随版本冻结拒绝改写。
