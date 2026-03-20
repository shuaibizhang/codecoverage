package partitionkey

// 分区类型
type PartitionType string

const (
	ReportType       PartitionType = "coverage_report"
	CoverageDataType PartitionType = "coverage_data"
	NormalCovType    PartitionType = "normal_cover_data"
)

// 测试类型
type TestType string

const (
	UnitTest      TestType = "unittest"
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
	// RealPathPrefix 获取实际存储路径前缀（不带扩展名，如 .cno 或 .cda）
	RealPathPrefix() string
	// Offset 获取在该路径下的偏移量（用于 .cda 随机访问）
	Offset() int64
	SetOffset(offset int64)
}
