package manager

import (
	"context"
	"fmt"

	"github.com/shuaibizhang/codecoverage/internal/diff"
	"github.com/shuaibizhang/codecoverage/internal/reports/report"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/storage/coder"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/storage/partitionkey"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/tree"
)

type StorageFactory func(ctx context.Context, pk partitionkey.PartitionKey) (report.Storage, error)

type ReportManager interface {
	/** 操作报告 **/
	// 创建一个全新报告
	CreateReport(ctx context.Context, meta report.MetaInfo, key partitionkey.PartitionKey) (report.CoverReport, error)
	// Open 只读打开：并发安全，无锁
	Open(ctx context.Context, pk partitionkey.PartitionKey) (report.CoverReport, error)
	// OpenWrite 读写打开：互斥锁，用于更新
	OpenWrite(ctx context.Context, pk partitionkey.PartitionKey) (report.CoverReport, error)

	/** 合并报告，以base报告为基础，合并other报告的覆盖行 **/
	// 同源合并，相同代码的覆盖率报告进行合并
	MergeSameCommitReport(ctx context.Context, base, other report.CoverReport) error
	// 异源合并，不同代码的覆盖率报告进行合并
	MergeDiffCommitReport(ctx context.Context, base, other report.CoverReport, diff map[string]*diff.DiffFile) error

	/** 增量覆盖率计算 **/
	// 修改basecommit，重新计算增量覆盖率
	RebaseReport(ctx context.Context, meta report.MetaInfo, report report.CoverReport, diff map[string]*diff.DiffFile) error
}

type reportManager struct {
	storageFactory StorageFactory
}

func NewReportManager(factory StorageFactory) ReportManager {
	return &reportManager{
		storageFactory: factory,
	}
}

func (m *reportManager) CreateReport(ctx context.Context, meta report.MetaInfo, key partitionkey.PartitionKey) (report.CoverReport, error) {
	st, err := m.storageFactory(ctx, key)
	if err != nil {
		return nil, err
	}
	return report.NewCoverReport(st, meta, key), nil
}

func (m *reportManager) Open(ctx context.Context, pk partitionkey.PartitionKey) (report.CoverReport, error) {
	st, err := m.storageFactory(ctx, pk)
	if err != nil {
		return nil, err
	}
	rep := report.NewCoverReport(st, report.MetaInfo{}, pk)
	if err := rep.Unmarshal(ctx, pk); err != nil {
		return nil, err
	}
	return rep, nil
}

func (m *reportManager) OpenWrite(ctx context.Context, pk partitionkey.PartitionKey) (report.CoverReport, error) {
	st, err := m.storageFactory(ctx, pk)
	if err != nil {
		return nil, err
	}
	rep := report.NewCoverReport(st, report.MetaInfo{}, pk)
	if err := rep.Unmarshal(ctx, pk); err != nil {
		return nil, err
	}
	return rep, nil
}

func (m *reportManager) MergeSameCommitReport(ctx context.Context, base, other report.CoverReport) error {
	// 遍历 other 报告中的所有文件
	// 这里假设我们可以通过某种方式遍历报告中的文件
	// 在 CoverReport 接口中并没有直接提供遍历文件的方法，但 CoverReportImpl 有 Tree
	// 我们可能需要为 CoverReport 增加遍历接口，或者在这里进行类型断言

	otherImpl, ok := other.(*report.CoverReportImpl)
	if !ok {
		return fmt.Errorf("other report is not CoverReportImpl")
	}

	// 使用访问者模式遍历
	visitor := &mergeVisitor{
		ctx:   ctx,
		base:  base,
		other: other,
	}
	otherImpl.Tree.Accept(visitor)

	return visitor.err
}

type mergeVisitor struct {
	ctx   context.Context
	base  report.CoverReport
	other report.CoverReport
	err   error
}

func (v *mergeVisitor) VisitDirEnter(dir *tree.DirNode) {}
func (v *mergeVisitor) VisitDirExit(dir *tree.DirNode)  {}
func (v *mergeVisitor) VisitFile(file *tree.FileNode) {
	if v.err != nil {
		return
	}

	path := file.Path()
	otherUintLines, err := v.other.GetFileCoverLines(path)
	if err != nil {
		v.err = err
		return
	}

	otherLines, otherAddedLines := coder.DecodeUintLines(otherUintLines)

	if v.base.ExistFile(path) {
		baseUintLines, err := v.base.GetFileCoverLines(path)
		if err != nil {
			v.err = err
			return
		}
		baseLines, baseAddedLines := coder.DecodeUintLines(baseUintLines)

		// 合并行覆盖率 (简单的按位或)
		if len(baseLines) != len(otherLines) {
			// 如果长度不一致，理论上同源合并不会发生，这里做简单处理
			v.err = fmt.Errorf("file %s lines length mismatch: %d vs %d", path, len(baseLines), len(otherLines))
			return
		}

		for i := range baseLines {
			if otherLines[i] > 0 {
				if baseLines[i] <= 0 || otherLines[i] > baseLines[i] {
					baseLines[i] = otherLines[i]
				}
			}
		}

		// 合并增量行号
		addedLinesMap := make(map[uint32]bool)
		for _, l := range baseAddedLines {
			addedLinesMap[l] = true
		}
		for _, l := range otherAddedLines {
			addedLinesMap[l] = true
		}
		mergedAddedLines := make([]uint32, 0, len(addedLinesMap))
		for l := range addedLinesMap {
			mergedAddedLines = append(mergedAddedLines, l)
		}

		// 获取 diffInfo，这里假设 MergeSameCommitReport 不需要更新 diffInfo 统计，或者从节点获取
		stat := file.GetStat()
		diffInfo := report.FileDiffInfo{
			AddLines:    stat.AddLines,
			DeleteLines: stat.DeleteLines,
			AddedLines:  mergedAddedLines,
		}

		v.err = v.base.UpdateFile(path, baseLines, diffInfo, 0)
	} else {
		// 如果 base 中不存在，则添加
		stat := file.GetStat()
		diffInfo := report.FileDiffInfo{
			AddLines:    stat.AddLines,
			DeleteLines: stat.DeleteLines,
			AddedLines:  otherAddedLines,
		}
		v.err = v.base.AddFile(path, otherLines, diffInfo)
	}
}

