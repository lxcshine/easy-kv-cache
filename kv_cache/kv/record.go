package kv

import "encoding/binary"

const (
	RecordNormal byte = iota
	RecordDeleted
	RecordVLog
)

type LogRecord struct {
	Key   []byte
	Value []byte
	Type  byte
}

type LogRecordPos struct {
	Offset   int64
	Size     uint32
	IsInVLog bool
}

func EncodeLogRecord(rec *LogRecord) []byte {
	headerSize := 1 + 4 + 4
	buf := make([]byte, headerSize+len(rec.Key)+len(rec.Value))

	buf[0] = rec.Type
	binary.LittleEndian.PutUint32(buf[1:5], uint32(len(rec.Key)))
	binary.LittleEndian.PutUint32(buf[5:9], uint32(len(rec.Value)))

	copy(buf[9:], rec.Key)
	copy(buf[9+len(rec.Key):], rec.Value)
	return buf
}

func DecodeLogRecord(buf []byte) *LogRecord {
	recType := buf[0]
	keySize := binary.LittleEndian.Uint32(buf[1:5])
	valSize := binary.LittleEndian.Uint32(buf[5:9])

	key := make([]byte, keySize)
	copy(key, buf[9:9+keySize])

	val := make([]byte, valSize)
	copy(val, buf[9+keySize:9+keySize+valSize])

	return &LogRecord{Key: key, Value: val, Type: recType}
}
