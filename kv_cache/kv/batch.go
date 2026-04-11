package kv

import "sync"

type WriteBatch struct {
	mu      sync.Mutex
	db      *DB
	records map[string]*LogRecord
}

func (db *DB) NewWriteBatch() *WriteBatch {
	return &WriteBatch{
		db:      db,
		records: make(map[string]*LogRecord),
	}
}

func (wb *WriteBatch) Put(key, value []byte) {
	wb.mu.Lock()
	defer wb.mu.Unlock()
	wb.records[string(key)] = &LogRecord{Key: key, Value: value, Type: RecordNormal}
}

func (wb *WriteBatch) Commit() error {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	if len(wb.records) == 0 {
		return nil
	}

	wb.db.mu.Lock()
	defer wb.db.mu.Unlock()

	for key, rec := range wb.records {
		encRecord := EncodeLogRecord(rec)
		offset, err := wb.db.dataFile.Write(encRecord)
		if err != nil {
			return err
		}
		wb.db.index.Put([]byte(key), &LogRecordPos{Offset: offset, Size: uint32(len(encRecord))})
	}

	wb.db.dataFile.Sync()
	return nil
}
