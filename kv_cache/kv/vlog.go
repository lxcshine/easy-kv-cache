package kv

import (
	"os"
	"sync"
)

type VLog struct {
	file   *os.File
	offset int64
	mu     sync.Mutex
}

const VLogThreshold = 128 * 1024

func OpenVLog(path string) (*VLog, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	stat, _ := f.Stat()
	return &VLog{file: f, offset: stat.Size()}, nil
}

func (v *VLog) Write(val []byte) (int64, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	currentOffset := v.offset
	n, err := v.file.Write(val)
	if err != nil {
		return 0, err
	}

	v.offset += int64(n)
	return currentOffset, nil
}

func (v *VLog) Read(offset int64, size uint32) ([]byte, error) {
	buf := make([]byte, size)
	_, err := v.file.ReadAt(buf, offset)
	return buf, err
}
