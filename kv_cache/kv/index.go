package kv

import "sync"


type Index struct {
	data sync.Map
}

func NewIndex() *Index {
	return &Index{}
}

func (i *Index) Put(key []byte, pos *LogRecordPos) {
	i.data.Store(string(key), pos)
}

func (i *Index) Get(key []byte) *LogRecordPos {
	if val, ok := i.data.Load(string(key)); ok {
		return val.(*LogRecordPos)
	}
	return nil
}

func (i *Index) Delete(key []byte) {
	i.data.Delete(string(key))
}
