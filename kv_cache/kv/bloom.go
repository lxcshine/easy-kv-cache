package kv

import "hash/crc32"

type BloomFilter struct {
	bitset []bool
	size   uint32
}

func NewBloomFilter(size uint32) *BloomFilter {
	return &BloomFilter{
		bitset: make([]bool, size),
		size:   size,
	}
}

func (bf *BloomFilter) hash(key []byte) uint32 {
	return crc32.ChecksumIEEE(key) % bf.size
}

func (bf *BloomFilter) Add(key []byte) {
	idx := bf.hash(key)
	bf.bitset[idx] = true
}

func (bf *BloomFilter) MightContain(key []byte) bool {
	idx := bf.hash(key)
	return bf.bitset[idx]
}
