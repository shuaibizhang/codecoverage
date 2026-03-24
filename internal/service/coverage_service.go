package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/shuaibizhang/codecoverage/idl/cover-server/coverage"
	cp "github.com/shuaibizhang/codecoverage/internal/code_provider"
	diffService "github.com/shuaibizhang/codecoverage/internal/diff/service"
	"github.com/shuaibizhang/codecoverage/internal/reports/manager"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/storage/partitionkey"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/tree"
	ststore "github.com/shuaibizhang/codecoverage/internal/systest/store"
	utstore "github.com/shuaibizhang/codecoverage/internal/unittest/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CoverageService interface {
	GetReportInfo(ctx context.Context, req *coverage.GetReportInfoRequest) (*coverage.GetReportInfoResponse, error)
	GetTreeNodes(ctx context.Context, req *coverage.GetTreeNodesRequest) (*coverage.GetTreeNodesResponse, error)
	GetFileCoverage(ctx context.Context, req *coverage.GetFileCoverageRequest) (*coverage.GetFileCoverageResponse, error)
	GetMetadataList(ctx context.Context, req *coverage.GetMetadataListRequest) (*coverage.GetMetadataListResponse, error)
	MergeReports(ctx context.Context, req *coverage.MergeReportsRequest) (*coverage.MergeReportsResponse, error)
	GetRootCoverage(ctx context.Context, req *coverage.GetRootCoverageRequest) (*coverage.GetRootCoverageResponse, error)
	SearchNodes(ctx context.Context, req *coverage.SearchNodesRequest) (*coverage.SearchNodesResponse, error)
}

type coverageService struct {
	mgr          manager.ReportManager
	codeProvider cp.CodeProvider
	diffSvc      diffService.Service
	utStore      utstore.UnitTestStore
	stStore      ststore.SystestStore
}

func NewCoverageService(mgr manager.ReportManager, codeProvider cp.CodeProvider, diffSvc diffService.Service, utStore utstore.UnitTestStore, stStore ststore.SystestStore) CoverageService {
	return &coverageService{
		mgr:          mgr,
		codeProvider: codeProvider,
		diffSvc:      diffSvc,
		utStore:      utStore,
		stStore:      stStore,
	}
}

func (s *coverageService) GetReportInfo(ctx context.Context, req *coverage.GetReportInfoRequest) (*coverage.GetReportInfoResponse, error) {
	log.Printf("GetReportInfo: type=%s, module=%s, branch=%s, commit=%s", req.Type, req.Module, req.Branch, req.Commit)

	var pk partitionkey.PartitionKey
	if req.Type == "unittest" || req.Type == "unit_test" || req.Type == "unit" {
		if s.utStore == nil {
			return nil, status.Errorf(codes.FailedPrecondition, "unittest store not initialized")
		}
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
	} else if req.Type == "systest" || req.Type == "integrate" || req.Type == "integration" {
		if s.stStore == nil {
			return nil, status.Errorf(codes.FailedPrecondition, "systest store not initialized")
		}
		task, err := s.stStore.Query(ctx, req.Module, req.Branch, req.Commit)
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
	module := meta.Module
	if module == "" {
		module = pk.GetModule()
	}
	branch := meta.Branch
	if branch == "" {
		branch = pk.GetBranch()
	}
	commit := meta.Commit
	if commit == "" {
		commit = pk.GetCommit()
	}

	return &coverage.GetReportInfoResponse{
		ReportId: pkStr,
		Meta: &coverage.MetaInfo{
			Module:     module,
			Branch:     branch,
			Commit:     commit,
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
			stat := child.GetStat()
			// 如果查看增量，过滤掉没有增量数据的节点
			if req.IsIncrement && !stat.HasIncrement {
				continue
			}

			nodeType := int32(0) // Dir
			if !child.IsDir() {
				nodeType = 1 // File
			}
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
					HasIncrement:   stat.HasIncrement,
				},
			})
		}
	} else {
		stat := node.GetStat()
		if !req.IsIncrement || stat.HasIncrement {
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
					HasIncrement:   stat.HasIncrement,
				},
			})
		}
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
	module := meta.Module
	if module == "" {
		module = pk.GetModule()
	}
	commit := meta.Commit
	if commit == "" {
		commit = pk.GetCommit()
	}

	lines, err := rep.GetFileCoverLines(req.Path)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get lines: %v", err)
	}

	content, err := s.codeProvider.GetFileContent(ctx, module, commit, req.Path)
	if err != nil {
		log.Printf("Error getting source code: %v (module=%s, commit=%s, path=%s)", err, module, commit, req.Path)
		content = fmt.Sprintf("Error reading source: %v (owner=%s, repo=%s, ref=%s, status=%s)", err, "", "", "", "")
	}

	return &coverage.GetFileCoverageResponse{
		Path:    req.Path,
		Lines:   lines,
		Content: content,
	}, nil
}

