package model

import "time"

// BatchStatus 词典批次状态机：整理中 → 待对齐 → 已发布 → 封存。
type BatchStatus string

const (
	BatchOrganizing BatchStatus = "organizing"
	BatchAligning   BatchStatus = "aligning"
	BatchPublished  BatchStatus = "published"
	BatchSealed     BatchStatus = "sealed"
)

// SenseStatus 义项状态机：待校验 → 可对齐 / 冲突 / 已拆分。
type SenseStatus string

const (
	SensePending   SenseStatus = "pending"
	SenseAlignable SenseStatus = "alignable"
	SenseConflict  SenseStatus = "conflict"
	SenseSplit     SenseStatus = "split"
)

// AlignRelation 对齐关系：候选 / 部分重合 / 确认 / 否决。
type AlignRelation string

const (
	AlignCandidate AlignRelation = "candidate"
	AlignPartial   AlignRelation = "partial"
	AlignConfirmed AlignRelation = "confirmed"
	AlignRejected  AlignRelation = "rejected"
)

// VersionStatus 映射版本状态机：草稿 → 共享 / 冻结 → 替代。
type VersionStatus string

const (
	VersionDraft      VersionStatus = "draft"
	VersionShared     VersionStatus = "shared"
	VersionFrozen     VersionStatus = "frozen"
	VersionSuperseded VersionStatus = "superseded"
)

// Batch 词典批次：一次对齐复核工作单元，归集若干跨语言词条。
type Batch struct {
	ID          string
	Name        string
	Description string
	Status      BatchStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Entry 词条：某一语言下的词目，隶属于一个批次。
type Entry struct {
	ID        string
	BatchID   string
	LangCode  string
	Headword  string
	Gloss     string
	CreatedAt time.Time
}

// Sense 义项：词条的一个含义，是跨语言对齐的最小裁决单元。
type Sense struct {
	ID          string
	EntryID     string
	Definition  string
	Status      SenseStatus
	RegisterTags []string // 语域标签：formal/informal/slang/technical/literary/colloquial/archaic/dialect
	CreatedAt   time.Time
}

// Example 例句：义项的用法证据，可带翻译与语域。
type Example struct {
	ID          string
	SenseID     string
	Text        string
	LangCode    string
	Translation string
	Register    string
	CreatedAt   time.Time
}

// Alignment 对齐关系：一条跨语言义项配对及其裁决结论。
type Alignment struct {
	ID              string
	SourceSenseID   string
	TargetSenseID   string
	Relation        AlignRelation
	CovScore        float64
	Hypernym        bool
	Hyponym         bool
	RegisterConflict bool
	Reason          string
	VersionID       string // 空表示未纳入任何版本
	CreatedAt       time.Time
	DecidedAt       time.Time
}

// Counterexample 反例：证明某义项配对不应合并的证据。
type Counterexample struct {
	ID        string
	SenseID   string
	Text      string
	Note      string
	CreatedAt time.Time
}

// MappingVersion 映射版本：某一时刻对齐结论的不可变快照。
type MappingVersion struct {
	ID        string
	BatchID   string
	Name      string
	Status    VersionStatus
	Snapshot  string // JSON 快照
	FrozenAt  time.Time
	CreatedAt time.Time
}

// Candidate 由 matcher 生成的候选对齐（尚未经人工裁决）。
type Candidate struct {
	SourceSenseID    string
	TargetSenseID    string
	CovScore         float64
	Hypernym         bool
	Hyponym          bool
	RegisterConflict bool
}

// ValidBatchStatus 校验批次状态取值。
func ValidBatchStatus(s string) bool {
	switch BatchStatus(s) {
	case BatchOrganizing, BatchAligning, BatchPublished, BatchSealed:
		return true
	}
	return false
}

// ValidSenseStatus 校验义项状态取值。
func ValidSenseStatus(s string) bool {
	switch SenseStatus(s) {
	case SensePending, SenseAlignable, SenseConflict, SenseSplit:
		return true
	}
	return false
}

// ValidAlignRelation 校验对齐关系取值。
func ValidAlignRelation(s string) bool {
	switch AlignRelation(s) {
	case AlignCandidate, AlignPartial, AlignConfirmed, AlignRejected:
		return true
	}
	return false
}

// ValidVersionStatus 校验版本状态取值。
func ValidVersionStatus(s string) bool {
	switch VersionStatus(s) {
	case VersionDraft, VersionShared, VersionFrozen, VersionSuperseded:
		return true
	}
	return false
}
