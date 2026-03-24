package kv

import (
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type DB struct {
	options  Options
	mu       sync.Mutex
	index    *Index
	dataFile *DataFile
}

func Open(opts Options) (*DB, error) {
	_ = os.MkdirAll(opts.DirPath, 0755)
	dataPath := filepath.Join(opts.DirPath, "data.kv")

	df, err := OpenDataFile(dataPath)
	if err != nil {
		return nil, err
	}

	db := &DB{
		options:  opts,
		index:    NewIndex(),
		dataFile: df,
	}

	if err := db.loadIndexFromDisk(); err != nil {
		return nil, err
	}

	return db, nil
}

// loadIndexFromDisk顺序扫描数据文件，重建内存索引
func (db *DB) loadIndexFromDisk() error {
	var offset int64 = 0

	for {
		// 读取9字节的Header
		headerBuf, err := db.dataFile.Read(offset, 9)
		if err != nil {
			if err == io.EOF {
				break // 文件读取完毕，正常退出
			}
			return err
		}

		// 解析Header
		recType := headerBuf[0]
		keySize := binary.LittleEndian.Uint32(headerBuf[1:5])
		valSize := binary.LittleEndian.Uint32(headerBuf[5:9])

		recordSize := 9 + keySize + valSize

		// 读取Key
		keyBuf, err := db.dataFile.Read(offset+9, keySize)
		if err != nil {
			return err
		}

		// 重建内存索引
		if recType == RecordNormal {
			db.index.Put(keyBuf, &LogRecordPos{Offset: offset, Size: recordSize})
		} else if recType == RecordDeleted {
			db.index.Delete(keyBuf)
		}

		// 游标前进到下一条记录的开头
		offset += int64(recordSize)
	}
	return nil
}

// put写入数据
func (db *DB) Put(key, value []byte) error {
	if len(key) == 0 {
		return ErrKeyIsEmpty
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	record := &LogRecord{Key: key, Value: value, Type: RecordNormal}
	encRecord := EncodeLogRecord(record)

	// 写入磁盘缓冲
	offset, err := db.dataFile.Write(encRecord)
	if err != nil {
		return err
	}

	// 更新内存索引
	db.index.Put(key, &LogRecordPos{Offset: offset, Size: uint32(len(encRecord))})
	return nil
}

// get读取数据
func (db *DB) Get(key []byte) ([]byte, error) {
	if len(key) == 0 {
		return nil, ErrKeyIsEmpty
	}

	pos := db.index.Get(key)
	if pos == nil {
		return nil, ErrKeyNotFound
	}

	encRecord, err := db.dataFile.Read(pos.Offset, pos.Size)
	if err != nil {
		return nil, err
	}

	record := DecodeLogRecord(encRecord)
	if record.Type == RecordDeleted {
		return nil, ErrKeyNotFound
	}
	return record.Value, nil
}

func (db *DB) Delete(key []byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	pos := db.index.Get(key)
	if pos == nil {
		return nil
	}

	record := &LogRecord{Key: key, Type: RecordDeleted}
	encRecord := EncodeLogRecord(record)

	if _, err := db.dataFile.Write(encRecord); err != nil {
		return err
	}

	db.index.Delete(key)
	return nil
}

func (db *DB) Sync() error {
	return db.dataFile.Sync()
}

func (db *DB) Close() error {
	if err := db.Sync(); err != nil {
		return err
	}
	return db.dataFile.Close()
}
