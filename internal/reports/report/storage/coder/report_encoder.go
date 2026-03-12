package coder

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/shuaibizhang/codecoverage/internal/reports/report"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/tree"
)

type ReportEncoder interface {
	Encode(report report.CoverReport) error
}

var treeNodeDataColumns = []string{
	"block_offset", "file_flags", "total_lines", "instr_lines", "add_lines", "delete_lines",
	"incr_instr_lines", "cover_lines", "coverage", "incr_cover_lines", "incr_coverage",
}

// reportEncoder 实现了覆盖率报告的编码
type reportEncoder struct {
	output io.Writer
	bufw   *bufio.Writer
}

func NewReportEncoder(w io.Writer) ReportEncoder {
	return &reportEncoder{
		output: w,
	}
}

func (e *reportEncoder) Encode(rep report.CoverReport) error {
	e.bufw = bufio.NewWriter(e.output)
	return e.encode(rep)
}

func (e *reportEncoder) encode(rep report.CoverReport) error {
	if rep == nil {
		return errors.New("report is nil")
	}

	meta := rep.GetMeta()
	// 总是升级到新的columns
	meta.Columns = treeNodeDataColumns

	err := e.encodeMeta(&meta)
	if err != nil {
		return fmt.Errorf("encode meta: %w", err)
	}

	// 获取树根节点
	impl, ok := rep.(*report.CoverReportImpl)
	if !ok {
		return fmt.Errorf("expected *report.CoverReportImpl, got %T", rep)
	}

	err = e.encodeTree(impl.Tree, meta.Columns)
	if err != nil {
		return fmt.Errorf("encode tree: %w", err)
	}
	return e.bufw.Flush()
}

func (e *reportEncoder) encodeMeta(meta *report.MetaInfo) error {
	w := e.bufw
	w.WriteString("@meta\n")

	w.WriteString("module: ")
	w.WriteString(meta.Module)
	w.WriteString("\n")

	w.WriteString("branch: ")
	w.WriteString(meta.Branch)
	w.WriteString("\n")

	w.WriteString("commit: ")
	w.WriteString(meta.Commit)
	w.WriteString("\n")

	w.WriteString("base_commit: ")
	w.WriteString(meta.BaseCommit)
	w.WriteString("\n")

	w.WriteString("filter_version: ")
	w.WriteString(strconv.FormatUint(uint64(meta.FilterVersion), 10))
	w.WriteString("\n")

	w.WriteString("total_files: ")
	w.WriteString(strconv.FormatUint(uint64(meta.TotalFiles), 10))
	w.WriteString("\n")

	w.WriteString("inherit_info: ")
	w.WriteString(meta.InheritInfo)
	w.WriteString("\n")

	w.WriteString("last_update: ")
	w.WriteString(meta.LastUpdate)
	w.WriteString("\n")

	w.WriteString("columns: ")
	w.WriteString(strings.Join(meta.Columns, ","))
	w.WriteString("\n")

	w.WriteString("\n")
	return nil
}

func (e *reportEncoder) encodeTree(root tree.TreeNode, columns []string) error {
	w := e.bufw
	w.WriteString("@tree\n")

	// 根节点特殊处理
	err := e.encodeTreeNode(0, root, columns)
	if err != nil {
		return fmt.Errorf("encode node: %w", err)
	}
	w.WriteString("\n")
	return nil
}

func (e *reportEncoder) encodeTreeNode(indent int, node tree.TreeNode, columns []string) error {
	w := e.bufw
	for i := 0; i < indent; i++ {
		w.WriteByte('-')
	}
	// 根节点不需要缩进后的空格
	if indent > 0 {
		w.WriteByte(' ')
	}
	w.WriteString(node.Name())
	if node.IsDir() {
		w.WriteByte('/')
	}
	w.WriteString(": ")

	err := e.encodeNodeData(w, node, columns)
	if err != nil {
		return err
	}
	w.WriteString("\n")

	for child := range node.Children() {
		err := e.encodeTreeNode(indent+indentSize, child, columns)
		if err != nil {
			return fmt.Errorf("encode node (%s): %w", child.Name(), err)
		}
	}
	return nil
}

func (e *reportEncoder) encodeNodeData(w *bufio.Writer, node tree.TreeNode, columns []string) error {
	data := node.GetStat()
	vals := make([]string, 0, len(columns))
	for _, col := range columns {
		switch col {
		case "block_offset":
			offset := int64(0)
			if fn, ok := node.(*tree.FileNode); ok {
				offset = fn.BlockOffset
			}
			vals = append(vals, strconv.FormatInt(offset, 10))
		case "file_flags":
			flags := uint32(0)
			if fn, ok := node.(*tree.FileNode); ok {
				flags = fn.FileFlags
			}
			vals = append(vals, strconv.FormatUint(uint64(flags), 10))
		case "total_lines":
			vals = append(vals, strconv.FormatUint(uint64(data.TotalLines), 10))
		case "instr_lines":
			vals = append(vals, strconv.FormatUint(uint64(data.InstrLines), 10))
		case "add_lines":
			vals = append(vals, strconv.FormatUint(uint64(data.AddLines), 10))
		case "delete_lines":
			vals = append(vals, strconv.FormatUint(uint64(data.DeleteLines), 10))
		case "incr_instr_lines":
			vals = append(vals, strconv.FormatUint(uint64(data.IncrInstrLines), 10))
		case "cover_lines":
			vals = append(vals, strconv.FormatUint(uint64(data.CoverLines), 10))
		case "coverage":
			vals = append(vals, strconv.FormatUint(uint64(data.Coverage), 10))
		case "incr_cover_lines":
			vals = append(vals, strconv.FormatUint(uint64(data.IncrCoverLines), 10))
		case "incr_coverage":
			vals = append(vals, strconv.FormatUint(uint64(data.IncrCoverage), 10))
		}
	}
	w.WriteString(strings.Join(vals, ","))
	return nil
}
