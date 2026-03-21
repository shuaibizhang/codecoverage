package scheduler

import (
	"context"
	"fmt"
	"time"
)

// 定时任务组件
type ITaskScheduler interface {
	AddTask(fn func(ctx context.Context) error, interval time.Duration)
	Run(ctx context.Context) // 执行任务组件，进行调度
}

type task struct {
	fn       func(ctx context.Context) error
	interval time.Duration
	lastRun  time.Time
}

// 定时任务调度器
type taskScheduler struct {
	isPause     bool
	tasks       []*task
	schedPeriod time.Duration
	isFirstRun  bool
}

func NewTaskScheduler() ITaskScheduler {
	return &taskScheduler{
		isPause:     false,
		tasks:       make([]*task, 0),
		schedPeriod: time.Second,
		isFirstRun:  true,
	}
}

func (t *taskScheduler) AddTask(fn func(ctx context.Context) error, interval time.Duration) {
	t.tasks = append(t.tasks, &task{
		fn:       fn,
		interval: interval,
		lastRun:  time.Now(),
	})
}

func (t *taskScheduler) Run(ctx context.Context) {
	// 一秒钟调度一次
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// 循环调度所有任务
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 遍历所有任务
			for _, task := range t.tasks {
				if t.isFirstRun || time.Since(task.lastRun) >= task.interval {
					t.isFirstRun = false
					// 执行调度任务
					err := task.fn(ctx)
					if err != nil {
						fmt.Println(err)
					}
					// 更新任务被调度事件
					task.lastRun = time.Now()
				}
			}
		}
	}
}
