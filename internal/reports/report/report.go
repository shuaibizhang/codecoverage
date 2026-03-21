package report

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/shuaibizhang/codecoverage/internal/reports/report/storage/partitionkey"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/tree"
)

/*
覆盖率报告
1、物理存储结构：
提供高性能存储，存储分为.cno文件和.cda文件
2）.cno文件存储覆盖率元数据信息和前缀目录树（方便快速索引行覆盖率文件）。
3）.cda文件存储压缩后的行覆盖率数据。
2、逻辑存储结构：
提供目录树结构，支持聚合目录的覆盖率数据。
*/

type CoverReport interface {
	/* 增，添加文件和其覆盖率数据 */
	AddFile(path string, lines []int32, diffInfo FileDiffInfo) error

	/* 改，修改文件报告覆盖率数据 */
	UpdateFile(path string, lines []int32, diffInfo FileDiffInfo, flags uint32) error

	/* 查，查找元数据、树节点覆盖率概览、文件覆盖行 */
	// GetMeta 获取覆盖率报告元数据信息(项目覆盖率概览数据)
	GetMeta() MetaInfo
	// FindFile 列出路径为path的目录下子节点的覆盖率数据概览概览列表
	ListFileStats(path string, isIncrement bool) ([]tree.TreeNodeData, error)
	// 获取文件的行覆盖率数据
	GetFileCoverLines(filePath string) ([]uint32, error)
	// ExistFile 判断是否存在路径为path的文件
	ExistFile(path string) bool

	// Unmarshal 使用 partitionkey 从存储中加载报告数据并填充当前报告对象
	Unmarshal(ctx context.Context, pk partitionkey.PartitionKey) error

	// FindNode 获取路径为 path 的节点
	FindNode(path string) tree.TreeNode

	// Flush 刷新报告到存储源
	Flush(ctx context.Context) error
	// Close 关闭报告，释放资源
	Close(ctx context.Context) error
}

// Storage 覆盖率报告持久化接口
type Storage interface {
	// 设置覆盖行到存储
	SetCoverLine(ctx context.Context, pk partitionkey.PartitionKey, coverLines []int32, addedLines []uint32) (partitionkey.PartitionKey, error)
	// 从存储获取覆盖行
	GetCoverLine(ctx context.Context, pk partitionkey.PartitionKey) ([]int32, []uint32, error)
	// 实现获取带了指令行、增量行标识的行覆盖率数据
	GetCoverLineWithFlag(ctx context.Context, pk partitionkey.PartitionKey) ([]uint32, error)

	// 设置报告到存储
	SetReport(ctx context.Context, pk partitionkey.PartitionKey, report CoverReport) (partitionkey.PartitionKey, error)
	// 从存储获取报告并填充到传入的对象中
	LoadReport(ctx context.Context, pk partitionkey.PartitionKey, report CoverReport) error

	// Close 关闭存储，释放资源
	Close() error
}

type MetaInfo struct {
	HostName      string   `json:"hostname"` // 上报主机名
	Module        string   `json:"module"`
	Branch        string   `json:"branch"`
	Commit        string   `json:"commit"`
	BaseCommit    string   `json:"base_commit"`
	FilterVersion uint32   `json:"filter_version"` // 过滤版本
	TotalFiles    uint32   `json:"total_files"`    // 总文件数
	InheritInfo   string   `json:"inherit_info"`   // 继承信息
	LastUpdate    string   `json:"last_update"`    // 最后更新时间
	Columns       []string `json:"columns"`        // 列名
}

type CoverReportImpl struct {
	Meta         MetaInfo                  // 元数据信息
	Tree         tree.TreeNode             // 逻辑目录树
	PartitionKey partitionkey.PartitionKey // 分区key，用于索引元数据信息

	Storage Storage // 存储提供者，用于延迟加载行覆盖率数据

	Change bool // 是否有变更

	// 进程锁
	Mutex sync.Mutex
}

func (r *CoverReportImpl) GetMeta() MetaInfo {
	return r.Meta
}

