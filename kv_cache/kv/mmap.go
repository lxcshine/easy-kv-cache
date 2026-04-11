package kv

import (
	"github.com/edsrzf/mmap-go"
	"os"
)

type MmapReader struct {
	data mmap.MMap
}

func OpenMmapReader(file *os.File) (*MmapReader, error) {
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	size := stat.Size()
	if size == 0 {
		return &MmapReader{data: nil}, nil
	}

	m, err := mmap.Map(file, mmap.RDONLY, 0)
	if err != nil {
		return nil, err
	}

	return &MmapReader{data: m}, nil
}

func (m *MmapReader) ReadZeroCopy(offset int64, size uint32) []byte {
	if m.data == nil || offset >= int64(len(m.data)) {
		return nil
	}
	return m.data[offset : offset+int64(size)]
}

func (m *MmapReader) Close() error {
	if m.data == nil {
		return nil
	}
	return m.data.Unmap()
}
