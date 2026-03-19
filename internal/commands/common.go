package commands

import (
	"context"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/spf13/cobra"
)

// CommandRunEFunc 装饰器，给 func(cmd *cobra.Command, args []string) error 函数添加信号处理、Context 设置和 Panic 恢复等功能
func CommandRunEFunc(fn func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// 1. 设置 context 和信号处理
		// 使用 cmd.Context() 作为父 context，如果为 nil 则使用 context.Background()
		parentCtx := cmd.Context()
		if parentCtx == nil {
			parentCtx = context.Background()
		}
		ctx, stop := signal.NotifyContext(parentCtx, syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		cmd.SetContext(ctx)

		// 2. 监听信号
		go func() {
			<-ctx.Done()
			if ctx.Err() != nil && ctx.Err() != context.Canceled {
				cmd.PrintErrln("\n收到退出信号，优雅退出")
			}
		}()

		// 3. Panic 恢复
		defer func() {
			if r := recover(); r != nil {
				cmd.PrintErrf("panic recovered: %v\n", r)
				// 打印堆栈信息
				buf := make([]byte, 4096)
				n := runtime.Stack(buf, false)
				cmd.PrintErrf("stack trace:\n%s\n", buf[:n])
			}
		}()

		// 4. 执行原函数
		return fn(cmd, args)
	}
}
