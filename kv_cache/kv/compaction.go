package kv

import (
	"os"
	"path/filepath"
)

func (db *DB) Merge() error {

	db.mu.Lock()
	defer db.mu.Lock()
	defer db.mu.Unlock()

	mergePath := filepath.Join(db.options.DirPath, "data_merge.kv")
	mergeFile, err := OpenDataFile(mergePath)
	if err != nil {
		return err
	}

	var mergeErr error

	db.index.data.Range(func(k, v interface{}) bool {
		key := k.(string)
		pos := v.(*LogRecordPos)

		encRecord, err := db.dataFile.Read(pos.Offset, pos.Size)
		if err != nil {
			mergeErr = err
			return false
		}

		newOffset, err := mergeFile.Write(encRecord)
		if err != nil {
			mergeErr = err
			return false
		}

		db.index.data.Store(key, &LogRecordPos{Offset: newOffset, Size: pos.Size})

		return true
	})

	if mergeErr != nil {
		return mergeErr
	}

	mergeFile.Sync()

	db.dataFile.Close()
	mergeFile.Close()

	oldPath := filepath.Join(db.options.DirPath, "data.kv")
	os.Remove(oldPath)
	os.Rename(mergePath, oldPath)

	db.dataFile, _ = OpenDataFile(oldPath)
	return nil
}
