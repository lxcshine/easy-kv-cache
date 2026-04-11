package kv

import (
	"encoding/binary"
	"io"
	"path/filepath"
	"sync"
)

type DB struct {
	options  Options
	mu       sync.Mutex
	index    *Index
	dataFile *DataFile

	vlog     *VLog
	cache    *LRUCache
	bloom    *BloomFilter
	fileLock *FileLock
}

func Open(opts Options) (*DB, error) {
	flock, err := AcquireFileLock(opts.DirPath)
	if err != nil {
		return nil, err
	}

	df, _ := OpenDataFile(filepath.Join(opts.DirPath, "data.kv"))
	vl, _ := OpenVLog(filepath.Join(opts.DirPath, "vlog.kv"))

	db := &DB{
		options:  opts,
		index:    NewIndex(),
		dataFile: df,
		vlog:     vl,
		cache:    NewLRUCache(opts.CacheCapacity),
		bloom:    NewBloomFilter(opts.BloomSize),
		fileLock: flock,
	}

	if err := db.loadIndexFromDisk(); err != nil {
		return nil, err
	}

	return db, nil
}

func (db *DB) loadIndexFromDisk() error {
	var offset int64 = 0

	for {
		headerBuf, err := db.dataFile.Read(offset, 9)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		recType := headerBuf[0]
		keySize := binary.LittleEndian.Uint32(headerBuf[1:5])
		valSize := binary.LittleEndian.Uint32(headerBuf[5:9])
		recordSize := 9 + keySize + valSize

		keyBuf, err := db.dataFile.Read(offset+9, keySize)
		if err != nil {
			return err
		}

		if recType == RecordNormal {

			db.index.Put(keyBuf, &LogRecordPos{Offset: offset, Size: recordSize, IsInVLog: false})
			db.bloom.Add(keyBuf)

		} else if recType == RecordVLog {

			valBuf, _ := db.dataFile.Read(offset+9+int64(keySize), valSize)

			vlogOffset := int64(binary.LittleEndian.Uint64(valBuf[:8]))
			vlogSize := binary.LittleEndian.Uint32(valBuf[8:])

			db.index.Put(keyBuf, &LogRecordPos{Offset: vlogOffset, Size: vlogSize, IsInVLog: true})
			db.bloom.Add(keyBuf)

		} else if recType == RecordDeleted {
			db.index.Delete(keyBuf)
		}

		offset += int64(recordSize)
	}
	return nil
}

func (db *DB) Put(key, value []byte) error {
	if len(key) == 0 {
		return ErrKeyIsEmpty
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	if int64(len(value)) > db.options.ValueThreshold {

		vlogOffset, err := db.vlog.Write(value)
		if err != nil {
			return err
		}

		ptrBuf := make([]byte, 12)
		binary.LittleEndian.PutUint64(ptrBuf[:8], uint64(vlogOffset))
		binary.LittleEndian.PutUint32(ptrBuf[8:], uint32(len(value)))

		enc := EncodeLogRecord(&LogRecord{Key: key, Value: ptrBuf, Type: RecordVLog})
		_, err = db.dataFile.Write(enc)
		if err != nil {
			return err
		}

		db.index.Put(key, &LogRecordPos{Offset: vlogOffset, Size: uint32(len(value)), IsInVLog: true})

	} else {

		enc := EncodeLogRecord(&LogRecord{Key: key, Value: value, Type: RecordNormal})

		offset, err := db.dataFile.Write(enc)
		if err != nil {
			return err
		}

		db.index.Put(key, &LogRecordPos{Offset: offset, Size: uint32(len(enc)), IsInVLog: false})
	}

	db.bloom.Add(key)
	db.cache.Put(string(key), value)
	return nil
}

func (db *DB) Get(key []byte) ([]byte, error) {

	if !db.bloom.MightContain(key) {
		return nil, ErrKeyNotFound
	}

	if val, hit := db.cache.Get(string(key)); hit {
		return val, nil
	}

	pos := db.index.Get(key)
	if pos == nil {
		return nil, ErrKeyNotFound
	}

	var valBytes []byte
	var err error // 这里的 err 是必需的，用于接收两个分支可能抛出的异常

	if pos.IsInVLog {
		valBytes, err = db.vlog.Read(pos.Offset, pos.Size)
	} else {
		var encRecord []byte
		encRecord, err = db.dataFile.Read(pos.Offset, pos.Size)
		if err == nil {
			valBytes = DecodeLogRecord(encRecord).Value
		}
	}

	if err != nil {
		return nil, err
	}

	// 回填缓存
	db.cache.Put(string(key), valBytes)
	return valBytes, nil
}

func (db *DB) Delete(key []byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	pos := db.index.Get(key)
	if pos == nil {
		return nil
	}

	encRecord := EncodeLogRecord(&LogRecord{Key: key, Type: RecordDeleted})
	if _, err := db.dataFile.Write(encRecord); err != nil {
		return err
	}

	db.index.Delete(key)
	return nil
}

func (db *DB) Sync() error {
	_ = db.vlog.file.Sync()
	return db.dataFile.Sync()
}

func (db *DB) Close() error {
	_ = db.Sync()
	db.dataFile.Close()
	db.vlog.file.Close()
	return db.fileLock.Release() // 释放进程排他锁
}
