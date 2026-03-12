package tree

import (
	"iter"
)

// 逻辑存储结构：
// 提供目录树结构，支持聚合目录的覆盖率数据。

// TreeNode 组合模式核心接口
type TreeNode interface {
	Name() string           // 节点名称
	Path() string           // 节点完整路径
	GetStat() *TreeNodeData // 获取节点的统计数据
	Add(child TreeNode)     // 添加子节点
	Remove(child TreeNode)  // 删除子节点

	IsDir() bool                    // 是否为目录
	Children() iter.Seq[TreeNode]   // 迭代器模式
	Accept(visitor TreeNodeVisitor) // 访问者模式
}

// DirNode 目录节点 (Composite)
type DirNode struct {
	name     string
	fullPath string
	children []TreeNode // 组合：持有子节点
	stat     TreeNodeData
}

func NewDirNode(name, fullPath string) *DirNode {
	return &DirNode{
		name:     name,
		fullPath: fullPath,
		children: make([]TreeNode, 0),
	}
}

func (d *DirNode) Name() string { return d.name }
func (d *DirNode) Path() string { return d.fullPath }
func (d *DirNode) IsDir() bool  { return true }

func (d *DirNode) Add(child TreeNode) {
	d.children = append(d.children, child)
}

func (d *DirNode) Remove(child TreeNode) {
	for i, c := range d.children {
		if c == child {
			d.children = append(d.children[:i], d.children[i+1:]...)
			break
		}
	}
}

// GetStat 利用访问者模式聚合统计数据
func (d *DirNode) GetStat() *TreeNodeData {
	agg := &StatAggregator{}
	d.Accept(agg)
	return &agg.Stat
}

// FileNode 文件节点 (Leaf)
type FileNode struct {
	name     string
	fullPath string
	stat     TreeNodeData

	BlockOffset int64  // 块偏移量
	FileFlags   uint32 // 记录在文件上的特殊标记
}

func NewFileNode(name, fullPath string, stat TreeNodeData) *FileNode {
	return &FileNode{
		name:     name,
		fullPath: fullPath,
		stat:     stat,
	}
}

func (f *FileNode) Name() string { return f.name }
func (f *FileNode) Path() string { return f.fullPath }
func (f *FileNode) IsDir() bool  { return false }

func (f *FileNode) GetStat() *TreeNodeData {
	return &f.stat
}

func (f *FileNode) Add(child TreeNode)    {} // 叶子节点不支持
func (f *FileNode) Remove(child TreeNode) {} // 叶子节点不支持
