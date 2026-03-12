package coverage

import (
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/runtime/protoimpl"
)

type GetReportInfoRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Module string `protobuf:"bytes,1,opt,name=module,proto3" json:"module,omitempty"`
	Branch string `protobuf:"bytes,2,opt,name=branch,proto3" json:"branch,omitempty"`
	Commit string `protobuf:"bytes,3,opt,name=commit,proto3" json:"commit,omitempty"`
	Type   string `protobuf:"bytes,4,opt,name=type,proto3" json:"type,omitempty"`
}

func (x *GetReportInfoRequest) ProtoReflect() protoreflect.Message { return nil }
func (x *GetReportInfoRequest) Reset()                             {}
func (x *GetReportInfoRequest) String() string                     { return "" }

type MetaInfo struct {
	state      protoimpl.MessageState
	Module     string `json:"module,omitempty"`
	Branch     string `json:"branch,omitempty"`
	Commit     string `json:"commit,omitempty"`
	TotalFiles uint32 `json:"total_files,omitempty"`
	Coverage   uint32 `json:"coverage,omitempty"`
	LastUpdate string `json:"last_update,omitempty"`
}

func (x *MetaInfo) ProtoReflect() protoreflect.Message { return nil }
func (x *MetaInfo) Reset()                             {}
func (x *MetaInfo) String() string                     { return "" }

type GetReportInfoResponse struct {
	state    protoimpl.MessageState
	ReportId string    `json:"report_id,omitempty"`
	Meta     *MetaInfo `json:"meta,omitempty"`
}

func (x *GetReportInfoResponse) ProtoReflect() protoreflect.Message { return nil }
func (x *GetReportInfoResponse) Reset()                             {}
func (x *GetReportInfoResponse) String() string                     { return "" }

type GetTreeNodesRequest struct {
	state    protoimpl.MessageState
	ReportId string `json:"report_id,omitempty"`
	Path     string `json:"path,omitempty"`
}

func (x *GetTreeNodesRequest) ProtoReflect() protoreflect.Message { return nil }
func (x *GetTreeNodesRequest) Reset()                             {}
func (x *GetTreeNodesRequest) String() string                     { return "" }

type TreeNodeStat struct {
	state          protoimpl.MessageState
	TotalLines     uint32 `json:"total_lines,omitempty"`
	InstrLines     uint32 `json:"instr_lines,omitempty"`
	CoverLines     uint32 `json:"cover_lines,omitempty"`
	Coverage       uint32 `json:"coverage,omitempty"`
	AddLines       uint32 `json:"add_lines,omitempty"`
	DeleteLines    uint32 `json:"delete_lines,omitempty"`
	IncrInstrLines uint32 `json:"incr_instr_lines,omitempty"`
	IncrCoverLines uint32 `json:"incr_cover_lines,omitempty"`
	IncrCoverage   uint32 `json:"incr_coverage,omitempty"`
}

func (x *TreeNodeStat) ProtoReflect() protoreflect.Message { return nil }
func (x *TreeNodeStat) Reset()                             {}
func (x *TreeNodeStat) String() string                     { return "" }

type TreeNode struct {
	state protoimpl.MessageState
	Name  string        `json:"name,omitempty"`
	Path  string        `json:"path,omitempty"`
	Type  int32         `json:"type,omitempty"`
	Stat  *TreeNodeStat `json:"stat,omitempty"`
}

func (x *TreeNode) ProtoReflect() protoreflect.Message { return nil }
func (x *TreeNode) Reset()                             {}
func (x *TreeNode) String() string                     { return "" }

type GetTreeNodesResponse struct {
	state protoimpl.MessageState
	Nodes []*TreeNode `json:"nodes,omitempty"`
}

func (x *GetTreeNodesResponse) ProtoReflect() protoreflect.Message { return nil }
func (x *GetTreeNodesResponse) Reset()                             {}
func (x *GetTreeNodesResponse) String() string                     { return "" }

type GetFileCoverageRequest struct {
	state    protoimpl.MessageState
	ReportId string `json:"report_id,omitempty"`
	Path     string `json:"path,omitempty"`
}

func (x *GetFileCoverageRequest) ProtoReflect() protoreflect.Message { return nil }
func (x *GetFileCoverageRequest) Reset()                             {}
func (x *GetFileCoverageRequest) String() string                     { return "" }

type GetFileCoverageResponse struct {
	state     protoimpl.MessageState
	Path      string   `json:"path,omitempty"`
	Lines     []uint32 `json:"lines,omitempty"`
	FileFlags uint32   `json:"file_flags,omitempty"`
	Content   string   `json:"content,omitempty"`
}

func (x *GetFileCoverageResponse) ProtoReflect() protoreflect.Message { return nil }
func (x *GetFileCoverageResponse) Reset()                             {}
func (x *GetFileCoverageResponse) String() string                     { return "" }
