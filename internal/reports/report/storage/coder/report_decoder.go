package coder

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/shuaibizhang/codecoverage/internal/reports/report"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/tree"
)

const indentSize = 2

type ReportDecoder interface {
	// 将输入流解码并填充到指定的 CoverReport 中
	Decode(rep report.CoverReport) error
}

// reportDecoder 覆盖率数据文件解析
type reportDecoder struct {
	input          io.Reader
	data           *report.CoverReportImpl
	scan           *bufio.Scanner
	treeNodeParser func(s string, data *tree.TreeNodeData) error
	lineNo         int
	nextSection    bool
	metaParsed     bool
	treeParsed     bool
	peekedLine     []byte // 用于预读一行
}

func NewReportDecoder(r io.Reader) ReportDecoder {
	return &reportDecoder{
		input: r,
	}
}

// Decode 解析覆盖率数据文件
func (d *reportDecoder) Decode(rep report.CoverReport) error {
	impl, ok := rep.(*report.CoverReportImpl)
	if !ok {
		return fmt.Errorf("expected *report.CoverReportImpl, got %T", rep)
	}
	d.data = impl

	// 如果 input 是 io.Seeker, 则回到起点
	if seeker, ok := d.input.(io.Seeker); ok {
		seeker.Seek(0, io.SeekStart)
	}

	d.scan = bufio.NewScanner(d.input)
	d.scan.Split(bufio.ScanLines)
	d.nextSection = false
	d.metaParsed = false
	d.treeParsed = false
	d.lineNo = 0
	d.peekedLine = nil
	err := d.decode()
	if err != nil {
		return err
	}
	if !d.metaParsed {
		return errors.New("meta section missed")
	}
	if !d.treeParsed {
		return errors.New("tree section missed")
	}
	return nil
}

// decode 实现解析逻辑
func (d *reportDecoder) decode() error {
	var (
		err  error
		line []byte
	)
	for {
		if len(line) <= 0 {
			line, err = d.nextLine()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return fmt.Errorf("line %d: %w", d.lineNo, err)
			}
		}
		line, err = d.parseLine(line)
		if err != nil {
			return fmt.Errorf("line %d: %w", d.lineNo, err)
		}
	}
	return nil
}

// nextLine 读取下一行数据
func (d *reportDecoder) nextLine() ([]byte, error) {
	if d.peekedLine != nil {
		line := d.peekedLine
		d.peekedLine = nil
		return line, nil
	}

	s := d.scan
	for s.Scan() {
		d.lineNo++
		text := s.Bytes()
		text = bytes.TrimSpace(text)
		if len(text) <= 0 {
			continue
		}

		if text[0] == '#' {
			continue
		}
		// 返回副本，因为 s.Bytes() 会在下次 Scan 时被覆盖
		res := make([]byte, len(text))
		copy(res, text)
		return res, nil
	}
	if err := s.Err(); err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	return nil, io.EOF
}

// peekLine 预读下一行
func (d *reportDecoder) peekLine() ([]byte, error) {
	if d.peekedLine != nil {
		return d.peekedLine, nil
	}
	line, err := d.nextLine()
	if err != nil {
		return nil, err
	}
	d.peekedLine = line
	return d.peekedLine, nil
}

// parseLine 解析一行数据
func (d *reportDecoder) parseLine(text []byte) ([]byte, error) {
	if d.hasSectionPrefix(text) {
		d.nextSection = false
		return d.parseSection(text)
	}
	if d.nextSection {
		return nil, nil
	}
	return nil, fmt.Errorf("unrecognized text: %s", text)
}

// hasSectionPrefix 是否包含段前缀
func (d *reportDecoder) hasSectionPrefix(text []byte) bool {
	if len(text) <= 0 {
		return false
	}
	return text[0] == '@'
}

// parseSection 解析指定段
func (d *reportDecoder) parseSection(text []byte) ([]byte, error) {
	var (
		err error
		tag = string(text[1:])
	)
	switch tag {
	case "meta":
		d.metaParsed = true
		text, err = d.parseMeta(&d.data.Meta)
	case "tree":
		d.treeParsed = true
		text, err = d.parseTree()
	default:
		// skip unknown section
		d.nextSection = true
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", tag, err)
	}
	return text, nil
}

// parseMeta 解析文件元数据
func (d *reportDecoder) parseMeta(meta *report.MetaInfo) ([]byte, error) {
	for {
		text, err := d.nextLine()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil, nil
			}
			return nil, err
		}
		// see next section
		if d.hasSectionPrefix(text) {
			return text, nil
		}
		strs := bytes.SplitN(text, []byte(":"), 2)
		if len(strs) < 2 {
			continue
		}
		key := strings.TrimSpace(string(strs[0]))
		val := strings.TrimSpace(string(strs[1]))
		err = d.updateMetaInfo(key, val, meta)
		if err != nil {
			return nil, err
		}
	}
}

func (d *reportDecoder) updateMetaInfo(key, val string, meta *report.MetaInfo) error {
	switch key {
	case "module":
		meta.Module = val
	case "branch":
		meta.Branch = val
	case "commit":
		meta.Commit = val
	case "base_commit":
		meta.BaseCommit = val
	case "filter_version":
		v, _ := strconv.Atoi(val)
		meta.FilterVersion = uint32(v)
	case "total_files":
		v, _ := strconv.Atoi(val)
		meta.TotalFiles = uint32(v)
	case "inherit_info":
		meta.InheritInfo = val
	case "last_update":
		meta.LastUpdate = val
	case "columns":
		meta.Columns = strings.Split(val, ",")
		for i := range meta.Columns {
			meta.Columns[i] = strings.TrimSpace(meta.Columns[i])
		}
	}
	return nil
}

