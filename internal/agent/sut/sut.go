package sut

type ISUTService interface {
	AddSUT(coverAddr, dataPath, language, module, branch, commitID, baseCommitID, BuildID string)
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
}

func NewSUTService(sutObservers []ISutObserver) ISUTService {
	return &sutService{
		sutMap:       make(map[string]*sut),
		sutObservers: sutObservers,
	}
}

func (s *sutService) AddSUT(coverAddr, dataPath, language, module, branch, commitID, baseCommitID, BuildID string) {
	// 如果已存在该模块的sut，直接返回
	if _, isExisted := s.sutMap[module]; isExisted {
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
}

func (s *sutService) IsReady() bool {
	return s.sutMap != nil && len(s.sutMap) > 0
}

func (s *sutService) GetSutMap() map[string]*sut {
	return s.sutMap
}
