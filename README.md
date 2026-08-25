# 跨语言词典义项对齐复核台（task244-sensealign）

面向词典编纂者的后端服务：导入双语词条、义项、例句与语域标签，依据**例句覆盖度、语域相容性、
上位/下位关系**自动生成候选义项对齐，由人工裁决（确认 / 部分重合 / 否决）并冻结为不可变映射版本。

## 业务闭环

1. 创建词典批次，导入两个跨语言词条（如英语 `bank` 与汉语「银行 / 岸」）。
2. 为每个词条登记义项、例句与语域标签。
3. 调用候选生成，得到义项两两配对的覆盖度、语域冲突与上位/下位标记。
4. 一对多冲突（多义词）可拆分义项；存在反例可否决对齐。
5. 裁决对齐后创建映射版本并冻结，发布可引用的不可变快照。

## 核心状态机

- 词典批次：`organizing → aligning → published → sealed`
- 义项：`pending → alignable / conflict / split`
- 对齐关系：`candidate → partial / confirmed / rejected`
- 映射版本：`draft → shared / frozen → superseded`

## 关键不变量

- 对齐关系 `(source_sense_id, target_sense_id)` 唯一，落盘即覆盖更新。
- 已冻结映射版本拒绝写入（裁决若绑定冻结版本返回冲突）。
- 已封存批次拒绝新增词条；已拆分义项不可再次拆分。
- 语域标签受控（formal/informal/slang/technical/literary/colloquial/archaic/dialect），入库前规范化。

## 目录结构

```
env/
├── cmd/sensealign/main.go      # 入口（--addr / --db / --smoke-test）
├── internal/
│   ├── model/                  # 实体与领域错误
│   ├── store/                  # SQLite 持久化与迁移
│   ├── evidence/               # 例句分词、覆盖度、语域约束
│   ├── matcher/                # 候选对齐生成、上位/下位、一对多
│   ├── adjudicate/             # 多义词拆分、版本冻结、冲突报告
│   ├── service/                # 业务编排与自检
│   └── httpapi/                # /api JSON 接口与复核页
├── component-versions.json
├── Dockerfile / benzhi.Dockerfile / build_benzhi_docker.sh
└── BENZHI_README.md
```

## 标准命令

```bash
cd env
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go test ./...
./cmd/sensealign/sensealign --smoke-test
./cmd/sensealign/sensealign --addr :8080 --db app.db
```