func (s *coverageService) MergeReports(ctx context.Context, req *coverage.MergeReportsRequest) (*coverage.MergeReportsResponse, error) {
	if req.BaseReport == nil {
		return nil, status.Errorf(codes.InvalidArgument, "base_report is required")
	}

	// 1. 解析基准报告
	basePk, err := s.resolveReportPk(ctx, req.BaseReport)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "failed to resolve base report: %v", err)
	}

	baseRep, err := s.mgr.OpenWrite(ctx, basePk)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to open base report: %v", err)
	}
	defer baseRep.Close(ctx)

	baseMeta := baseRep.GetMeta()
	baseModule := baseMeta.Module
	if baseModule == "" {
		baseModule = basePk.GetModule()
	}
	baseCommit := baseMeta.Commit
	if baseCommit == "" {
		baseCommit = basePk.GetCommit()
	}

	// 2. 依次合并其他报告
	for _, otherSource := range req.OtherReports {
		otherPk, err := s.resolveReportPk(ctx, otherSource)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "failed to resolve report: %v", err)
		}

		otherPkStr, _ := otherPk.Marshal()
		basePkStr, _ := basePk.Marshal()
		if otherPkStr == basePkStr {
			continue // 跳过自身
		}

		otherRep, err := s.mgr.Open(ctx, otherPk)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to open report: %v", err)
		}

		otherMeta := otherRep.GetMeta()
		otherModule := otherMeta.Module
		if otherModule == "" {
			otherModule = otherPk.GetModule()
		}
		otherCommit := otherMeta.Commit
		if otherCommit == "" {
			otherCommit = otherPk.GetCommit()
		}

		// 3. 校验模块是否相同
		if otherModule != baseModule {
			otherRep.Close(ctx)
			return nil, status.Errorf(codes.InvalidArgument, "cannot merge reports from different modules: base=%s, other=%s", baseModule, otherModule)
		}

		// 4. 判断是否为异源合并
		if otherCommit == baseCommit {
			// 同源合并
			if err := s.mgr.MergeSameCommitReport(ctx, baseRep, otherRep); err != nil {
				log.Printf("MergeSameCommitReport failed: %v", err)
			}
		} else {
			// 异源合并：需要获取 diff
			gitDiffMap, err := s.diffSvc.GetDiff(ctx, baseModule, "", otherCommit, baseCommit)
			if err != nil {
				log.Printf("GetDiff failed for heterogeneous merge: %v", err)
				// 兜底：如果获取不到 diff，尝试直接合并（可能会有偏移风险）
				if err := s.mgr.MergeSameCommitReport(ctx, baseRep, otherRep); err != nil {
					log.Printf("Fallback MergeSameCommitReport failed: %v", err)
				}
			} else {
				if err := s.mgr.MergeDiffCommitReport(ctx, baseRep, otherRep, gitDiffMap.DiffFileMap); err != nil {
					log.Printf("MergeDiffCommitReport failed: %v", err)
				}
			}
		}
		otherRep.Close(ctx)
	}

	// 4. 保存合并后的结果
	if err := baseRep.Flush(ctx); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to flush merged report: %v", err)
	}

	finalPkStr, _ := basePk.Marshal()
	return &coverage.MergeReportsResponse{
		MergedReportId: finalPkStr,
		Success:        true,
	}, nil
}

func (s *coverageService) resolveReportPk(ctx context.Context, source *coverage.MergeReportsRequest_ReportSource) (partitionkey.PartitionKey, error) {
	if source == nil {
		return nil, fmt.Errorf("source is nil")
	}

	log.Printf("resolveReportPk: source=%+v", source)

	switch src := source.Source.(type) {
	case *coverage.MergeReportsRequest_ReportSource_ReportId:
		log.Printf("resolveReportPk: using report_id=%s", src.ReportId)
		pk := partitionkey.NewReportKey("", "", "", "")
		if err := pk.Unmarshal(src.ReportId); err != nil {
			return nil, fmt.Errorf("invalid report_id: %v", err)
		}
		return pk, nil

	case *coverage.MergeReportsRequest_ReportSource_Selector:
		log.Printf("resolveReportPk: using selector=%+v", src.Selector)
		// 通过业务维度查询报告 ID
		req := &coverage.GetReportInfoRequest{
			Module: src.Selector.Module,
			Branch: src.Selector.Branch,
			Commit: src.Selector.Commit,
			Type:   src.Selector.Type,
		}
		resp, err := s.GetReportInfo(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("report not found by selector: %v", err)
		}
		pk := partitionkey.NewReportKey("", "", "", "")
		if err := pk.Unmarshal(resp.ReportId); err != nil {
			return nil, fmt.Errorf("invalid resolved report_id: %v", err)
		}
		return pk, nil

	default:
		return nil, fmt.Errorf("unknown report source type")
	}
}

