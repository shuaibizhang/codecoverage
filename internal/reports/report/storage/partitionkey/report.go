package partitionkey

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

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
	if ttype == Snapshot {
		return NewSnapshotReportKey(module, branch, commit)
	}
	return &reportKey{
		TType:     ttype,
		Module:    module,
		Branch:    branch,
		Commit:    commit,
		Timestamp: time.Now().Unix(),
	}
}

// NewSystestReportKey 创建系统测试报告寻址句柄
func NewSystestReportKey(module, branch, commit string) PartitionKey {
	return &reportKey{
		TType:     Systest,
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

// RealPathPrefix 根据规则生成路径前缀（不带 .cno 扩展名）
func (k *reportKey) RealPathPrefix() string {
	if k.Path != "" {
		return k.Path
	}

	dateStr := time.Unix(k.Timestamp, 0).Format(time.DateOnly)
	safeModule := strings.ReplaceAll(k.Module, "/", "_")
	safeBranch := strings.ReplaceAll(k.Branch, "/", "_")
	prefix := filepath.Join(string(k.TType), dateStr, string(ReportType), safeModule)

	var filename string
	if k.TType == AutoTest {
		// 自动化测试：测试计划id_执行id
		filename = fmt.Sprintf("%d_%d", k.PlanID, k.ExecID)
	} else {
		// 集成、单元、线上：模块_分支_commit[:8]_时间戳
		shortCommit := k.Commit
		if len(shortCommit) > 8 {
			shortCommit = shortCommit[:8]
		}
		// 如果 Module 或 Branch 包含 /，替换为 _ 以免生成多级子目录
		filename = fmt.Sprintf("%s_%s_%d", safeBranch, shortCommit, k.Timestamp)
	}

	k.Path = filepath.Join(prefix, filename)
	return k.Path
}

func (k *reportKey) Offset() int64     { return k.offset }
func (k *reportKey) SetOffset(o int64) { k.offset = o }

func (k *reportKey) GetModule() string { return k.Module }
func (k *reportKey) GetBranch() string { return k.Branch }
func (k *reportKey) GetCommit() string { return k.Commit }
