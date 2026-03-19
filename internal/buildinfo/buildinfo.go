package buildinfo

// 构建信息，用于做版本管理，在build时通过ldflags链接时注入信息
var (
	Version   string = "1.0.0" // 语义化版本
	BuildTime string           // 构建时间
	Commit    string           // 提交commit记录
)
