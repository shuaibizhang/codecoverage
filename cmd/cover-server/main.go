package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	cp "github.com/shuaibizhang/codecoverage/internal/code_provider"
	"github.com/shuaibizhang/codecoverage/internal/config"
	diffProvider "github.com/shuaibizhang/codecoverage/internal/diff/provider"
	diffService "github.com/shuaibizhang/codecoverage/internal/diff/service"
	diffStore "github.com/shuaibizhang/codecoverage/internal/diff/store"
	githubCli "github.com/shuaibizhang/codecoverage/internal/github"
	"github.com/shuaibizhang/codecoverage/internal/oss"
	"github.com/shuaibizhang/codecoverage/internal/reports/manager"
	"github.com/shuaibizhang/codecoverage/internal/reports/report"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/storage"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/storage/datasource"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/storage/partitionkey"
	"github.com/shuaibizhang/codecoverage/internal/server"
	"github.com/shuaibizhang/codecoverage/internal/server/controller"
	"github.com/shuaibizhang/codecoverage/internal/service"
	snservice "github.com/shuaibizhang/codecoverage/internal/snapshot/service"
	snstore "github.com/shuaibizhang/codecoverage/internal/snapshot/store"
	stservice "github.com/shuaibizhang/codecoverage/internal/systest/service"
	ststore "github.com/shuaibizhang/codecoverage/internal/systest/store"
	utservice "github.com/shuaibizhang/codecoverage/internal/unittest/service"
	utstore "github.com/shuaibizhang/codecoverage/internal/unittest/store"
	"github.com/shuaibizhang/codecoverage/logger"
	"github.com/shuaibizhang/codecoverage/store"
	"github.com/shuaibizhang/codecoverage/store/db"
)

var confPath = flag.String("conf", "conf/server-dev.toml", "")

func main() {
	// 初始化全局 Logger (基于环境变量 LOG_OUTPUT 和 LOG_FILE_PATH)
	logger.SetDefault(logger.NewZapLogger(logger.NewProductionConfig()))
	// 重定向标准库 log 的输出到我们的 logger
	logger.RedirectStdLog()

	log.Printf("--------------------------------------------------")
	log.Printf("Cover Server is starting...")
	log.Printf("PID: %d", os.Getpid())
	log.Printf("--------------------------------------------------")
	flag.Parse()
	if confPath == nil || *confPath == "" {
		panic("conf path err")
	}
	ctx := context.Background()
	// 0. 初始化配置
	if err := config.Init(ctx, *confPath); err != nil {
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
	if err := ossCli.MakeBucket(ctx, cfg.OssConfig.BucketName); err != nil {
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
	var systestStore ststore.SystestStore
	var snapshotStore snstore.SnapshotStore
	var snapshotSvc snservice.SnapshotService
	if cfg.DbConfig.Host != "" {
		database, err := db.Open(&cfg.DbConfig)
		if err != nil {
			log.Printf("Warning: failed to open database: %v", err)
		} else {
			dbStore = store.NewStore(database)
			unittestStore = utstore.NewUnitTestStore(dbStore)
			systestStore = ststore.NewSystestStore(dbStore)
			snapshotStore = snstore.NewSnapshotStore(dbStore)
			snapshotSvc = snservice.NewSnapshotService(snapshotStore, mgr)
			log.Printf("Database initialized successfully")
		}
	}

	// 1.4 初始化 GitHub 客户端 (如果配置了 token)
	var ghCli githubCli.Client
	if cfg.GithubConfig.Token != "" {
		ghCli = githubCli.NewClient(cfg.GithubConfig.Token)
		log.Printf("GitHub client initialized successfully")
	}

	// 1.5 初始化 CodeProvider
	var codeProv cp.CodeProvider
	if ghCli != nil {
		log.Printf("Using GitHubCodeProvider with owner: %s", cfg.GithubConfig.Owner)
		codeProv = cp.NewGithubCodeProvider(ghCli, cfg.GithubConfig.Owner)
	} else {
		log.Printf("Using LocalCodeProvider")
		codeProv = cp.NewLocalCodeProvider("")
	}

	// 1.6 初始化 DiffService
	var diffSvc diffService.Service
	if dbStore != nil {
		dStore := diffStore.NewDiffStore(dbStore)
		ossProv := diffProvider.NewOSSDiffProvider(ossCli, dStore, cfg.OssConfig.BucketName)

		var baseProv diffProvider.DiffProvider
		if ghCli != nil {
			log.Printf("Using GithubDiffProvider with owner: %s", cfg.GithubConfig.Owner)
			baseProv = diffProvider.NewGithubDiffProvider(ghCli, cfg.GithubConfig.Owner, ossCli, dStore, cfg.OssConfig.BucketName)
		} else {
			log.Printf("Using LocalDiffProvider as GitHub token is missing")
			baseProv = diffProvider.NewLocalDiffProvider("")
		}

		decoratorProv := diffProvider.NewCacheDiffProvider(ossProv, baseProv)
		diffSvc = diffService.NewDiffService(decoratorProv)
		log.Printf("DiffService initialized successfully")
	}

	// 1.6 初始化 Service
	covSvc := service.NewCoverageService(mgr, codeProv, diffSvc, unittestStore, systestStore, snapshotSvc)
	var unittestSvc utservice.UnitTestService
	if unittestStore != nil {
		unittestSvc = utservice.NewUnitTestService(unittestStore, ossCli, mgr, cfg.OssConfig.BucketName)
	}
	var systestSvc stservice.SystestService
	if systestStore != nil {
		systestSvc = stservice.NewSystestService(systestStore, ossCli, mgr, diffSvc, cfg.OssConfig.BucketName)
	}
	regSvc := service.NewRegisterService()

	covCtrl := controller.NewCoverageController(covSvc)
	var utCtrl *controller.UnitTestController
	if unittestSvc != nil {
		utCtrl = controller.NewUnitTestController(unittestSvc)
	}
	var stCtrl *controller.SystestController
	if systestSvc != nil {
		stCtrl = controller.NewSystestController(systestSvc)
	}
	regCtrl := controller.NewRegisterController(regSvc)
	ctrl := controller.NewController(covCtrl, utCtrl, stCtrl, regCtrl)

	// 2. 启动服务器
	srv := server.NewServer(cfg.ServerConfig.GrpcAddr, cfg.ServerConfig.HttpAddr, ctrl)
	defer logger.Default().Sync()
	if err := srv.Run(); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

type simpleLock struct{}

func (l *simpleLock) Lock(ctx context.Context) error    { return nil }
func (l *simpleLock) Unlock(ctx context.Context) error  { return nil }
func (l *simpleLock) CanWrite(ctx context.Context) bool { return true }
