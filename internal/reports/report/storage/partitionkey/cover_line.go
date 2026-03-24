package partitionkey

// coverageKey 物理位置寻址句柄，用于定位 .cda 文件中的特定块
type coverageKey struct {
	Path   string `json:"path"` // .cda 文件路径
	offset int64  // 块起始偏移量 (block_offset)
}

func NewCoverageKey(prefix string, offset int64) PartitionKey {
	return &coverageKey{
		Path:   prefix,
		offset: offset,
	}
}

func (k *coverageKey) Type() PartitionType { return CoverageDataType }

func (k *coverageKey) Marshal() (string, error) {
	return "", nil
}

func (k *coverageKey) Unmarshal(data string) error {
	return nil
}

func (k *coverageKey) RealPathPrefix() string { return k.Path }
func (k *coverageKey) Offset() int64          { return k.offset }
func (k *coverageKey) SetOffset(o int64)      { k.offset = o }

func (k *coverageKey) GetModule() string { return "" }
func (k *coverageKey) GetBranch() string { return "" }
func (k *coverageKey) GetCommit() string { return "" }
