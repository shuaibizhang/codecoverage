package partitionkey

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// normalCovKey 归一化行覆盖率数据句柄，用于定位 .json/.diff 文件
type normalCovKey struct {
	TType  TestType `json:"test_type"`
	Module string   `json:"module"`
	Branch string   `json:"branch"`
	Commit string   `json:"commit"`

	// 单元测试独有
	UnittestRunID string `json:"unittest_run_id,omitempty"`

	Timestamp int64 `json:"timestamp"`

	Path   string `json:"path"` // .json/.diff 相对路径
	offset int64  // 偏移量
}

// NewNormalCovKey 创建正常覆盖率数据寻址句柄
func NewUnitTestNormalCovKey(module, branch, commit string, unittestRunID string) PartitionKey {
	return &normalCovKey{
		TType:         UnitTest,
		Module:        module,
		Branch:        branch,
		Commit:        commit,
		UnittestRunID: unittestRunID,
	}
}

func (k *normalCovKey) Type() PartitionType { return NormalCovType }

func (k *normalCovKey) Marshal() (string, error) {
	data, err := json.Marshal(k)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (k *normalCovKey) Unmarshal(data string) error {
	return json.Unmarshal([]byte(data), k)
}

// RealPathPrefix 根据规则生成路径前缀，不带扩展名
func (k *normalCovKey) RealPathPrefix() string {
	if k.Path != "" {
		return k.Path
	}

	dateStr := time.Unix(k.Timestamp, 0).Format(time.DateOnly)
	safeModule := strings.ReplaceAll(k.Module, "/", "_")
	safeBranch := strings.ReplaceAll(k.Branch, "/", "_")
	prefix := filepath.Join(string(k.TType), dateStr, safeModule)

	var filename string
	if k.TType == AutoTest {
		// 自动化测试：测试计划id_执行id
	} else {
		// 集成、单元、线上：模块_分支_commit[:8]_时间戳
		shortCommit := k.Commit
		if len(shortCommit) > 8 {
			shortCommit = shortCommit[:8]
		}
		if k.UnittestRunID != "" {
			filename = fmt.Sprintf("%s_%s_%s", safeBranch, shortCommit, k.UnittestRunID)
		} else {
			filename = fmt.Sprintf("%s_%s_%d", safeBranch, shortCommit, k.Timestamp)
		}
	}

	k.Path = filepath.Join(prefix, filename)
	return k.Path
}

func (k *normalCovKey) Offset() int64     { return k.offset }
func (k *normalCovKey) SetOffset(o int64) { k.offset = o }
