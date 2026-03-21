package tree

type TreeNodeData struct {
	Name  string
	Path  string
	IsDir bool
	FileLineInfo
	FileCoverInfo
	HasIncrement bool // 是否有增量数据
}

// FileLineInfo 文件行信息
type FileLineInfo struct {
	TotalLines     uint32 // 代码总行数
	InstrLines     uint32 // 指令行数
	AddLines       uint32 // 新增行数
	DeleteLines    uint32 // 删除行数
	IncrInstrLines uint32 // 新增指令行数
}

// FileCoverInfo 文件覆盖率信息
type FileCoverInfo struct {
	CoverLines     uint32 // 全量覆盖代码行数
	Coverage       uint32 // 全量覆盖率
	IncrCoverLines uint32 // 增量覆盖行数
	IncrCoverage   uint32 // 增量覆盖率
}
