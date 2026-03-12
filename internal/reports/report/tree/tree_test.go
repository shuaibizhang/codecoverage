package tree

import (
	"testing"
)

func TestDirNode_GetStat(t *testing.T) {
	// 1. 创建文件节点 1
	file1Stat := TreeNodeData{
		FileLineInfo: FileLineInfo{
			InstrLines: 100,
		},
		FileCoverInfo: FileCoverInfo{
			CoverLines: 80,
		},
	}
	file1 := NewFileNode("file1.go", "a/b/file1.go", file1Stat)

	// 2. 创建文件节点 2
	file2Stat := TreeNodeData{
		FileLineInfo: FileLineInfo{
			InstrLines: 200,
		},
		FileCoverInfo: FileCoverInfo{
			CoverLines: 120,
		},
	}
	file2 := NewFileNode("file2.go", "a/b/file2.go", file2Stat)

	// 3. 创建目录节点并添加文件
	dir := NewDirNode("b", "a/b")
	dir.Add(file1)
	dir.Add(file2)

	// 4. 获取目录统计数据
	stat := dir.GetStat()

	// 5. 验证结果
	// 总指令行数应该是 100 + 200 = 300
	if stat.InstrLines != 300 {
		t.Errorf("Expected InstrLines 300, got %d", stat.InstrLines)
	}
	// 总覆盖行数应该是 80 + 120 = 200
	if stat.CoverLines != 200 {
		t.Errorf("Expected CoverLines 200, got %d", stat.CoverLines)
	}
	// 覆盖率应该是 (200 / 300) * 100 = 66
	if stat.Coverage != 66 {
		t.Errorf("Expected Coverage 66, got %d", stat.Coverage)
	}

	// 6. 测试嵌套目录
	parentDir := NewDirNode("a", "a")
	parentDir.Add(dir)

	file3Stat := TreeNodeData{
		FileLineInfo: FileLineInfo{
			InstrLines: 100,
		},
		FileCoverInfo: FileCoverInfo{
			CoverLines: 100,
		},
	}
	file3 := NewFileNode("file3.go", "a/file3.go", file3Stat)
	parentDir.Add(file3)

	parentStat := parentDir.GetStat()
	// 总指令行数应该是 300 + 100 = 400
	if parentStat.InstrLines != 400 {
		t.Errorf("Expected parent InstrLines 400, got %d", parentStat.InstrLines)
	}
	// 总覆盖行数应该是 200 + 100 = 300
	if parentStat.CoverLines != 300 {
		t.Errorf("Expected parent CoverLines 300, got %d", parentStat.CoverLines)
	}
	// 覆盖率应该是 (300 / 400) * 100 = 75
	if parentStat.Coverage != 75 {
		t.Errorf("Expected parent Coverage 75, got %d", parentStat.Coverage)
	}
}

func TestTreeBuilder_Build(t *testing.T) {
	builder := NewTreeBuilder("project")

	// 添加文件 1: project/src/main.go
	builder.AddFile("src/main.go", TreeNodeData{
		FileLineInfo:  FileLineInfo{InstrLines: 100},
		FileCoverInfo: FileCoverInfo{CoverLines: 50},
	})

	// 添加文件 2: project/src/utils/helper.go
	builder.AddFile("src/utils/helper.go", TreeNodeData{
		FileLineInfo:  FileLineInfo{InstrLines: 200},
		FileCoverInfo: FileCoverInfo{CoverLines: 150},
	})

	// 添加文件 3: project/tests/test_main.go
	builder.AddFile("tests/test_main.go", TreeNodeData{
		FileLineInfo:  FileLineInfo{InstrLines: 50},
		FileCoverInfo: FileCoverInfo{CoverLines: 50},
	})

	root := builder.Build()

	// 1. 验证根节点名
	if root.Name() != "project" {
		t.Errorf("Expected root name 'project', got %s", root.Name())
	}

	// 2. 验证总统计数据
	stat := root.GetStat()
	// 总指令行: 100 + 200 + 50 = 350
	if stat.InstrLines != 350 {
		t.Errorf("Expected total InstrLines 350, got %d", stat.InstrLines)
	}
	// 总覆盖行: 50 + 150 + 50 = 250
	if stat.CoverLines != 250 {
		t.Errorf("Expected total CoverLines 250, got %d", stat.CoverLines)
	}
	// 总覆盖率: (250 / 350) * 100 = 71
	if stat.Coverage != 71 {
		t.Errorf("Expected total Coverage 71, got %d", stat.Coverage)
	}

	// 3. 验证子目录 src 的统计数据
	var srcDir *DirNode
	for _, child := range root.children {
		if d, ok := child.(*DirNode); ok && d.Name() == "src" {
			srcDir = d
			break
		}
	}

	if srcDir == nil {
		t.Fatal("src directory not found")
	}

	srcStat := srcDir.GetStat()
	// src 总指令行: 100 + 200 = 300
	if srcStat.InstrLines != 300 {
		t.Errorf("Expected src InstrLines 300, got %d", srcStat.InstrLines)
	}
	// src 总覆盖率: (200 / 300) * 100 = 66
	if srcStat.Coverage != 66 {
		t.Errorf("Expected src Coverage 66, got %d", srcStat.Coverage)
	}
}

