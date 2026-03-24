package kv

import (
	"bufio"
	"os"
	"sync"
)

// DataFile 管理磁盘上的数据文件
type DataFile struct {
	File   *os.File
	Writer *bufio.Writer // 核心优化：引入内存写缓冲
	Offset int64
	mu     sync.Mutex
}

func OpenDataFile(path string) (*DataFile, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	stat, _ := file.Stat()
	return &DataFile{
		File: file,
		// 使用 64KB 的缓冲区
		Writer: bufio.NewWriterSize(file, 64*1024),
		Offset: stat.Size(),
	}, nil
}

func (df *DataFile) Write(data []byte) (int64, error) {
	df.mu.Lock()
	defer df.mu.Unlock()

	offset := df.Offset
	// 写入内存缓冲而不是直接刷盘，速度提升百倍
	n, err := df.Writer.Write(data)
	if err != nil {
		return 0, err
	}
	df.Offset += int64(n)
	return offset, nil
}

func (df *DataFile) Read(offset int64, size uint32) ([]byte, error) {
	buf := make([]byte, size)
	_, err := df.File.ReadAt(buf, offset)
	return buf, err
}

func (df *DataFile) Sync() error {
	df.mu.Lock()
	defer df.mu.Unlock()
	if err := df.Writer.Flush(); err != nil {
		return err
	}
	return df.File.Sync()
}

func (df *DataFile) Close() error {
	_ = df.Sync()
	return df.File.Close()
}
