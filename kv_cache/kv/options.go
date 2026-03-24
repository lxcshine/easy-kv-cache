package kv

// Options 定义了数据库的启动配置
type Options struct {
	DirPath string // 数据库数据目录
}

var DefaultOptions = Options{
	DirPath: "./kv_data",
}
