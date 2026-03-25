package partitionkey

import (
	"encoding/json"
)

// 分区类型
type PartitionType string

const (
	ReportType       PartitionType = "coverage_report"
	CoverageDataType PartitionType = "coverage_data"
	NormalCovType    PartitionType = "normal_cover_data"
	DiffType         PartitionType = "diff_data"
)

// 测试类型
type TestType string

const (
	UnitTest      TestType = "unittest"
	IntegrateTest TestType = "integrate"
	OnlineTest    TestType = "online"
	AutoTest      TestType = "auto"
	Systest       TestType = "systest"
	Snapshot      TestType = "snapshot"
)

// PartitionKey 分区接口，既是报告的寻址句柄，也是磁盘物理位置的寻址句柄
type PartitionKey interface {
	// 返回分区类型
	Type() PartitionType
	// 序列化、反序列化支持，方便存储
	Marshal() (string, error)
	Unmarshal(partitionKey string) error
	// RealPathPrefix 获取实际存储路径前缀（不带扩展名，如 .cno 或 .cda）
	RealPathPrefix() string
	// Offset 获取在该路径下的偏移量（用于 .cda 随机访问）
	Offset() int64
	SetOffset(offset int64)

	// 获取元数据信息
	GetModule() string
	GetBranch() string
	GetCommit() string
}

// UnmarshalPartitionKey 根据 json 中的 test_type 自动识别并反序列化为对应的 PartitionKey 实现
func UnmarshalPartitionKey(data string) (PartitionKey, error) {
	var base struct {
		TType TestType `json:"test_type"`
	}
	if err := json.Unmarshal([]byte(data), &base); err != nil {
		return nil, err
	}

	var pk PartitionKey
	switch base.TType {
	case Snapshot:
		pk = &snapshotKey{}
	case UnitTest, IntegrateTest, OnlineTest, AutoTest, Systest:
		pk = &reportKey{}
	default:
		// 如果没有 test_type，或者 test_type 不匹配，尝试作为普通的 reportKey（兼容旧版本）
		pk = &reportKey{}
	}

	if err := pk.Unmarshal(data); err != nil {
		return nil, err
	}
	return pk, nil
}
