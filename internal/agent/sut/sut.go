package sut

import (
	"context"

	"github.com/shuaibizhang/codecoverage/logger"
)

const TagSUTService = "_sut_service"

type ISUTService interface {
	AddSUT(ctx context.Context, coverAddr, dataPath, language, module, branch, commitID, baseCommitID, BuildID string)
	GetSutMap() map[string]*sut
	IsReady() bool
}

// SUT，service Under Test，待收集测试覆盖率的服务单元
type sut struct {
	// 收集覆盖率时所需信息
	CoverAddr string // 待测服务插桩包覆盖率代理服务器地址
	DataPath  string // 待测服务路径

	// 待测服务的元数据信息，插桩时写入
	Language     string // 待测服务语言，不同语言有不同的处理器
	Module       string // 待测服务模块名
	Branch       string // 待测服务分支
	CommitID     string // 待测服务commit ID
	BaseCommitID string // 待测服务baseCommit ID
	BuildID      string // 待测服务build ID
}

var _ ISUTService = (*sutService)(nil)

type sutService struct {
	// key：模块名、待测单元
	sutMap       map[string]*sut
	sutObservers []ISutObserver
	logger       logger.Logger
}

func NewSUTService(sutObservers []ISutObserver, logger logger.Logger) ISUTService {
	return &sutService{
		sutMap:       make(map[string]*sut),
		sutObservers: sutObservers,
		logger:       logger,
	}
}

func (s *sutService) AddSUT(ctx context.Context, coverAddr, dataPath, language, module, branch, commitID, baseCommitID, BuildID string) {
	// 如果已存在该模块的sut，直接返回
	if _, isExisted := s.sutMap[module]; isExisted {
		s.logger.Infof(ctx, TagSUTService, "sut for module %s already existed, coverAddr: %s, dataPath: %s", module, coverAddr, dataPath)
		return
	}

	// 添加sut
	tmpSut := &sut{
		CoverAddr:    coverAddr,
		DataPath:     dataPath,
		Language:     language,
		Module:       module,
		Branch:       branch,
		CommitID:     commitID,
		BaseCommitID: baseCommitID,
		BuildID:      BuildID,
	}

	s.sutMap[module] = tmpSut

	// 通知observer, sut已添加
	for _, observer := range s.sutObservers {
		observer.OnSutAdded(tmpSut)
	}
	s.logger.Infof(ctx, TagSUTService, "sut for module %s added, coverAddr: %s, dataPath: %s, language: %s, branch: %s, commitID: %s, baseCommitID: %s, BuildID: %s", module, coverAddr, dataPath, language, branch, commitID, baseCommitID, BuildID)
}

func (s *sutService) IsReady() bool {
	return s.sutMap != nil && len(s.sutMap) > 0
}

func (s *sutService) GetSutMap() map[string]*sut {
	return s.sutMap
}
