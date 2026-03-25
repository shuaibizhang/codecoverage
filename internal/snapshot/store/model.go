package store

import (
	"time"
)

// SnapshotInfo 对应数据库中的 snapshot_info 表
type SnapshotInfo struct {
	ID                 uint64    `json:"id" ddb:"id"`
	Module             string    `json:"module" ddb:"module"`
	Branch             string    `json:"branch" ddb:"branch"`
	Commit             string    `json:"commit" ddb:"commit"`
	BaseCommit         string    `json:"base_commit" ddb:"base_commit"`
	SnapshotID         string    `json:"snapshot_id" ddb:"snapshot_id"`
	ReportPartitionKey string    `json:"report_partition_key" ddb:"report_partition_key"`
	CreatedTime        time.Time `json:"created_time" ddb:"_created_time"`
	UpdatedTime        time.Time `json:"updated_time" ddb:"_updated_time"`
	Deleted            int8      `json:"deleted" ddb:"_deleted"`
}

// TableName 指定映射的表名
func (SnapshotInfo) TableName() string {
	return "snapshot_info"
}
