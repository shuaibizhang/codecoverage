package main

import (
	"context"
	"log"
	"os"

	cp "github.com/shuaibizhang/codecoverage/internal/code_provider"
	"github.com/shuaibizhang/codecoverage/internal/config"
	"github.com/shuaibizhang/codecoverage/internal/oss"
	"github.com/shuaibizhang/codecoverage/internal/reports/manager"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/storage"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/storage/datasource"
	"github.com/shuaibizhang/codecoverage/internal/server"
	"github.com/shuaibizhang/codecoverage/internal/server/controller"
	"github.com/shuaibizhang/codecoverage/internal/service"
	utservice "github.com/shuaibizhang/codecoverage/internal/unittest/service"
	utstore "github.com/shuaibizhang/codecoverage/internal/unittest/store"
	"github.com/shuaibizhang/codecoverage/store"
	"github.com/shuaibizhang/codecoverage/store/db"
)

func main() {
	// 0. 初始化配置
	if err := config.Init(""); err != nil {
		log.Printf("Warning: failed to init config, using default values: %v", err)
	}
	cfg := config.GetConfig()

	// 1. 初始化存储
	// 1.1 初始化文件存储
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

	storageStore := storage.NewStorage(metaFile, coverFile, lock)
	mgr := manager.NewReportManager(storageStore)

	// 1.2 初始化数据库存储
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

	// 1.3 初始化 OSS 客户端
	ossCli, err := oss.NewMinioOSS(cfg.OssConfig)
	if err != nil {
		log.Fatalf("failed to init oss client: %v", err)
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
	covSvc := service.NewCoverageService(mgr, codeProv)
	unittestSvc := utservice.NewUnitTestService(unittestStore, ossCli, mgr, cfg.OssConfig.BucketName)

	covCtrl := controller.NewCoverageController(covSvc)
	utCtrl := controller.NewUnitTestController(unittestSvc)
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
