package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	cp "github.com/shuaibizhang/codecoverage/internal/code_provider"
	"github.com/shuaibizhang/codecoverage/internal/config"
	"github.com/shuaibizhang/codecoverage/internal/oss"
	"github.com/shuaibizhang/codecoverage/internal/reports/manager"
	"github.com/shuaibizhang/codecoverage/internal/reports/report"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/storage"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/storage/datasource"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/storage/partitionkey"
	"github.com/shuaibizhang/codecoverage/internal/server"
	"github.com/shuaibizhang/codecoverage/internal/server/controller"
	"github.com/shuaibizhang/codecoverage/internal/service"
	utservice "github.com/shuaibizhang/codecoverage/internal/unittest/service"
	utstore "github.com/shuaibizhang/codecoverage/internal/unittest/store"
	"github.com/shuaibizhang/codecoverage/store"
	"github.com/shuaibizhang/codecoverage/store/db"
)

var confPath = flag.String("conf", "conf/dev.toml", "")

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("--------------------------------------------------")
	log.Printf("Cover Server is starting...")
	log.Printf("PID: %d", os.Getpid())
	log.Printf("--------------------------------------------------")
	flag.Parse()
	if confPath == nil || *confPath == "" {
		panic("conf path err")
	}

	// 0. 初始化配置
	if err := config.Init(*confPath); err != nil {
		log.Printf("Warning: failed to init config, using default values: %v", err)
	}
	cfg := config.GetConfig()

	// 1. 初始化存储
	// 1.1 初始化 OSS 客户端
	ossCli, err := oss.NewMinioOSS(cfg.OssConfig)
	if err != nil {
		log.Fatalf("failed to init oss client: %v", err)
	}

	// 1.1.1 确保存储桶存在
	if err := ossCli.MakeBucket(context.Background(), cfg.OssConfig.BucketName); err != nil {
		log.Printf("Warning: failed to make bucket %s: %v", cfg.OssConfig.BucketName, err)
	}

	// 简单的内存锁
	lock := &simpleLock{}

	// 1.2 初始化 ReportManager (使用动态存储工厂)
	mgr := manager.NewReportManager(func(ctx context.Context, pk partitionkey.PartitionKey) (report.Storage, error) {
		prefix := pk.RealPathPrefix()
		metaPath := prefix + ".cno"
		coverPath := prefix + ".cda"

		metaFile, err := datasource.NewOSSDataSource(ctx, ossCli, cfg.OssConfig.BucketName, metaPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open meta datasource from oss: %w", err)
		}
		coverFile, err := datasource.NewOSSDataSource(ctx, ossCli, cfg.OssConfig.BucketName, coverPath)
		if err != nil {
			metaFile.Close()
			return nil, fmt.Errorf("failed to open cover datasource from oss: %w", err)
		}

		return storage.NewStorage(metaFile, coverFile, lock), nil
	})

	// 1.3 初始化数据库存储
	var dbStore *store.Store
	var unittestStore utstore.UnitTestStore
	if cfg.DbConfig.Host != "" {
		database, err := db.Open(&cfg.DbConfig)
		if err != nil {
			log.Printf("Warning: failed to open database: %v", err)
		} else {
			dbStore = store.NewStore(database)
			unittestStore = utstore.NewUnitTestStore(dbStore)
			log.Printf("Database initialized successfully")
		}
	}

	// 1.4 初始化 CodeProvider
	var codeProv cp.CodeProvider
	if cfg.GithubConfig.Token != "" {
		log.Printf("Using GitHubCodeProvider with owner: %s", cfg.GithubConfig.Owner)
		codeProv = cp.NewGithubCodeProvider(cfg.GithubConfig.Token, cfg.GithubConfig.Owner)
	} else {
		log.Printf("Using LocalCodeProvider")
		codeProv = cp.NewLocalCodeProvider("")
	}

	// 1.5 初始化 Service
	covSvc := service.NewCoverageService(mgr, codeProv, unittestStore)
	var unittestSvc utservice.UnitTestService
	if unittestStore != nil {
		unittestSvc = utservice.NewUnitTestService(unittestStore, ossCli, mgr, cfg.OssConfig.BucketName)
	}

	covCtrl := controller.NewCoverageController(covSvc)
	var utCtrl *controller.UnitTestController
	if unittestSvc != nil {
		utCtrl = controller.NewUnitTestController(unittestSvc)
	}
	ctrl := controller.NewController(covCtrl, utCtrl)

	// 2. 启动服务器
	srv := server.NewServer(":9090", ":8080", ctrl)
	if err := srv.Run(); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

type simpleLock struct{}

func (l *simpleLock) Lock(ctx context.Context) error    { return nil }
func (l *simpleLock) Unlock(ctx context.Context) error  { return nil }
func (l *simpleLock) CanWrite(ctx context.Context) bool { return true }
