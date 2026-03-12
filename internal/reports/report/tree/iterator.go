package tree

import (
	"iter"
)

// Children 迭代器模式实现：遍历目录子节点
func (d *DirNode) Children() iter.Seq[TreeNode] {
	return func(yield func(TreeNode) bool) {
		for _, child := range d.children {
			if !yield(child) {
				return
			}
		}
	}
}

// Children 迭代器模式实现：文件节点没有子节点
func (f *FileNode) Children() iter.Seq[TreeNode] {
	return func(yield func(TreeNode) bool) {
		// 叶子节点直接返回空序列
	}
}

// FindChild 利用迭代器模式查找同名子节点
func (d *DirNode) FindChild(name string) TreeNode {
	for child := range d.Children() {
		if child.Name() == name {
			return child
		}
	}
	return nil
}
