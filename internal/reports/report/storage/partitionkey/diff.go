package partitionkey

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// diffKey 用于定位序列化后的 gitDiffMap JSON 文件
type diffKey struct {
	Module     string `json:"module"`
	Branch     string `json:"branch"`
	Commit     string `json:"commit"`
	BaseCommit string `json:"base_commit"`
	Timestamp  int64  `json:"timestamp"`
	Path       string `json:"path"`
}

// NewDiffKey 创建 diff 寻址句柄
func NewDiffKey(module, branch, commit, baseCommit string) PartitionKey {
	return &diffKey{
		Module:     module,
		Branch:     branch,
		Commit:     commit,
		BaseCommit: baseCommit,
		Timestamp:  time.Now().Unix(),
	}
}

func (k *diffKey) Type() PartitionType { return DiffType }

func (k *diffKey) Marshal() (string, error) {
	data, err := json.Marshal(k)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (k *diffKey) Unmarshal(data string) error {
	return json.Unmarshal([]byte(data), k)
}

func (k *diffKey) RealPathPrefix() string {
	if k.Path != "" {
		return k.Path
	}

	dateStr := time.Unix(k.Timestamp, 0).Format(time.DateOnly)
	safeModule := strings.ReplaceAll(k.Module, "/", "_")
	safeBranch := strings.ReplaceAll(k.Branch, "/", "_")

	shortCommit := k.Commit
	if len(shortCommit) > 8 {
		shortCommit = shortCommit[:8]
	}
	shortBaseCommit := k.BaseCommit
	if len(shortBaseCommit) > 8 {
		shortBaseCommit = shortBaseCommit[:8]
	}

	// 格式：/diff/日期分片/模块/模块_分支_commit[:8]_baseCommit[:8]
	filename := fmt.Sprintf("%s_%s_%s_%s", safeModule, safeBranch, shortCommit, shortBaseCommit)
	k.Path = filepath.Join(string(DiffType), dateStr, safeModule, filename)
	return k.Path
}

func (k *diffKey) Offset() int64     { return 0 }
func (k *diffKey) SetOffset(o int64) {}

func (k *diffKey) GetModule() string { return k.Module }
func (k *diffKey) GetBranch() string { return k.Branch }
func (k *diffKey) GetCommit() string { return k.Commit }
