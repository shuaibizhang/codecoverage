package store

import (
	"time"
)

// DiffCache 对应数据库中的 diff_cache 表
type DiffCache struct {
	ID               uint64    `json:"id" ddb:"id"`
	Module           string    `json:"module" ddb:"module"`
	CommitID         string    `json:"commit_id" ddb:"commit_id"`
	BaseCommitID     string    `json:"base_commit_id" ddb:"base_commit_id"`
	DiffPartitionKey string    `json:"diff_partition_key" ddb:"diff_partition_key"`
	CreatedTime      time.Time `json:"created_time" ddb:"_created_time"`
	UpdatedTime      time.Time `json:"updated_time" ddb:"_updated_time"`
	Deleted          int8      `json:"deleted" ddb:"_deleted"`
}

// TableName 指定映射的表名
func (DiffCache) TableName() string {
	return "diff_cache"
}
