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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CoverageService struct {
	coverage.UnimplementedCoverageServiceServer
	mgr          manager.ReportManager
	codeProvider cp.CodeProvider
}

func NewCoverageService(mgr manager.ReportManager, codeProvider cp.CodeProvider) *CoverageService {
	return &CoverageService{
		mgr:          mgr,
		codeProvider: codeProvider,
	}
}

func (s *CoverageService) GetReportInfo(ctx context.Context, req *coverage.GetReportInfoRequest) (*coverage.GetReportInfoResponse, error) {
	pk := partitionkey.NewReportKey(partitionkey.TestType(req.Type), req.Module, req.Branch, req.Commit)
	rep, err := s.mgr.Open(ctx, pk)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "report not found: %v", err)
	}

	pkStr, err := pk.Marshal()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal report_id: %v", err)
	}
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

func (s *CoverageService) GetTreeNodes(ctx context.Context, req *coverage.GetTreeNodesRequest) (*coverage.GetTreeNodesResponse, error) {
	pk := partitionkey.NewReportKey("", "", "", "")
	if err := pk.Unmarshal(req.ReportId); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid report_id")
	}

	rep, err := s.mgr.Open(ctx, pk)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "report not found")
	}

	// 我们需要知道哪些是目录，哪些是文件
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
		// 如果是文件，返回自身
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

func (s *CoverageService) GetFileCoverage(ctx context.Context, req *coverage.GetFileCoverageRequest) (*coverage.GetFileCoverageResponse, error) {
	log.Printf("GetFileCoverage: report_id=%s, path=%s", req.ReportId, req.Path)
	pk := partitionkey.NewReportKey("", "", "", "")
	if err := pk.Unmarshal(req.ReportId); err != nil {
		log.Printf("GetFileCoverage: invalid report_id: %v", err)
		return nil, status.Errorf(codes.InvalidArgument, "invalid report_id")
	}

	rep, err := s.mgr.Open(ctx, pk)
	if err != nil {
		log.Printf("GetFileCoverage: report not found: %v", err)
		return nil, status.Errorf(codes.NotFound, "report not found")
	}

	meta := rep.GetMeta()

	lines, err := rep.GetFileCoverLines(req.Path)
	if err != nil {
		log.Printf("GetFileCoverage: failed to get lines for path %s: %v", req.Path, err)
		return nil, status.Errorf(codes.Internal, "failed to get lines: %v", err)
	}

	// 读取源码
	content, err := s.codeProvider.GetFileContent(ctx, meta.Module, meta.Commit, req.Path)
	if err != nil {
		log.Printf("GetFileCoverage: codeProvider failed for path %s: %v", req.Path, err)
		// 源码读取失败不应导致接口报错，只是不返回 content
		content = fmt.Sprintf("Error reading source: %v", err)
	}

	return &coverage.GetFileCoverageResponse{
		Path:    req.Path,
		Lines:   lines,
		Content: content,
	}, nil
}
