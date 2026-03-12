package tree

import (
	"strings"
)

// TreeBuildListener 监听器模式接口：监听树构建过程中的事件
type TreeBuildListener interface {
	OnNodeFound(parent *DirNode, node TreeNode)
	OnNodeCreated(parent *DirNode, node TreeNode)
}

// defaultBuildListener 默认空实现
type defaultBuildListener struct{}

func (l *defaultBuildListener) OnNodeFound(parent *DirNode, node TreeNode)   {}
func (l *defaultBuildListener) OnNodeCreated(parent *DirNode, node TreeNode) {}

// PathNavigator 路径导航器：执行目录寻址逻辑并触发监听器
type PathNavigator struct {
	listener TreeBuildListener
}

func NewPathNavigator(l TreeBuildListener) *PathNavigator {
	if l == nil {
		l = &defaultBuildListener{}
	}
	return &PathNavigator{listener: l}
}

func (n *PathNavigator) GetOrCreateDir(current *DirNode, name, fullPath string) *DirNode {
	child := current.FindChild(name)
	if child != nil {
		if dir, ok := child.(*DirNode); ok {
			n.listener.OnNodeFound(current, dir)
			return dir
		}
	}

	// 没找到则创建新目录并触发监听事件
	newDir := NewDirNode(name, fullPath)
	current.Add(newDir)
	n.listener.OnNodeCreated(current, newDir)
	return newDir
}

// TreeBuilder 构造者模式实现：封装从路径列表构建目录树的逻辑
type TreeBuilder struct {
	root      *DirNode
	navigator *PathNavigator
}

func NewTreeBuilder(rootName string) *TreeBuilder {
	return &TreeBuilder{
		root:      NewDirNode(rootName, rootName),
		navigator: NewPathNavigator(nil),
	}
}

func (b *TreeBuilder) WithListener(l TreeBuildListener) *TreeBuilder {
	b.navigator.listener = l
	return b
}

func (b *TreeBuilder) AddFile(path string, stat TreeNodeData) {
	parts := strings.Split(path, "/")
	current := b.root

	for i := 0; i < len(parts)-1; i++ {
		dirName := parts[i]
		fullPath := strings.Join(parts[:i+1], "/")
		current = b.navigator.GetOrCreateDir(current, dirName, fullPath)
	}

	// 最后处理文件叶子节点
	fileName := parts[len(parts)-1]
	fileNode := NewFileNode(fileName, path, stat)
	current.Add(fileNode)
	b.navigator.listener.OnNodeCreated(current, fileNode)
}

func (b *TreeBuilder) Build() *DirNode {
	return b.root
}
