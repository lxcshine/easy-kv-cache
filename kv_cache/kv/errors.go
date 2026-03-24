package kv

import "errors"

var (
	ErrKeyNotFound = errors.New("key not found in database")
	ErrKeyIsEmpty  = errors.New("key cannot be empty")
	ErrMergeFailed = errors.New("merge/compaction failed")
)