func (r *CoverReportImpl) AddFile(path string, lines []int32, diffInfo FileDiffInfo) error {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	// 1. 计算统计数据
	stat := tree.TreeNodeData{}
	stat.TotalLines = uint32(len(lines))
	for _, l := range lines {
		if l != -1 {
			stat.InstrLines++
			if l > 0 {
				stat.CoverLines++
			}
		}
	}
	if stat.InstrLines > 0 {
		stat.Coverage = uint32(uint64(stat.CoverLines) * 100 / uint64(stat.InstrLines))
	}

	// 增量统计
	stat.AddLines = diffInfo.AddLines
	stat.DeleteLines = diffInfo.DeleteLines
	// 这里假设 diffInfo.AddedLines 包含了增量指令行的行号(1-based)
	for _, lineNum := range diffInfo.AddedLines {
		if lineNum > 0 && int(lineNum) <= len(lines) {
			l := lines[lineNum-1]
			if l != -1 {
				stat.IncrInstrLines++
				if l > 0 {
					stat.IncrCoverLines++
				}
			}
		}
	}
	if stat.IncrInstrLines > 0 {
		stat.IncrCoverage = uint32(uint64(stat.IncrCoverLines) * 100 / uint64(stat.IncrInstrLines))
		stat.HasIncrement = true
	}

	// 2. 存入数据源获取 PartitionKey
	if r.Storage != nil {
		pk := partitionkey.NewCoverageKey(r.PartitionKey.RealPathPrefix(), 0)
		newPk, err := r.Storage.SetCoverLine(context.Background(), pk, lines, diffInfo.AddedLines)
		if err != nil {
			return fmt.Errorf("storage set cover line: %w", err)
		}

		// 3. 创建并插入节点
		// 简便起见，我们手动导航并添加
		trimmedPath := strings.Trim(path, "/")
		parts := strings.Split(trimmedPath, "/")
		current := r.Tree.(*tree.DirNode)

		for i := 0; i < len(parts)-1; i++ {
			dirName := parts[i]
			if dirName == "" {
				continue
			}
			fullPath := strings.Join(parts[:i+1], "/")
			// FindChild or Create
			child := current.FindChild(dirName)
			if child == nil {
				newDir := tree.NewDirNode(dirName, fullPath)
				current.Add(newDir)
				current = newDir
			} else {
				dir, ok := child.(*tree.DirNode)
				if !ok {
					return fmt.Errorf("node %s is not a directory", fullPath)
				}
				current = dir
			}
		}

		fileName := parts[len(parts)-1]
		fileNode := tree.NewFileNode(fileName, path, stat)
		fileNode.BlockOffset = newPk.Offset()
		current.Add(fileNode)
	}

	r.Change = true
	r.Meta.TotalFiles++
	return nil
}

func (r *CoverReportImpl) UpdateFile(path string, lines []int32, diffInfo FileDiffInfo, flags uint32) error {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	// 查找节点
	node := r.findNode(path)
	if node == nil {
		return fmt.Errorf("file not found: %s", path)
	}
	fileNode, ok := node.(*tree.FileNode)
	if !ok {
		return fmt.Errorf("not a file node: %s", path)
	}

	// 1. 更新统计数据
	stat := tree.TreeNodeData{}
	stat.TotalLines = uint32(len(lines))
	for _, l := range lines {
		if l != -1 {
			stat.InstrLines++
			if l > 0 {
				stat.CoverLines++
			}
		}
	}
	if stat.InstrLines > 0 {
		stat.Coverage = uint32(uint64(stat.CoverLines) * 100 / uint64(stat.InstrLines))
	}

	stat.AddLines = diffInfo.AddLines
	stat.DeleteLines = diffInfo.DeleteLines
	for _, lineNum := range diffInfo.AddedLines {
		if lineNum > 0 && int(lineNum) <= len(lines) {
			l := lines[lineNum-1]
			if l != -1 {
				stat.IncrInstrLines++
				if l > 0 {
					stat.IncrCoverLines++
				}
			}
		}
	}
	if stat.IncrInstrLines > 0 {
		stat.IncrCoverage = uint32(uint64(stat.IncrCoverLines) * 100 / uint64(stat.IncrInstrLines))
		stat.HasIncrement = true
	}

	// 2. 存入存储
	if r.Storage != nil {
		pk := partitionkey.NewCoverageKey(r.PartitionKey.RealPathPrefix(), fileNode.BlockOffset)
		newPk, err := r.Storage.SetCoverLine(context.Background(), pk, lines, diffInfo.AddedLines)
		if err != nil {
			return fmt.Errorf("storage set cover line: %w", err)
		}

		// 3. 更新节点
		// 既然 tree.TreeNode 没有直接修改 Stat 的接口，这里我们假设 FileNode.stat 是导出的或通过指针修改
		// 根据之前的代码，FileNode.stat 是私有的，但有 GetStat()。
		// 由于我们在同一个 package 或者 tree 包中，可能需要微调。
		// 这里我们重新创建一个节点替换，或者直接修改（如果 tree 包支持）。
		// 暂且直接赋值（假设我们在开发过程中可以调整 tree 包）。
		*fileNode.GetStat() = stat
		fileNode.BlockOffset = newPk.Offset()
		fileNode.FileFlags = flags
	}

	r.Change = true
	return nil
}

