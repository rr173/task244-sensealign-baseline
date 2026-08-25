// Package model 定义跨语言词典义项对齐复核台的核心实体、状态枚举与领域错误。
//
// 业务域：词典编纂者复核两个翻译词条（例如一个外语词目对应多个汉语义项）究竟是同一义项、
// 部分重合，还是被过度合并。核心实体为义项（sense）、例句（example）、语域标签（register）、
// 对齐关系（alignment）与映射版本（mapping version）。状态机描述的是语义裁决与版本冻结，
// 不是 OA/工单/任务流转。
package model

import "errors"

// 领域错误。HTTP 层据此映射为 4xx。
var (
	ErrNotFound               = errors.New("resource not found")
	ErrBatchSealed            = errors.New("batch is sealed and cannot be modified")
	ErrBatchNotAligning       = errors.New("batch is not in aligning state")
	ErrSenseSplit             = errors.New("sense already split")
	ErrFrozenWrite            = errors.New("mapping version is frozen and rejects writes")
	ErrDuplicate              = errors.New("duplicate resource")
	ErrInvalidLang            = errors.New("invalid or missing language code")
	ErrSelfAlign              = errors.New("a sense cannot align to itself")
	ErrRegisterConflict       = errors.New("register tags conflict for this alignment")
	ErrUnknownSense           = errors.New("referenced sense does not exist")
	ErrUnknownEntry           = errors.New("referenced entry does not exist")
	ErrUnknownBatch           = errors.New("referenced batch does not exist")
	ErrVersionFrozen          = errors.New("cannot modify a frozen mapping version")
	ErrEmptyDefinition        = errors.New("sense definition must not be empty")
	ErrSameEntry              = errors.New("alignment requires two different entries")
	ErrBadRelation            = errors.New("invalid alignment relation")
	ErrInvalidBatchTransition = errors.New("invalid batch status transition")
	ErrCrossBatch             = errors.New("cross-batch alignment is not allowed")
	ErrCounterexampleConflict = errors.New("counterexample blocks confirmation")
	ErrDuplicateTarget        = errors.New("duplicate split target")
)