func (m *reportManager) MergeDiffCommitReport(ctx context.Context, base, other report.CoverReport, diff map[string]*diff.DiffFile) error {
	// 异源合并逻辑更为复杂，需要根据 diff 进行行号映射映射
	// 1. 遍历 other
	// 2. 查找 diff 映射关系
	// 3. 将 other 的行号映射到 base 的行号
	// 4. 合并数据

	otherImpl, ok := other.(*report.CoverReportImpl)
	if !ok {
		return fmt.Errorf("other report is not CoverReportImpl")
	}

	visitor := &mergeDiffVisitor{
		ctx:     ctx,
		base:    base,
		other:   other,
		diffMap: diff,
	}
	otherImpl.Tree.Accept(visitor)

	return visitor.err
}

type mergeDiffVisitor struct {
	ctx     context.Context
	base    report.CoverReport
	other   report.CoverReport
	diffMap map[string]*diff.DiffFile
	err     error
}

func (v *mergeDiffVisitor) VisitDirEnter(dir *tree.DirNode) {}
func (v *mergeDiffVisitor) VisitDirExit(dir *tree.DirNode)  {}
func (v *mergeDiffVisitor) VisitFile(file *tree.FileNode) {
	if v.err != nil {
		return
	}

	path := file.Path()
	// 这里简化处理：如果不在 diffMap 中，认为代码没变或不在关注范围内
	// 实际上需要更复杂的逻辑来处理重命名等

	otherUintLines, err := v.other.GetFileCoverLines(path)
	if err != nil {
		v.err = err
		return
	}
	otherLines, _ := coder.DecodeUintLines(otherUintLines)

	// 如果有 diff 信息，进行行号转换
	if dFile, ok := v.diffMap[path]; ok {
		// TODO: 实现基于 Hunks 的行号映射逻辑
		// 暂时只处理文件存在的场景
		_ = dFile
	}

	// 简化的逻辑：如果文件在 base 中存在，则合并（暂不考虑异源行号偏移）
	if v.base.ExistFile(path) {
		baseUintLines, err := v.base.GetFileCoverLines(path)
		if err != nil {
			v.err = err
			return
		}
		baseLines, baseAddedLines := coder.DecodeUintLines(baseUintLines)

		// 合并逻辑... (同上，实际需考虑偏移)
		for i := range baseLines {
			if i < len(otherLines) && otherLines[i] > 0 {
				baseLines[i] = otherLines[i]
			}
		}

		stat := file.GetStat()
		v.err = v.base.UpdateFile(path, baseLines, report.FileDiffInfo{
			AddLines:    stat.AddLines,
			DeleteLines: stat.DeleteLines,
			AddedLines:  baseAddedLines,
		}, 0)
	}
}

func (m *reportManager) RebaseReport(ctx context.Context, meta report.MetaInfo, rep report.CoverReport, diff map[string]*diff.DiffFile) error {
	// Rebase 主要是重新计算增量覆盖率
	rImpl, ok := rep.(*report.CoverReportImpl)
	if !ok {
		return fmt.Errorf("report is not CoverReportImpl")
	}

	// 更新元数据
	rImpl.Meta.BaseCommit = meta.BaseCommit

	// 遍历所有文件节点，重新根据 diff 计算增量信息
	visitor := &rebaseVisitor{
		rep:     rep,
		diffMap: diff,
	}
	rImpl.Tree.Accept(visitor)

	return visitor.err
}

type rebaseVisitor struct {
	rep     report.CoverReport
	diffMap map[string]*diff.DiffFile
	err     error
}

func (v *rebaseVisitor) VisitDirEnter(dir *tree.DirNode) {}
func (v *rebaseVisitor) VisitDirExit(dir *tree.DirNode)  {}
func (v *rebaseVisitor) VisitFile(file *tree.FileNode) {
	if v.err != nil {
		return
	}

	path := file.Path()
	uintLines, err := v.rep.GetFileCoverLines(path)
	if err != nil {
		v.err = err
		return
	}
	lines, _ := coder.DecodeUintLines(uintLines)

	diffInfo := report.FileDiffInfo{}
	if dFile, ok := v.diffMap[path]; ok {
		// 从 diffFile 中提取增量行号
		for _, line := range dFile.GetFileChangeLines() {
			diffInfo.AddedLines = append(diffInfo.AddedLines, uint32(line))
		}
		diffInfo.AddLines = dFile.GetAddLinesCount()
		diffInfo.DeleteLines = dFile.GetDeleteLinesCount()
	}

	// 更新文件，重新触发增量统计计算
	v.err = v.rep.UpdateFile(path, lines, diffInfo, 0)
}
