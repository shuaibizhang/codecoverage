package processor

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/parnurzeal/gorequest"
	"github.com/shuaibizhang/codecoverage/internal/parser"
)

type GoProcessor struct {
	addr   string
	parser parser.Parser

	moduleInfo *ModuleInfo
	cacheData  parser.CoverDataMap
}

const (
	BaseCommitId = "Goc-Header-Basecommit-Id"
	BranchName   = "Goc-Header-Branch"
	BuildId      = "Goc-Header-Build-Id"
	CommitId     = "Goc-Header-Commit-Id"
	ModuleName   = "Goc-Header-Module"
	SessionId    = "Goc-Header-Session-Id"
)

const (
	DLTagProcessOnGo = "_cover_on_go"
)

var (
	testGoGetRawCoverage func() (string, http.Header, error)
	_                    IProcessor = (*GoProcessor)(nil)
)

func NewGoProcessor(addr string, parser parser.Parser) *GoProcessor {
	return &GoProcessor{
		addr:   addr,
		parser: parser,
	}
}

// GetFullCoverage 获取全量覆盖率数据
func (p *GoProcessor) GetFullCoverage(ctx context.Context, isAutotest bool) (*ModuleInfo, parser.CoverDataMap, error) {
	data, header, err := p.getRawCoverage(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("get raw coverage: %v", err)
	}

	// 从覆盖率数据中获取模块信息
	moduleInfo, err := p.getModuleInfo(header)
	if err != nil {
		return nil, nil, fmt.Errorf("get module info from header: %v", err)
	}

	// 解析覆盖率数据
	coverData, covErr := p.parser.Parse(data)
	if covErr != nil {
		return nil, nil, fmt.Errorf("parser cover data fail: %v", covErr)
	}

	return moduleInfo, coverData.CoverageData, nil
}

// GetIncrCoverage 获取增量覆盖率数据
func (p *GoProcessor) GetIncrCoverage(ctx context.Context, isAutotest bool) (*ModuleInfo, parser.CoverDataMap, bool, error) {
	moduleInfo, coverData, covErr := p.GetFullCoverage(ctx, isAutotest)
	if covErr != nil {
		return nil, nil, false, fmt.Errorf("get full coverage: %v", covErr)
	}

	defer func() {
		p.moduleInfo = moduleInfo
		p.cacheData = coverData
	}()

	if p.IsModuleInfoChanged(ctx, p.moduleInfo, moduleInfo) {
		return moduleInfo, coverData, false, nil
	}

	incrData, _, covErr := p.DiffCoverData(ctx, p.cacheData, coverData)
	if covErr != nil {
		return nil, nil, false, fmt.Errorf("get coverage diff: %v", covErr)
	}
	return moduleInfo, incrData, true, nil
}

func (p *GoProcessor) getRawCoverage(ctx context.Context) (string, http.Header, error) {
	if testGoGetRawCoverage != nil {
		return testGoGetRawCoverage()
	}

	// http://127.0.0.1:8080/v1/cover/profile
	url := fmt.Sprintf("http://%s/v1/cover/profile", p.addr)
	resp, respBody, errs := gorequest.New().Post(url).Type("json").End()
	if len(errs) > 0 {
		err := errors.Join(errs...)
		return "", nil, err
	}
	return respBody, resp.Header, nil
}

// CleanCache 清空用于计算增量覆盖率的缓存数据
func (p *GoProcessor) CleanCache() {
	p.moduleInfo = nil
	p.cacheData = nil
}

// IsModuleInfoChanged 模块信息是否发生变化
func (p *GoProcessor) IsModuleInfoChanged(ctx context.Context, oldInfo, newInfo *ModuleInfo) bool {
	if oldInfo == nil {
		return true
	}

	// 通过 session 判断服务是否重启
	if newInfo.SessionID == oldInfo.SessionID {
		return false
	}

	// 通过 build id 判断服务是否重新构建
	if newInfo.BuildID != oldInfo.BuildID {
	}
	return true
}

// DiffCoverData 计算覆盖率增量
func (p *GoProcessor) DiffCoverData(ctx context.Context, oldData, newData parser.CoverDataMap) (parser.CoverDataMap, bool, error) {
	if len(oldData) <= 0 {
		return newData, false, nil
	}
	incrCoverDataMap, _, covErr := parser.GetIncrChangeCoverMap(oldData, newData)
	if covErr != nil {
		return nil, false, covErr
	}
	return incrCoverDataMap, true, nil
}

// getModuleInfo
func (p *GoProcessor) getModuleInfo(header http.Header) (*ModuleInfo, error) {
	var res ModuleInfo
	res.Module = header.Get(ModuleName)
	res.Branch = header.Get(BranchName)
	res.Commit = header.Get(CommitId)
	res.BaseCommit = header.Get(BaseCommitId)
	res.BuildID = header.Get(BuildId)
	res.SessionID = header.Get(SessionId)

	if err := res.IsValid(); err != nil {
		return nil, err
	}

	return &res, nil
}

func (p *GoProcessor) IsReady(ctx context.Context) bool {
	return HttpProbe(fmt.Sprintf("http://%s/v1/cover/buildinfo", p.addr), 100*time.Millisecond)
}

func HttpProbe(url string, timeout time.Duration) bool {
	client := http.Client{
		Timeout: timeout,
	}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
