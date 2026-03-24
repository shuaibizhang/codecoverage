package server

import (
	"context"
	"log"
	"net"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/shuaibizhang/codecoverage/idl/cover-server/coverage"
	"github.com/shuaibizhang/codecoverage/idl/cover-server/register"
	"github.com/shuaibizhang/codecoverage/internal/server/controller"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"
)

type Server struct {
	grpcAddr string
	httpAddr string
	ctrl     *controller.Controller
}

func NewServer(grpcAddr, httpAddr string, ctrl *controller.Controller) *Server {
	return &Server{
		grpcAddr: grpcAddr,
		httpAddr: httpAddr,
		ctrl:     ctrl,
	}
}

func (s *Server) Run() error {
	// 1. 启动 gRPC 服务器
	lis, err := net.Listen("tcp", s.grpcAddr)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(100*1024*1024), // 允许接收 100MB 数据
		grpc.MaxSendMsgSize(100*1024*1024), // 允许发送 100MB 数据
	)
	coverage.RegisterCoverageServiceServer(grpcServer, s.ctrl)
	register.RegisterRegisterServiceServer(grpcServer, s.ctrl)
	reflection.Register(grpcServer)

	go func() {
		log.Printf("serving gRPC on %s", s.grpcAddr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()

	// 2. 启动 gRPC-Gateway
	gwmux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
		MarshalOptions: protojson.MarshalOptions{
			UseProtoNames:   true,
			EmitUnpopulated: true,
		},
		UnmarshalOptions: protojson.UnmarshalOptions{
			DiscardUnknown: true,
		},
	}))
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(100 * 1024 * 1024)), // 设置为 100MB
	}
	err = coverage.RegisterCoverageServiceHandlerFromEndpoint(context.Background(), gwmux, s.grpcAddr, opts)
	if err != nil {
		return err
	}
	err = register.RegisterRegisterServiceHandlerFromEndpoint(context.Background(), gwmux, s.grpcAddr, opts)
	if err != nil {
		return err
	}

	// 允许跨域
	handler := cors(gwmux)

	log.Printf("serving HTTP on %s", s.httpAddr)
	return http.ListenAndServe(s.httpAddr, handler)
}

func cors(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Incoming request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		if r.Method == "OPTIONS" {
			return
		}
		h.ServeHTTP(w, r)
	})
}
