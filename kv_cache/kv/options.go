package kv

type Options struct {
	DirPath        string
	ValueThreshold int64
	CacheCapacity  int
	BloomSize      uint32
}

var DefaultOptions = Options{
	DirPath:        "./kv_data",
	ValueThreshold: 128 * 1024,
	CacheCapacity:  10000,
	BloomSize:      1000000,
}
