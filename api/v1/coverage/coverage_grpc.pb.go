package coverage

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CoverageServiceClient 是 CoverageService 的客户端接口
type CoverageServiceClient interface {
	GetReportInfo(ctx context.Context, in *GetReportInfoRequest, opts ...grpc.CallOption) (*GetReportInfoResponse, error)
	GetTreeNodes(ctx context.Context, in *GetTreeNodesRequest, opts ...grpc.CallOption) (*GetTreeNodesResponse, error)
	GetFileCoverage(ctx context.Context, in *GetFileCoverageRequest, opts ...grpc.CallOption) (*GetFileCoverageResponse, error)
}

type coverageServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewCoverageServiceClient(cc grpc.ClientConnInterface) CoverageServiceClient {
	return &coverageServiceClient{cc}
}

func (c *coverageServiceClient) GetReportInfo(ctx context.Context, in *GetReportInfoRequest, opts ...grpc.CallOption) (*GetReportInfoResponse, error) {
	out := new(GetReportInfoResponse)
	err := c.cc.Invoke(ctx, "/api.v1.coverage.CoverageService/GetReportInfo", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *coverageServiceClient) GetTreeNodes(ctx context.Context, in *GetTreeNodesRequest, opts ...grpc.CallOption) (*GetTreeNodesResponse, error) {
	out := new(GetTreeNodesResponse)
	err := c.cc.Invoke(ctx, "/api.v1.coverage.CoverageService/GetTreeNodes", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *coverageServiceClient) GetFileCoverage(ctx context.Context, in *GetFileCoverageRequest, opts ...grpc.CallOption) (*GetFileCoverageResponse, error) {
	out := new(GetFileCoverageResponse)
	err := c.cc.Invoke(ctx, "/api.v1.coverage.CoverageService/GetFileCoverage", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// UnimplementedCoverageServiceServer 必须被嵌入以保证向前兼容性
type UnimplementedCoverageServiceServer struct{}

func (UnimplementedCoverageServiceServer) GetReportInfo(context.Context, *GetReportInfoRequest) (*GetReportInfoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetReportInfo not implemented")
}
func (UnimplementedCoverageServiceServer) GetTreeNodes(context.Context, *GetTreeNodesRequest) (*GetTreeNodesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetTreeNodes not implemented")
}
func (UnimplementedCoverageServiceServer) GetFileCoverage(context.Context, *GetFileCoverageRequest) (*GetFileCoverageResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetFileCoverage not implemented")
}
func (UnimplementedCoverageServiceServer) mustEmbedUnimplementedCoverageServiceServer() {}

// CoverageServiceServer 是 CoverageService 的服务端接口
type CoverageServiceServer interface {
	GetReportInfo(context.Context, *GetReportInfoRequest) (*GetReportInfoResponse, error)
	GetTreeNodes(context.Context, *GetTreeNodesRequest) (*GetTreeNodesResponse, error)
	GetFileCoverage(context.Context, *GetFileCoverageRequest) (*GetFileCoverageResponse, error)
	mustEmbedUnimplementedCoverageServiceServer()
}

// RegisterCoverageServiceServer 注册服务
func RegisterCoverageServiceServer(s grpc.ServiceRegistrar, srv CoverageServiceServer) {
	s.RegisterService(&CoverageService_ServiceDesc, srv)
}

// CoverageService_ServiceDesc 是 CoverageService 的服务描述
var CoverageService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "api.v1.coverage.CoverageService",
	HandlerType: (*CoverageServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetReportInfo",
			Handler:    _CoverageService_GetReportInfo_Handler,
		},
		{
			MethodName: "GetTreeNodes",
			Handler:    _CoverageService_GetTreeNodes_Handler,
		},
		{
			MethodName: "GetFileCoverage",
			Handler:    _CoverageService_GetFileCoverage_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "api/v1/coverage/coverage.proto",
}

func _CoverageService_GetReportInfo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetReportInfoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CoverageServiceServer).GetReportInfo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.v1.coverage.CoverageService/GetReportInfo",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CoverageServiceServer).GetReportInfo(ctx, req.(*GetReportInfoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CoverageService_GetTreeNodes_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetTreeNodesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CoverageServiceServer).GetTreeNodes(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.v1.coverage.CoverageService/GetTreeNodes",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CoverageServiceServer).GetTreeNodes(ctx, req.(*GetTreeNodesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CoverageService_GetFileCoverage_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetFileCoverageRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CoverageServiceServer).GetFileCoverage(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.v1.coverage.CoverageService/GetFileCoverage",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CoverageServiceServer).GetFileCoverage(ctx, req.(*GetFileCoverageRequest))
	}
	return interceptor(ctx, in, info, handler)
}
