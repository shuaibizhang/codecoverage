package store

import (
	"time"
)

var zeroTime = time.Date(1000, 1, 1, 0, 0, 0, 0, time.Local)

func ZeroTime() time.Time {
	return zeroTime
}

// DeleteAutoColumns 删除由数据库自动更新的字段
func DeleteAutoColumns(data map[string]interface{}) {
	delete(data, "id")
	delete(data, "_created_time")
	delete(data, "_updated_time")
	delete(data, "_deleted")
}
