// Package store 负责 SQLite 持久化：建表迁移、ID 生成与跨实体读写。
// 使用 modernc.org/sqlite（纯 Go、CGO 无关），单写者模式保证状态机一致。
package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Store 持有数据库连接，按实体在多个文件上以方法扩展。
type Store struct {
	DB *sql.DB
}

// Open 打开（必要时创建）SQLite 数据库并完成迁移。
// DSN 启用 WAL、忙等待与外键约束；SetMaxOpenConns(1) 保证单写者串行化。
func Open(dbPath string) (*Store, error) {
	dsn := fmt.Sprintf(
		"file:%s?_pragma=busy_timeout(8000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=synchronous(NORMAL)",
		dbPath,
	)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	if err := Migrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &Store{DB: db}, nil
}

// Migrate 创建全部表与唯一索引（幂等）。
func Migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS batches (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS entries (
			id TEXT PRIMARY KEY,
			batch_id TEXT NOT NULL,
			lang_code TEXT NOT NULL,
			headword TEXT NOT NULL,
			gloss TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			FOREIGN KEY (batch_id) REFERENCES batches(id)
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_entries_batch_headword ON entries(batch_id, lang_code, headword)`,
		`CREATE TABLE IF NOT EXISTS senses (
			id TEXT PRIMARY KEY,
			entry_id TEXT NOT NULL,
			definition TEXT NOT NULL,
			status TEXT NOT NULL,
			register_tags TEXT NOT NULL DEFAULT '[]',
			created_at TEXT NOT NULL,
			FOREIGN KEY (entry_id) REFERENCES entries(id)
		)`,
		`CREATE TABLE IF NOT EXISTS examples (
			id TEXT PRIMARY KEY,
			sense_id TEXT NOT NULL,
			text TEXT NOT NULL,
			lang_code TEXT NOT NULL DEFAULT '',
			translation TEXT NOT NULL DEFAULT '',
			register TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			FOREIGN KEY (sense_id) REFERENCES senses(id)
		)`,
		`CREATE TABLE IF NOT EXISTS alignments (
			id TEXT PRIMARY KEY,
			source_sense_id TEXT NOT NULL,
			target_sense_id TEXT NOT NULL,
			relation TEXT NOT NULL,
			coverage_score REAL NOT NULL DEFAULT 0,
			hypernym INTEGER NOT NULL DEFAULT 0,
			hyponym INTEGER NOT NULL DEFAULT 0,
			register_conflict INTEGER NOT NULL DEFAULT 0,
			reason TEXT NOT NULL DEFAULT '',
			version_id TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			decided_at TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_align_pair ON alignments(source_sense_id, target_sense_id)`,
		`CREATE TABLE IF NOT EXISTS counterexamples (
			id TEXT PRIMARY KEY,
			sense_id TEXT NOT NULL,
			text TEXT NOT NULL,
			note TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			FOREIGN KEY (sense_id) REFERENCES senses(id)
		)`,
		`CREATE TABLE IF NOT EXISTS mapping_versions (
			id TEXT PRIMARY KEY,
			batch_id TEXT NOT NULL,
			name TEXT NOT NULL,
			status TEXT NOT NULL,
			snapshot TEXT NOT NULL DEFAULT '',
			frozen_at TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			FOREIGN KEY (batch_id) REFERENCES batches(id)
		)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("exec migrate stmt: %w", err)
		}
	}
	return nil
}

// NewID 生成带前缀的随机 ID（前缀用于区分实体种类，便于阅读与调试）。
func NewID(prefix string) string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		// 极不可能失败；退化为时间戳兜底。
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(buf)
}

// NowRFC 返回当前时间（东八区）的 RFC3339 字符串。
func NowRFC() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// ParseTime 解析存储的时间字符串，失败返回零值。
func ParseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