// parseTree 解析目录树
func (d *reportDecoder) parseTree() ([]byte, error) {
	var (
		parents = make([]*tree.DirNode, 0, 8)
	)

	for {
		line, err := d.nextLine()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil, nil
			}
			return nil, err
		}
		if d.hasSectionPrefix(line) {
			return line, nil
		}
		text := string(line)
		strs := strings.SplitN(text, ":", 2)
		if len(strs) < 2 {
			return nil, fmt.Errorf("tree node must split with `:`: %s", text)
		}

		nodeName := strings.TrimSpace(strs[0])
		if nodeName == "*" || nodeName == "*/" {
			// 根节点必然是目录
			root := tree.NewDirNode("*", "*")
			// 注意：DirNode.GetStat() 会通过访问者模式重新计算统计数据
			// 这里我们直接将解析出的数据存入 root.stat (需要修改 tree 包以支持，或通过其他方式)
			// 暂时先解析到临时的 data 中
			var rootData tree.TreeNodeData
			_, _, err := d.parseTreeNodeData(&d.data.Meta, strs[1], &rootData)
			if err != nil {
				return nil, fmt.Errorf("parse root node data (%s): %w", text, err)
			}
			// 既然 tree.TreeNode 接口没提供 SetStat，我们暂时保留这个行为
			d.data.Tree = root
			parents = append(parents, root)
			continue
		}

		name, indent, err := d.parseTreeNodeName(strs[0])
		if err != nil {
			return nil, fmt.Errorf("parse filename (%s): %w", strs[0], err)
		}

		isDir := false
		if strings.HasSuffix(name, "/") {
			isDir = true
			name = strings.TrimSuffix(name, "/")
		}

		var nodeData tree.TreeNodeData
		blockOffset, fileFlags, err := d.parseTreeNodeData(&d.data.Meta, strs[1], &nodeData)
		if err != nil {
			return nil, fmt.Errorf("parse node data (%s): %w", text, err)
		}

		// 预读下一行来判断当前节点是目录还是文件 (作为兜底)
		if !isDir {
			nextLine, peekErr := d.peekLine()
			if peekErr == nil && !d.hasSectionPrefix(nextLine) {
				_, nextIndent, _ := d.parseTreeNodeName(string(nextLine))
				if nextIndent > indent {
					isDir = true
				}
			}
		}

		// 计算父节点
		// indent 每次增加 indentSize (2)
		// 根节点的 indent 是 0 (对于 "*" 来说)，但子节点 "--" indent 是 2
		// 栈中的 parents[0] 是 root (indent 0)
		// parents[1] 是 indent 2 的目录，以此类推
		level := indent / indentSize
		// 确保 parents 长度与当前 level 匹配
		// 如果 level 是 1，我们需要 parents[0] 作为父节点，此时 parents 长度应为 1
		// 如果 level 是 2，我们需要 parents[1] 作为父节点，此时 parents 长度应为 2
		if level < len(parents) {
			parents = parents[:level]
		} else if level > len(parents) {
			// 这种情况下 test.cno 缩进跳跃了，理论上不应该发生
			// 但为了防止 panic，我们不做处理，直接使用当前的最后一个 parent
		}

		if len(parents) == 0 {
			return nil, fmt.Errorf("no parent for node %s at indent %d (line %d)", name, indent, d.lineNo)
		}
		parent := parents[len(parents)-1]

		var newNode tree.TreeNode
		fullPath := parent.Path() + "/" + name
		if parent.Path() == "*" {
			fullPath = name
		}

		if isDir {
			dirNode := tree.NewDirNode(name, fullPath)
			// 注意：DirNode 的 Stat 是通过访问者模式聚合的，但 test.cno 中提供了预计算的值
			// 如果我们想要保留 test.cno 中的原始值，可能需要直接设置。
			// 但目前 DirNode.GetStat() 是动态计算的。这里先创建一个 DirNode。
			newNode = dirNode
			parent.Add(newNode)
			parents = append(parents, dirNode)
		} else {
			fileNode := tree.NewFileNode(name, fullPath, nodeData)
			fileNode.BlockOffset = blockOffset
			fileNode.FileFlags = fileFlags
			newNode = fileNode
			parent.Add(newNode)
		}
	}
}

func (d *reportDecoder) parseTreeNodeName(text string) (string, int, error) {
	var indent int
	for i := 0; i < len(text); i++ {
		if text[i] != '-' {
			return strings.TrimSpace(text[i:]), indent, nil
		}
		indent++
	}
	return "", 0, errors.New("empty text")
}

func (d *reportDecoder) parseTreeNodeData(meta *report.MetaInfo, text string, data *tree.TreeNodeData) (int64, uint32, error) {
	parts := strings.Split(text, ",")
	if len(parts) < 11 {
		return 0, 0, fmt.Errorf("invalid node data: %s", text)
	}

	u32 := func(s string) uint32 {
		v, _ := strconv.ParseUint(strings.TrimSpace(s), 10, 32)
		return uint32(v)
	}

	blockOffset, _ := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	fileFlags := u32(parts[1])

	data.TotalLines = u32(parts[2])
	data.InstrLines = u32(parts[3])
	data.AddLines = u32(parts[4])
	data.DeleteLines = u32(parts[5])
	data.IncrInstrLines = u32(parts[6])
	data.CoverLines = u32(parts[7])
	data.Coverage = u32(parts[8])
	data.IncrCoverLines = u32(parts[9])
	data.IncrCoverage = u32(parts[10])

	if len(parts) >= 12 {
		data.HasIncrement = strings.TrimSpace(parts[11]) == "1"
	} else {
		// 向后兼容：如果不存在该列，根据 IncrInstrLines 判断
		data.HasIncrement = data.IncrInstrLines > 0
	}

	return blockOffset, fileFlags, nil
}
