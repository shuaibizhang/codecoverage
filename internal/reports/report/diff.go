package report

type FileDiffInfo struct {
	AddedLines  []uint32
	AddLines    uint32 // 新增行数
	DeleteLines uint32 // 删除行数
}
