package controller

import (
	"context"
	"fmt"

	"github.com/shuaibizhang/codecoverage/internal/agent/cover"
	"github.com/shuaibizhang/codecoverage/internal/parser"
)

type CronController struct {
	coverCli cover.ICoverCli
	isPause  bool
}

func NewCronController(coverCli cover.ICoverCli) *CronController {
	return &CronController{
		coverCli: coverCli,
	}
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

	coverData, isIncr, err = c.coverCli.GetIncrCoverage(ctx, false)
	if err != nil {
		fmt.Printf("GetIncrCoverage err: %v\n", err)
		return
	}
	if coverData == nil || len(coverData.CoverageData) <= 0 {
		return
	}

	fmt.Printf("cover data is %+v\n, isIncr=%v", coverData, isIncr)
}
