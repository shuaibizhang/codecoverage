package store

import "time"

// SystestTask 对应数据库中的 systest_task 表
type SystestTask struct {
	ID                          uint64    `json:"id" ddb:"id"`
	Language                    string    `json:"language" ddb:"language"`
	Module                      string    `json:"module" ddb:"module"`
	Branch                      string    `json:"branch" ddb:"branch"`
	Commit                      string    `json:"commit" ddb:"commit"`
	CommitCreateTime            time.Time `json:"commit_create_time" ddb:"commit_create_time"`
	BaseCommit                  string    `json:"base_commit" ddb:"base_commit"`
	Status                      string    `json:"status" ddb:"status"`
	InheritCommit               string    `json:"inherit_commit" ddb:"inherit_commit"`
	InheritStatus               string    `json:"inherit_status" ddb:"inherit_status"`
	InheritLog                  string    `json:"inherit_log" ddb:"inherit_log"`
	NormalCoverDataPartitionKey string    `json:"normal_cover_data_partition_key" ddb:"normal_cover_data_partition_key"`
	ReportPartitionKey          string    `json:"report_partition_key" ddb:"report_partition_key"`
	CreatedTime                 time.Time `json:"created_time" ddb:"_created_time"`
	UpdatedTime                 time.Time `json:"updated_time" ddb:"_updated_time"`
	Deleted                     int8      `json:"deleted" ddb:"_deleted"`
}

// TableName 指定映射的表名
func (SystestTask) TableName() string {
	return "systest_task"
}
