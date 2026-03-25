package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/shuaibizhang/codecoverage/idl/cover-server/coverage"
	"github.com/shuaibizhang/codecoverage/internal/reports/manager"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/storage/partitionkey"
	"github.com/shuaibizhang/codecoverage/internal/snapshot/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SnapshotService interface {
	CreateSnapshot(ctx context.Context, req *coverage.CreateSnapshotRequest) (*coverage.CreateSnapshotResponse, error)
	GetMetadataList(ctx context.Context, module, branch string) ([]string, []string, []string, error)
	QueryLatestByCommit(ctx context.Context, module, branch, commit string) (*store.SnapshotInfo, error)
}

type snapshotService struct {
	snStore store.SnapshotStore
	mgr     manager.ReportManager
}

func NewSnapshotService(snStore store.SnapshotStore, mgr manager.ReportManager) SnapshotService {
	return &snapshotService{
		snStore: snStore,
		mgr:     mgr,
	}
}

func (s *snapshotService) CreateSnapshot(ctx context.Context, req *coverage.CreateSnapshotRequest) (*coverage.CreateSnapshotResponse, error) {
	log.Printf("CreateSnapshot: report_id=%s", req.ReportId)
	// 1. 解析 report_id
	pk, err := partitionkey.UnmarshalPartitionKey(req.ReportId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid report_id: %v", err)
	}

	// 2. 打开原报告
	srcRep, err := s.mgr.Open(ctx, pk)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "source report not found: %v", err)
	}
	defer srcRep.Close(ctx)

	// 3. 创建快照报告
	meta := srcRep.GetMeta()
	// 确保 meta 中的关键信息不为空，如果为空则从原 pk 中获取
	if meta.Module == "" {
		meta.Module = pk.GetModule()
	}
	if meta.Branch == "" {
		meta.Branch = pk.GetBranch()
	}
	if meta.Commit == "" {
		meta.Commit = pk.GetCommit()
	}

	// 校验元数据
	if meta.Module == "" || meta.Branch == "" || meta.Commit == "" {
		return nil, status.Errorf(codes.InvalidArgument, "invalid report metadata: module, branch and commit are required (module=%s, branch=%s, commit=%s)", meta.Module, meta.Branch, meta.Commit)
	}

	// 设置新的元数据
	meta.LastUpdate = time.Now().Format("2006-01-02 15:04:05")

	snapshotPk := partitionkey.NewSnapshotReportKey(meta.Module, meta.Branch, meta.Commit)
	snapshotPkStr, err := snapshotPk.Marshal()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal snapshot report_id: %v", err)
	}

	destRep, err := s.mgr.CreateReport(ctx, meta, snapshotPk)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create snapshot report: %v", err)
	}
	defer destRep.Close(ctx)

	// 4. 合并数据（同源合并）
	if err := s.mgr.MergeSameCommitReport(ctx, destRep, srcRep); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to merge data to snapshot: %v", err)
	}

	// 5. 持久化快照
	if err := destRep.Flush(ctx); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to flush snapshot: %v", err)
	}

	// 6. 保存快照信息到数据库
	snapshotID := fmt.Sprintf("sn_%d", time.Now().UnixNano())
	info := &store.SnapshotInfo{
		Module:             meta.Module,
		Branch:             meta.Branch,
		Commit:             meta.Commit,
		BaseCommit:         meta.BaseCommit,
		SnapshotID:         snapshotID,
		ReportPartitionKey: snapshotPkStr,
	}
	if err := s.snStore.Save(ctx, info); err != nil {
		log.Printf("CreateSnapshot: failed to save snapshot info to db: %v", err)
		// 数据库保存失败，但不影响 OSS 已经写入，仅记录日志
	}

	log.Printf("CreateSnapshot success: snapshot_report_id=%s, snapshot_id=%s", snapshotPkStr, snapshotID)
	return &coverage.CreateSnapshotResponse{
		SnapshotReportId: snapshotPkStr,
		Success:          true,
		Message:          "Snapshot created successfully",
	}, nil
}

func (s *snapshotService) GetMetadataList(ctx context.Context, module, branch string) ([]string, []string, []string, error) {
	return s.snStore.GetMetadataList(ctx, module, branch)
}

func (s *snapshotService) QueryLatestByCommit(ctx context.Context, module, branch, commit string) (*store.SnapshotInfo, error) {
	return s.snStore.QueryLatestByCommit(ctx, module, branch, commit)
}
