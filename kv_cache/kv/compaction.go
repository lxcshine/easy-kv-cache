package kv

import (
	"os"
	"path/filepath"
)

// Merge重写数据文件，清理被删除和被覆盖的旧数据
func (db *DB) Merge() error {
	// 在执行Merge期间，为了保证数据绝对安全，短暂阻塞写操作
	db.mu.Lock()
	defer db.mu.Lock()
	defer db.mu.Unlock()

	mergePath := filepath.Join(db.options.DirPath, "data_merge.kv")
	mergeFile, err := OpenDataFile(mergePath)
	if err != nil {
		return err
	}

	var mergeErr error

	// 核心优化：使用sync.Map专属的Range方法进行遍历
	db.index.data.Range(func(k, v interface{}) bool {
		key := k.(string)
		pos := v.(*LogRecordPos)

		// 从旧文件中读取出有效数据
		encRecord, err := db.dataFile.Read(pos.Offset, pos.Size)
		if err != nil {
			mergeErr = err
			return false // 遇到错误，返回false停止遍历
		}

		// 将有效数据追加写入到新的merge文件中
		newOffset, err := mergeFile.Write(encRecord)
		if err != nil {
			mergeErr = err
			return false
		}

		db.index.data.Store(key, &LogRecordPos{Offset: newOffset, Size: pos.Size})

		return true // 返回true继续遍历下一个key
	})

	if mergeErr != nil {
		return mergeErr
	}

	// 确保新文件的数据全部强制刷入物理硬盘
	mergeFile.Sync()

	db.dataFile.Close()
	mergeFile.Close()

	oldPath := filepath.Join(db.options.DirPath, "data.kv")
	os.Remove(oldPath)
	os.Rename(mergePath, oldPath)

	// 重新挂载合并后清爽的数据文件
	db.dataFile, _ = OpenDataFile(oldPath)
	return nil
}