type mockVisitor struct {
	filesVisited []string
	dirsEntered  []string
	dirsExited   []string
}

func (m *mockVisitor) VisitDirEnter(dir *DirNode) { m.dirsEntered = append(m.dirsEntered, dir.Name()) }
func (m *mockVisitor) VisitDirExit(dir *DirNode)  { m.dirsExited = append(m.dirsExited, dir.Name()) }
func (m *mockVisitor) VisitFile(file *FileNode) {
	m.filesVisited = append(m.filesVisited, file.Name())
}

func TestTreeNode_Accept(t *testing.T) {
	builder := NewTreeBuilder("root")
	builder.AddFile("a/b/f1.go", TreeNodeData{})
	builder.AddFile("a/f2.go", TreeNodeData{})
	root := builder.Build()

	visitor := &mockVisitor{}
	root.Accept(visitor)

	// 预期访问顺序（深度优先）: root(Enter), a(Enter), b(Enter), f1.go(Visit), b(Exit), f2.go(Visit), a(Exit), root(Exit)
	expectedFiles := []string{"f1.go", "f2.go"}
	expectedDirsEntered := []string{"root", "a", "b"}
	expectedDirsExited := []string{"b", "a", "root"}

	if len(visitor.filesVisited) != len(expectedFiles) {
		t.Errorf("Files visited mismatch: got %v, want %v", visitor.filesVisited, expectedFiles)
	}
	if len(visitor.dirsEntered) != len(expectedDirsEntered) {
		t.Errorf("Dirs entered mismatch: got %v, want %v", visitor.dirsEntered, expectedDirsEntered)
	}
	if len(visitor.dirsExited) != len(expectedDirsExited) {
		t.Errorf("Dirs exited mismatch: got %v, want %v", visitor.dirsExited, expectedDirsExited)
	}
}

func TestTreeNode_ChildrenIterator(t *testing.T) {
	builder := NewTreeBuilder("root")
	builder.AddFile("a.go", TreeNodeData{})
	builder.AddFile("b.go", TreeNodeData{})
	root := builder.Build()

	var children []string
	for child := range root.Children() {
		children = append(children, child.Name())
	}

	expected := []string{"a.go", "b.go"}
	if len(children) != len(expected) {
		t.Errorf("Expected %d children, got %v", len(expected), children)
	}
	for i, name := range expected {
		if children[i] != name {
			t.Errorf("Expected child %d to be %s, got %s", i, name, children[i])
		}
	}
}

type buildTracker struct {
	foundCount   int
	createdCount int
}

func (b *buildTracker) OnNodeFound(parent *DirNode, node TreeNode)   { b.foundCount++ }
func (b *buildTracker) OnNodeCreated(parent *DirNode, node TreeNode) { b.createdCount++ }

func TestTreeBuilder_WithListener(t *testing.T) {
	tracker := &buildTracker{}
	builder := NewTreeBuilder("root").WithListener(tracker)

	// 1. 添加第一个文件：创建 a, a/b, a/b/f1.go (3 个节点)
	builder.AddFile("a/b/f1.go", TreeNodeData{})
	if tracker.createdCount != 3 {
		t.Errorf("Expected 3 nodes created, got %d", tracker.createdCount)
	}

	// 2. 添加第二个文件：发现 a, a/b, 创建 a/b/f2.go (发现 2 次，创建 1 次)
	builder.AddFile("a/b/f2.go", TreeNodeData{})
	if tracker.foundCount != 2 {
		t.Errorf("Expected 2 nodes found, got %d", tracker.foundCount)
	}
	if tracker.createdCount != 4 { // 3 + 1
		t.Errorf("Expected 4 nodes total created, got %d", tracker.createdCount)
	}
}