func (s *coverageService) GetMetadataList(ctx context.Context, req *coverage.GetMetadataListRequest) (*coverage.GetMetadataListResponse, error) {
	log.Printf("GetMetadataList called with type: %s, module: %s", req.Type, req.Module)

	var modules, branches, commits []string
	var err error

	if req.Type == "systest" || req.Type == "integrate" || req.Type == "integration" {
		if s.stStore == nil {
			return nil, status.Errorf(codes.FailedPrecondition, "systest store not initialized")
		}
		modules, branches, commits, err = s.stStore.GetMetadataList(ctx, req.Module, req.Branch)
	} else if req.Type == "unittest" || req.Type == "unit_test" || req.Type == "unit" || req.Type == "" {
		if s.utStore == nil {
			return nil, status.Errorf(codes.FailedPrecondition, "unittest store not initialized")
		}
		modules, branches, commits, err = s.utStore.GetMetadataList(ctx, req.Module, req.Branch)
	} else {
		log.Printf("GetMetadataList type not supported: %s", req.Type)
		return &coverage.GetMetadataListResponse{}, nil
	}

	if err != nil {
		log.Printf("GetMetadataList failed: type=%s, module=%s, err=%v", req.Type, req.Module, err)
		return nil, status.Errorf(codes.Internal, "failed to get metadata list: %v", err)
	}

	log.Printf("GetMetadataList success: type=%s, module=%s, branches_count=%d", req.Type, req.Module, len(branches))
	return &coverage.GetMetadataListResponse{
		Modules:  modules,
		Branches: branches,
		Commits:  commits,
	}, nil
}

func (s *coverageService) GetRootCoverage(ctx context.Context, req *coverage.GetRootCoverageRequest) (*coverage.GetRootCoverageResponse, error) {
	pk := partitionkey.NewReportKey("", "", "", "")
	if err := pk.Unmarshal(req.ReportId); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid report_id")
	}

	rep, err := s.mgr.Open(ctx, pk)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "report not found")
	}
	defer rep.Close(ctx)

	node := rep.FindNode("*")
	if node == nil {
		node = rep.FindNode("")
	}
	if node == nil {
		return nil, status.Errorf(codes.NotFound, "root node not found")
	}

	stat := node.GetStat()
	return &coverage.GetRootCoverageResponse{
		RootNode: &coverage.TreeNode{
			Name: node.Name(),
			Path: node.Path(),
			Type: 0, // Dir
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
				HasIncrement:   stat.HasIncrement,
			},
		},
	}, nil
}

func (s *coverageService) SearchNodes(ctx context.Context, req *coverage.SearchNodesRequest) (*coverage.SearchNodesResponse, error) {
	pk := partitionkey.NewReportKey("", "", "", "")
	if err := pk.Unmarshal(req.ReportId); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid report_id")
	}

	rep, err := s.mgr.Open(ctx, pk)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "report not found")
	}
	defer rep.Close(ctx)

	start := time.Now()
	matchedNodes, err := rep.Match(req.Keyword, req.IsIncrement)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "match nodes failed: %v", err)
	}
	matchDuration := time.Since(start)

	nodes := make([]*coverage.TreeNode, 0, len(matchedNodes))
	for _, node := range matchedNodes {
		nodeType := int32(1) // File
		if node.IsDir() {
			nodeType = 0 // Dir
		}

		// 恢复 Stat 数据返回，让前端显示正确的百分比和行数统计
		// 我们之前已经对 GetStat() 进行了内部优化 (缓存 hasIncr 标志)，
		// 这里虽然会触发递归计算，但由于 Match 结果已经是剪枝后的稀疏节点，计算量可控
		stat := node.GetStat()
		nodes = append(nodes, &coverage.TreeNode{
			Name: node.Name(),
			Path: node.Path(),
			Type: nodeType,
			Stat: &coverage.TreeNodeStat{
				Coverage:       stat.Coverage,
				CoverLines:     stat.CoverLines,
				InstrLines:     stat.InstrLines,
				IncrCoverage:   stat.IncrCoverage,
				IncrCoverLines: stat.IncrCoverLines,
				IncrInstrLines: stat.IncrInstrLines,
				HasIncrement:   node.HasIncrement(),
			},
		})
	}

	fmt.Printf("SearchNodes: keyword=%s, matched=%d, match_time=%v, total_time=%v\n",
		req.Keyword, len(matchedNodes), matchDuration, time.Since(start))

	return &coverage.SearchNodesResponse{
		Nodes: nodes,
	}, nil
}
