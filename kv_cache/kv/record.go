package kv

import "encoding/binary"

const (
	RecordNormal  byte = iota // 正常数据
	RecordDeleted             // 表示删除
)

// 写入磁盘的日志记录
type LogRecord struct {
	Key   []byte
	Value []byte
	Type  byte
}

// 内存索引位置信息
type LogRecordPos struct {
	Offset int64  // 记录在文件中的起始位置
	Size   uint32 // 记录在文件中的总长度
}

// 记录编码为字节流
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

// 从字节流中解码出记录
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
