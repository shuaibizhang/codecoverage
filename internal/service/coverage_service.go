package service

import (
	"context"
	"fmt"
	"log"

	"github.com/shuaibizhang/codecoverage/api/v1/coverage"
	cp "github.com/shuaibizhang/codecoverage/internal/code_provider"
	"github.com/shuaibizhang/codecoverage/internal/reports/manager"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/storage/partitionkey"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/tree"
	"github.com/shuaibizhang/codecoverage/internal/unittest/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CoverageService interface {
	GetReportInfo(ctx context.Context, req *coverage.GetReportInfoRequest) (*coverage.GetReportInfoResponse, error)
	GetTreeNodes(ctx context.Context, req *coverage.GetTreeNodesRequest) (*coverage.GetTreeNodesResponse, error)
	GetFileCoverage(ctx context.Context, req *coverage.GetFileCoverageRequest) (*coverage.GetFileCoverageResponse, error)
	GetMetadataList(ctx context.Context, req *coverage.GetMetadataListRequest) (*coverage.GetMetadataListResponse, error)
}

type coverageService struct {
	mgr          manager.ReportManager
	codeProvider cp.CodeProvider
	utStore      store.UnitTestStore
}

func NewCoverageService(mgr manager.ReportManager, codeProvider cp.CodeProvider, utStore store.UnitTestStore) CoverageService {
	return &coverageService{
		mgr:          mgr,
		codeProvider: codeProvider,
		utStore:      utStore,
	}
}

func (s *coverageService) GetReportInfo(ctx context.Context, req *coverage.GetReportInfoRequest) (*coverage.GetReportInfoResponse, error) {
	log.Printf("GetReportInfo: type=%s, module=%s, branch=%s, commit=%s", req.Type, req.Module, req.Branch, req.Commit)

	var pk partitionkey.PartitionKey
	if req.Type == "unittest" {
		task, err := s.utStore.Query(ctx, req.Module, req.Branch, req.Commit)
		if err != nil {
			log.Printf("GetReportInfo: query db failed: %v", err)
			return nil, status.Errorf(codes.NotFound, "report not found in database: %v", err)
		}
		if task.ReportPartitionKey == "" {
			log.Printf("GetReportInfo: task found but ReportPartitionKey is empty")
			return nil, status.Errorf(codes.NotFound, "report partition key is empty in database")
		}

		pk = partitionkey.NewReportKey("", "", "", "")
		if err := pk.Unmarshal(task.ReportPartitionKey); err != nil {
			log.Printf("GetReportInfo: unmarshal pk failed: %v, data=%s", err, task.ReportPartitionKey)
			return nil, status.Errorf(codes.Internal, "invalid report_id in database: %v", err)
		}
	} else {
		pk = partitionkey.NewReportKey(partitionkey.TestType(req.Type), req.Module, req.Branch, req.Commit)
	}

	pkStr, err := pk.Marshal()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal report_id: %v", err)
	}

	log.Printf("GetReportInfo: opening report with PK path: %s", pk.RealPathPrefix())
	rep, err := s.mgr.Open(ctx, pk)
	if err != nil {
		log.Printf("GetReportInfo: open report failed: %v, path=%s", err, pk.RealPathPrefix())
		return nil, status.Errorf(codes.NotFound, "report not found: %v", err)
	}
	defer rep.Close(ctx)

	meta := rep.GetMeta()

	return &coverage.GetReportInfoResponse{
		ReportId: pkStr,
		Meta: &coverage.MetaInfo{
			Module:     meta.Module,
			Branch:     meta.Branch,
			Commit:     meta.Commit,
			TotalFiles: uint32(meta.TotalFiles),
			LastUpdate: meta.LastUpdate,
		},
	}, nil
}

