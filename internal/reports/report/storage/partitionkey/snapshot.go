package partitionkey

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// snapshotKey 快照寻址句柄，用于定位 .cno、.cda 文件
type snapshotKey struct {
	TType  TestType `json:"test_type"`
	Module string   `json:"module"`
	Branch string   `json:"branch"`
	Commit string   `json:"commit"`

	Timestamp int64 `json:"timestamp"`

	Path   string `json:"path"` // .cno 相对路径
	offset int64  // 报告文件通常从 0 开始
}

// NewSnapshotReportKey 创建快照报告寻址句柄
func NewSnapshotReportKey(module, branch, commit string) PartitionKey {
	return &snapshotKey{
		TType:     Snapshot,
		Module:    module,
		Branch:    branch,
		Commit:    commit,
		Timestamp: time.Now().Unix(),
	}
}

func (k *snapshotKey) Type() PartitionType { return ReportType }

func (k *snapshotKey) Marshal() (string, error) {
	data, err := json.Marshal(k)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (k *snapshotKey) Unmarshal(data string) error {
	return json.Unmarshal([]byte(data), k)
}

// RealPathPrefix 根据规则生成路径前缀（不带 .cno 扩展名）
func (k *snapshotKey) RealPathPrefix() string {
	if k.Path != "" {
		return k.Path
	}

	dateStr := time.Unix(k.Timestamp, 0).Format(time.DateOnly)
	safeModule := strings.ReplaceAll(k.Module, "/", "_")
	safeBranch := strings.ReplaceAll(k.Branch, "/", "_")
	prefix := filepath.Join(string(k.TType), dateStr, string(ReportType), safeModule)

	shortCommit := k.Commit
	if len(shortCommit) > 8 {
		shortCommit = shortCommit[:8]
	}
	// 快照文件名规则：分支_commit[:8]_时间戳
	filename := fmt.Sprintf("%s_%s_%d", safeBranch, shortCommit, k.Timestamp)

	k.Path = filepath.Join(prefix, filename)
	return k.Path
}

func (k *snapshotKey) Offset() int64     { return k.offset }
func (k *snapshotKey) SetOffset(o int64) { k.offset = o }

func (k *snapshotKey) GetModule() string { return k.Module }
func (k *snapshotKey) GetBranch() string { return k.Branch }
func (k *snapshotKey) GetCommit() string { return k.Commit }
