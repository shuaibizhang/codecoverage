package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/shuaibizhang/codecoverage/api/v1/coverage"
	cp "github.com/shuaibizhang/codecoverage/internal/code_provider"
	"github.com/shuaibizhang/codecoverage/internal/config"
	"github.com/shuaibizhang/codecoverage/internal/reports/manager"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/storage"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/storage/datasource"
	svc "github.com/shuaibizhang/codecoverage/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// 0. 初始化配置
	if err := config.Init(""); err != nil {
		log.Printf("Warning: failed to init config, using default values: %v", err)
	}
	cfg := config.GetConfig()

	// 1. 初始化存储
	// 为了演示，我们使用本地文件作为数据源
	os.MkdirAll("coverage/reports", 0755)

	// 如果文件不存在则创建
	metaPath := "coverage/reports/meta.cno"
	coverPath := "coverage/reports/cover.cda"

	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		f, _ := os.Create(metaPath)
		f.Close()
	}
	if _, err := os.Stat(coverPath); os.IsNotExist(err) {
		f, _ := os.Create(coverPath)
		f.Close()
	}

	metaFile, err := datasource.OpenFileDataSource(metaPath)
	if err != nil {
		log.Fatalf("failed to open meta datasource: %v", err)
	}
	coverFile, err := datasource.OpenFileDataSource(coverPath)
	if err != nil {
		log.Fatalf("failed to open cover datasource: %v", err)
	}

	// 简单的内存锁
	lock := &simpleLock{}

	store := storage.NewStorage(metaFile, coverFile, lock)
	mgr := manager.NewReportManager(store)

	// 初始化 CodeProvider
	var codeProv cp.CodeProvider
	if cfg.GithubConfig.Token != "" {
		log.Printf("Using GitHubCodeProvider with owner: %s", cfg.GithubConfig.Owner)
		codeProv = cp.NewGithubCodeProvider(cfg.GithubConfig.Token, cfg.GithubConfig.Owner)
	} else {
		log.Printf("Using LocalCodeProvider")
		codeProv = cp.NewLocalCodeProvider("")
	}

	serviceImpl := svc.NewCoverageService(mgr, codeProv)

	// 2. 启动 gRPC 服务器
	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	coverage.RegisterCoverageServiceServer(s, serviceImpl)
	reflection.Register(s)

	go func() {
		log.Printf("serving gRPC on 0.0.0.0:9090")
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()

	// 3. 启动 HTTP API (手动实现以替代 gRPC-Gateway)
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/coverage/report", func(w http.ResponseWriter, r *http.Request) {
		req := &coverage.GetReportInfoRequest{
			Module: r.URL.Query().Get("module"),
			Branch: r.URL.Query().Get("branch"),
			Commit: r.URL.Query().Get("commit"),
			Type:   r.URL.Query().Get("type"),
		}
		resp, err := serviceImpl.GetReportInfo(r.Context(), req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/api/v1/coverage/tree", func(w http.ResponseWriter, r *http.Request) {
		req := &coverage.GetTreeNodesRequest{
			ReportId: r.URL.Query().Get("report_id"),
			Path:     r.URL.Query().Get("path"),
		}
		resp, err := serviceImpl.GetTreeNodes(r.Context(), req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/api/v1/coverage/file", func(w http.ResponseWriter, r *http.Request) {
		req := &coverage.GetFileCoverageRequest{
			ReportId: r.URL.Query().Get("report_id"),
			Path:     r.URL.Query().Get("path"),
		}
		resp, err := serviceImpl.GetFileCoverage(r.Context(), req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	// 允许跨域
	handler := cors(mux)

	log.Printf("serving HTTP on 0.0.0.0:8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatalf("failed to serve HTTP: %v", err)
	}
}

type simpleLock struct{}

func (l *simpleLock) Lock(ctx context.Context) error    { return nil }
func (l *simpleLock) Unlock(ctx context.Context) error  { return nil }
func (l *simpleLock) CanWrite(ctx context.Context) bool { return true }

func cors(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		if r.Method == "OPTIONS" {
			return
		}
		h.ServeHTTP(w, r)
	})
}
