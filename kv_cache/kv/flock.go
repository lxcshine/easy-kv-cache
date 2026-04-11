package kv

import (
	"fmt"
	"github.com/gofrs/flock"
)

type FileLock struct {
	lock *flock.Flock
}

func AcquireFileLock(dirPath string) (*FileLock, error) {
	lockPath := dirPath + "/LOCK"

	fileLock := flock.New(lockPath)

	locked, err := fileLock.TryLock()
	if err != nil {
		return nil, fmt.Errorf("尝试获取数据库锁时发生系统错误: %v", err)
	}

	if !locked {
		return nil, fmt.Errorf("数据库已被其他进程独占")
	}

	return &FileLock{lock: fileLock}, nil
}

func (fl *FileLock) Release() error {
	if fl.lock != nil {
		return fl.lock.Unlock()
	}
	return nil
}
