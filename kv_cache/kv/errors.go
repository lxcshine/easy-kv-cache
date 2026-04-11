package kv

import "errors"

var (
	ErrKeyNotFound      = errors.New("key not found in database")
	ErrKeyIsEmpty       = errors.New("key cannot be empty")
	ErrMergeFailed      = errors.New("merge/compaction failed")
	ErrDatabaseIsLocked = errors.New("数据库已被其他进程独占")
	ErrChecksumMismatch = errors.New("数据损坏：CRC32校验和不匹配")
)
