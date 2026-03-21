package sut

import (
	"context"
	"time"

	"github.com/shuaibizhang/codecoverage/internal/scheduler"
)

type ICronController interface {
	Run(ctx context.Context)
}

type ISutObserver interface {
	OnSutAdded(sut *sut)
}

type ITaskSchedulerWithSutObserver interface {
	ISutObserver
	scheduler.ITaskScheduler
}

type taskSchedulerWithSutObserver struct {
	taskScheduler  scheduler.ITaskScheduler
	cronController ICronController
}

var _ ITaskSchedulerWithSutObserver = (*taskSchedulerWithSutObserver)(nil)

func NewTaskSchedulerWithSutObserver(taskScheduler scheduler.ITaskScheduler, cronController ICronController) ITaskSchedulerWithSutObserver {
	return &taskSchedulerWithSutObserver{
		taskScheduler:  taskScheduler,
		cronController: cronController,
	}
}

func (t *taskSchedulerWithSutObserver) OnSutAdded(sut *sut) {
	// 当sut添加时，添加定时任务
	t.taskScheduler.AddTask(func(ctx context.Context) error {
		t.cronController.Run(ctx)
		return nil
	}, 5*time.Second)
}

func (t *taskSchedulerWithSutObserver) AddTask(fn func(ctx context.Context) error, interval time.Duration) {
	t.taskScheduler.AddTask(fn, interval)
}

func (t *taskSchedulerWithSutObserver) Run(ctx context.Context) {
	t.taskScheduler.Run(ctx)
}
