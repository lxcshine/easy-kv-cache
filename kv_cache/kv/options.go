package kv

type Options struct {
	DirPath string // 数据库数据目录
}

var DefaultOptions = Options{
	DirPath: "./kv_data",
}