func (r *CoverReportImpl) ListFileStats(path string, isIncrement bool) ([]tree.TreeNodeData, error) {
	node := r.findNode(path)
	if node == nil {
		return nil, fmt.Errorf("node not found: %s", path)
	}

	dir, ok := node.(*tree.DirNode)
	if !ok {
		// 如果是文件，只返回该文件自身的统计信息
		stat := *node.GetStat()
		if isIncrement && !stat.HasIncrement {
			return nil, nil
		}
		return []tree.TreeNodeData{stat}, nil
	}

	// 如果是目录，返回该目录下所有一级子节点的统计信息
	var res []tree.TreeNodeData
	for child := range dir.Children() {
		stat := *child.GetStat()
		if isIncrement && !stat.HasIncrement {
			continue
		}
		res = append(res, stat)
	}
	return res, nil
}

func (r *CoverReportImpl) GetFileCoverLines(filePath string) ([]uint32, error) {
	node := r.findNode(filePath)
	if node == nil {
		return nil, fmt.Errorf("file not found: %s", filePath)
	}
	fileNode, ok := node.(*tree.FileNode)
	if !ok {
		return nil, fmt.Errorf("not a file node: %s", filePath)
	}

	if r.Storage == nil {
		return nil, fmt.Errorf("storage not initialized")
	}

	// 延迟加载：从存储读取带标识的覆盖行
	pk := partitionkey.NewCoverageKey(r.PartitionKey.RealPathPrefix(), fileNode.BlockOffset)
	return r.Storage.GetCoverLineWithFlag(context.Background(), pk)
}

func (r *CoverReportImpl) ExistFile(path string) bool {
	return r.findNode(path) != nil
}

func (r *CoverReportImpl) Unmarshal(ctx context.Context, pk partitionkey.PartitionKey) error {
	if r.Storage == nil {
		return fmt.Errorf("storage not initialized")
	}
	r.PartitionKey = pk
	return r.Storage.LoadReport(ctx, pk, r)
}

func (r *CoverReportImpl) FindNode(path string) tree.TreeNode {
	return r.findNode(path)
}

// 辅助方法：查找节点
func (r *CoverReportImpl) findNode(path string) tree.TreeNode {
	if path == "" || path == "*" {
		return r.Tree
	}
	// 去除首尾的 / 并分割
	trimmedPath := strings.Trim(path, "/")
	parts := strings.Split(trimmedPath, "/")
	var current tree.TreeNode = r.Tree

	for _, part := range parts {
		if part == "" {
			continue
		}
		dir, ok := current.(*tree.DirNode)
		if !ok {
			return nil
		}
		current = dir.FindChild(part)
		if current == nil {
			return nil
		}
	}
	return current
}

func (r *CoverReportImpl) Flush(ctx context.Context) error {
	if r.Storage == nil {
		return fmt.Errorf("storage not initialized")
	}
	_, err := r.Storage.SetReport(ctx, r.PartitionKey, r)
	return err
}

func (r *CoverReportImpl) Close(ctx context.Context) error {
	if r.Storage != nil {
		return r.Storage.Close()
	}
	return nil
}

func NewCoverReport(storage Storage, meta MetaInfo, key partitionkey.PartitionKey) *CoverReportImpl {
	root := tree.NewDirNode("*", "*")
	return &CoverReportImpl{
		Tree:         root,
		Storage:      storage,
		PartitionKey: key,
		Meta:         meta,
	}
}
