package partitionkey

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// 分区类型
type PartitionType string

const (
	ReportType       PartitionType = "report"
	CoverageDataType PartitionType = "coverage_data"
)

// 测试类型
type TestType string

const (
	UnitTest      TestType = "unit"
	IntegrateTest TestType = "integrate"
	OnlineTest    TestType = "online"
	AutoTest      TestType = "auto"
)

// PartitionKey 分区接口，既是报告的寻址句柄，也是磁盘物理位置的寻址句柄
type PartitionKey interface {
	// 返回分区类型
	Type() PartitionType
	// 序列化、反序列化支持，方便存储
	Marshal() (string, error)
	Unmarshal(partitionKey string) error
	// RealPath 获取实际存储路径（如 .cno 或 .cda 文件路径）
	RealPath() string
	// Offset 获取在该路径下的偏移量（用于 .cda 随机访问）
	Offset() int64
	SetOffset(offset int64)
}

// reportKey 报告寻址句柄，用于定位 .cno、.cda 文件
type reportKey struct {
	TType  TestType `json:"test_type"`
	Module string   `json:"module"`
	Branch string   `json:"branch"`
	Commit string   `json:"commit"`

	// 自动化测试特有
	PlanID uint64 `json:"plan_id,omitempty"`
	ExecID uint64 `json:"exec_id,omitempty"`

	Timestamp int64 `json:"timestamp"`

	Path   string `json:"path"` // .cno 相对路径
	offset int64  // 报告文件通常从 0 开始
}

// NewReportKey 创建报告寻址句柄，用于定位 .cno、.cda 文件
func NewReportKey(ttype TestType, module, branch, commit string) PartitionKey {
	return &reportKey{
		TType:     ttype,
		Module:    module,
		Branch:    branch,
		Commit:    commit,
		Timestamp: time.Now().Unix(),
	}
}

// NewAutoReportKey 创建自动化测试报告寻址句柄，用于定位 .cno 文件
func NewAutoReportKey(module, branch, commit string, planID, execID uint64) PartitionKey {
	return &reportKey{
		TType:     AutoTest,
		Module:    module,
		Branch:    branch,
		Commit:    commit,
		PlanID:    planID,
		ExecID:    execID,
		Timestamp: time.Now().Unix(),
	}
}

func (k *reportKey) Type() PartitionType { return ReportType }

func (k *reportKey) Marshal() (string, error) {
	data, err := json.Marshal(k)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (k *reportKey) Unmarshal(data string) error {
	return json.Unmarshal([]byte(data), k)
}

// RealPath 根据规则生成 .cno 路径
func (k *reportKey) RealPath() string {
	if k.Path != "" {
		return k.Path
	}

	dateStr := time.Unix(k.Timestamp, 0).Format(time.DateOnly)
	safeModule := strings.ReplaceAll(k.Module, "/", "_")
	safeBranch := strings.ReplaceAll(k.Branch, "/", "_")
	prefix := filepath.Join(string(k.TType), dateStr, safeModule)

	var filename string
	if k.TType == AutoTest {
		// 自动化测试：测试计划id_执行id.cno
		filename = fmt.Sprintf("%d_%d.cno", k.PlanID, k.ExecID)
	} else {
		// 集成、单元、线上：模块_分支_commit[:8]_时间戳.cno
		shortCommit := k.Commit
		if len(shortCommit) > 8 {
			shortCommit = shortCommit[:8]
		}
		// 如果 Module 或 Branch 包含 /，替换为 _ 以免生成多级子目录
		filename = fmt.Sprintf("%s_%s_%d.cno", safeBranch, shortCommit, k.Timestamp)
	}

	k.Path = filepath.Join(prefix, filename)
	return k.Path
}

func (k *reportKey) Offset() int64     { return k.offset }
func (k *reportKey) SetOffset(o int64) { k.offset = o }

// coverageKey 物理位置寻址句柄，用于定位 .cda 文件中的特定块
type coverageKey struct {
	Path   string `json:"path"` // .cda 文件路径
	offset int64  // 块起始偏移量 (block_offset)
}

func NewCoverageKey(cnoPath string, offset int64) PartitionKey {
	// .cda 路径通常与 .cno 同名同路径，只是后缀不同
	cdaPath := cnoPath
	if filepath.Ext(cnoPath) == ".cno" {
		cdaPath = cnoPath[:len(cnoPath)-len(".cno")] + ".cda"
	}
	return &coverageKey{
		Path:   cdaPath,
		offset: offset,
	}
}

func (k *coverageKey) Type() PartitionType { return CoverageDataType }

func (k *coverageKey) Marshal() (string, error) {
	return "", nil
}

func (k *coverageKey) Unmarshal(data string) error {
	return nil
}

func (k *coverageKey) RealPath() string  { return k.Path }
func (k *coverageKey) Offset() int64     { return k.offset }
func (k *coverageKey) SetOffset(o int64) { k.offset = o }
