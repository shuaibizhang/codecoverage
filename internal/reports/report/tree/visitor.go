package tree

// TreeNodeVisitor 访问者模式接口，用于遍历树执行业务操作
type TreeNodeVisitor interface {
	VisitDirEnter(dir *DirNode)
	VisitDirExit(dir *DirNode)
	VisitFile(file *FileNode)
}

// Accept 实现访问者模式接口
func (d *DirNode) Accept(v TreeNodeVisitor) {
	v.VisitDirEnter(d)
	for _, child := range d.children {
		child.Accept(v)
	}
	v.VisitDirExit(d)
}

// Accept 实现访问者模式接口
func (f *FileNode) Accept(v TreeNodeVisitor) {
	v.VisitFile(f)
}

// StatAggregator 统计聚合器：具体的访问者实现
type StatAggregator struct {
	Stat TreeNodeData
}

func (s *StatAggregator) VisitDirEnter(dir *DirNode) {}

func (s *StatAggregator) VisitDirExit(dir *DirNode) {
	// 在退出目录时计算该层级的覆盖率百分比
	if s.Stat.InstrLines > 0 {
		s.Stat.Coverage = uint32(uint64(s.Stat.CoverLines) * 100 / uint64(s.Stat.InstrLines))
	}
	if s.Stat.IncrInstrLines > 0 {
		s.Stat.IncrCoverage = uint32(uint64(s.Stat.IncrCoverLines) * 100 / uint64(s.Stat.IncrInstrLines))
	}
}

func (s *StatAggregator) VisitFile(file *FileNode) {
	stat := file.GetStat()
	s.Stat.TotalLines += stat.TotalLines
	s.Stat.InstrLines += stat.InstrLines
	s.Stat.AddLines += stat.AddLines
	s.Stat.DeleteLines += stat.DeleteLines
	s.Stat.IncrInstrLines += stat.IncrInstrLines
	s.Stat.CoverLines += stat.CoverLines
	s.Stat.IncrCoverLines += stat.IncrCoverLines
}