func (s *coverageService) GetTreeNodes(ctx context.Context, req *coverage.GetTreeNodesRequest) (*coverage.GetTreeNodesResponse, error) {
	pk := partitionkey.NewReportKey("", "", "", "")
	if err := pk.Unmarshal(req.ReportId); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid report_id")
	}

	rep, err := s.mgr.Open(ctx, pk)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "report not found")
	}
	defer rep.Close(ctx)

	node := rep.FindNode(req.Path)
	if node == nil {
		return nil, status.Errorf(codes.NotFound, "path not found")
	}

	dir, ok := node.(*tree.DirNode)
	var nodes []*coverage.TreeNode
	if ok {
		for child := range dir.Children() {
			nodeType := int32(0) // Dir
			if !child.IsDir() {
				nodeType = 1 // File
			}
			stat := child.GetStat()
			nodes = append(nodes, &coverage.TreeNode{
				Name: child.Name(),
				Path: child.Path(),
				Type: nodeType,
				Stat: &coverage.TreeNodeStat{
					TotalLines:     stat.TotalLines,
					InstrLines:     stat.InstrLines,
					CoverLines:     stat.CoverLines,
					Coverage:       stat.Coverage,
					AddLines:       stat.AddLines,
					DeleteLines:    stat.DeleteLines,
					IncrInstrLines: stat.IncrInstrLines,
					IncrCoverLines: stat.IncrCoverLines,
					IncrCoverage:   stat.IncrCoverage,
				},
			})
		}
	} else {
		stat := node.GetStat()
		nodes = append(nodes, &coverage.TreeNode{
			Name: node.Name(),
			Path: node.Path(),
			Type: 1,
			Stat: &coverage.TreeNodeStat{
				TotalLines:     stat.TotalLines,
				InstrLines:     stat.InstrLines,
				CoverLines:     stat.CoverLines,
				Coverage:       stat.Coverage,
				AddLines:       stat.AddLines,
				DeleteLines:    stat.DeleteLines,
				IncrInstrLines: stat.IncrInstrLines,
				IncrCoverLines: stat.IncrCoverLines,
				IncrCoverage:   stat.IncrCoverage,
			},
		})
	}

	return &coverage.GetTreeNodesResponse{Nodes: nodes}, nil
}

func (s *coverageService) GetFileCoverage(ctx context.Context, req *coverage.GetFileCoverageRequest) (*coverage.GetFileCoverageResponse, error) {
	log.Printf("GetFileCoverage: report_id=%s, path=%s", req.ReportId, req.Path)
	pk := partitionkey.NewReportKey("", "", "", "")
	if err := pk.Unmarshal(req.ReportId); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid report_id")
	}

	rep, err := s.mgr.Open(ctx, pk)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "report not found")
	}
	defer rep.Close(ctx)

	meta := rep.GetMeta()

	lines, err := rep.GetFileCoverLines(req.Path)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get lines: %v", err)
	}

	content, err := s.codeProvider.GetFileContent(ctx, meta.Module, meta.Commit, req.Path)
	if err != nil {
		log.Printf("Error getting source code: %v (module=%s, commit=%s, path=%s)", err, meta.Module, meta.Commit, req.Path)
		content = fmt.Sprintf("Error reading source: %v", err)
	}

	return &coverage.GetFileCoverageResponse{
		Path:    req.Path,
		Lines:   lines,
		Content: content,
	}, nil
}

func (s *coverageService) GetMetadataList(ctx context.Context, req *coverage.GetMetadataListRequest) (*coverage.GetMetadataListResponse, error) {
	log.Printf("GetMetadataList called with type: %s", req.Type)
	// 目前仅支持 unittest 类型
	if req.Type != "unittest" && req.Type != "" {
		log.Printf("GetMetadataList type not supported: %s", req.Type)
		return &coverage.GetMetadataListResponse{}, nil
	}

	modules, branches, commits, err := s.utStore.GetMetadataList(ctx)
	if err != nil {
		log.Printf("GetMetadataList failed: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to get metadata list: %v", err)
	}

	log.Printf("GetMetadataList success: modules=%v, branches=%v, commits=%v", modules, branches, commits)
	return &coverage.GetMetadataListResponse{
		Modules:  modules,
		Branches: branches,
		Commits:  commits,
	}, nil
}
