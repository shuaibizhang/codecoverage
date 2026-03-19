package store

import (
	"time"
)

// UnittestTask 对应数据库中的 unittest_task 表
type UnittestTask struct {
	ID                          uint64    `json:"id" ddb:"id"`
	Language                    string    `json:"language" ddb:"language"`
	Module                      string    `json:"module" ddb:"module"`
	Branch                      string    `json:"branch" ddb:"branch"`
	Commit                      string    `json:"commit" ddb:"commit"`
	BaseCommit                  string    `json:"base_commit" ddb:"base_commit"`
	RunID                       string    `json:"run_id" ddb:"run_id"`
	Status                      string    `json:"status" ddb:"status"`
	NormalCoverDataPartitionKey string    `json:"normal_cover_data_partition_key" ddb:"normal_cover_data_partition_key"`
	ReportPartitionKey          string    `json:"report_partition_key" ddb:"report_partition_key"`
	CreatedTime                 time.Time `json:"created_time" ddb:"_created_time"`
	UpdatedTime                 time.Time `json:"updated_time" ddb:"_updated_time"`
	Deleted                     int8      `json:"deleted" ddb:"_deleted"`
}

// TableName 指定映射的表名
func (UnittestTask) TableName() string {
	return "unittest_task"
}
