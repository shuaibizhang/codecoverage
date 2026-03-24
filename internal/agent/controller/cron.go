package controller

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/shuaibizhang/codecoverage/idl/cover-server/coverage"
	"github.com/shuaibizhang/codecoverage/internal/agent/cover"
	"github.com/shuaibizhang/codecoverage/internal/agent/cover/openapi"
	"github.com/shuaibizhang/codecoverage/internal/oss"
	"github.com/shuaibizhang/codecoverage/internal/parser"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/storage/partitionkey"
	"github.com/shuaibizhang/codecoverage/logger"
)

const TagCronController = "_cron_controller"

// CronController 定时任务控制器
type CronController struct {
	coverCli   cover.ICoverCli
	isPause    bool
	ossCli     oss.OSS
	bucketName string
	coverApi   openapi.CoverAPI
	logger     logger.Logger
}

func NewCronController(coverCli cover.ICoverCli, logger logger.Logger) *CronController {
	return &CronController{
		coverCli: coverCli,
		logger:   logger,
	}
}

func (c *CronController) SetOSSClient(ossCli oss.OSS, bucketName string) {
	c.ossCli = ossCli
	c.bucketName = bucketName
}

func (c *CronController) SetCoverAPI(coverApi openapi.CoverAPI) {
	c.coverApi = coverApi
}

func (c *CronController) Run(ctx context.Context) {
	if c.isPause {
		return
	}

	var (
		err       error
		coverData *parser.CovNormalInfo
		isIncr    bool
	)

	c.logger.Infof(ctx, TagCronController, "get incr coverage...")

	coverData, isIncr, err = c.coverCli.GetIncrCoverage(ctx, false)
	if err != nil {
		c.logger.Errorf(ctx, TagCronController, "get incr coverage failed! err: %v", err)
		return
	}
	if coverData == nil || len(coverData.CoverageData) <= 0 {
		c.logger.Infof(ctx, TagCronController, "coverData is nil or empty, need not upload!")
		return
	}

	info := &coverage.UploadSystestCoverDataRequest{
		Language:    "go",
		Module:      coverData.Module,
		Branch:      coverData.Branch,
		Commit:      coverData.Commit,
		BaseCommit:  coverData.BaseCommit,
		CreatedTime: time.Now().Format(time.DateTime),
	}

	// 如果是增量数据且大小没超过限制，直接上报
	if isIncr && !isDataSizeExceedLimit(coverData) {
		info.CoverData = convertToProtoCoverData(coverData.CoverageData)
	} else {
		// 全量或者是增量但数据过大，上报到 OSS
		pk := partitionkey.NewSystestNormalCovKey(info.Module, info.Branch, info.Commit)
		path := pk.RealPathPrefix() + ".json"

		data, err := json.Marshal(coverData)
		if err != nil {
			c.logger.Errorf(ctx, TagCronController, "marshal coverData err: %v", err)
			return
		}

		err = c.ossCli.PutObject(ctx, c.bucketName, path, strings.NewReader(string(data)), int64(len(data)))
		if err != nil {
			c.logger.Errorf(ctx, TagCronController, "PutObject err: %v", err)
			return
		}

		pkStr, _ := pk.Marshal()
		info.NormalCoverDataPartitionKey = pkStr
		c.logger.Infof(ctx, TagCronController, "upload cover data to oss success! cover data is %+v", coverData)
	}

	_, err = c.coverApi.UploadSystestCoverData(ctx, info)
	if err != nil {
		c.logger.Errorf(ctx, TagCronController, "upload systest cover data to cover server failed! err: %v", err)
		return
	}
	c.logger.Infof(ctx, TagCronController, "upload systest cover data to cover server success! req: %+v", info)
}

func convertToProtoCoverData(data parser.CoverDataMap) map[string]*coverage.CoverData {
	res := make(map[string]*coverage.CoverData)
	for k, v := range data {
		res[k] = &coverage.CoverData{
			TotalLines:    uint32(v.TotalLines),
			CoverLines:    uint32(v.CoverLines),
			InstrLines:    uint32(v.InstrLines),
			CoverLineData: v.CoverLineData,
		}
	}
	return res
}

func isDataSizeExceedLimit(data *parser.CovNormalInfo) bool {
	if data == nil {
		return false
	}

	bytes, _ := json.Marshal(data)

	return len(bytes) >= 1*1024*1024
}
