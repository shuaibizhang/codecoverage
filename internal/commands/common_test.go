package commands

import (
	"errors"
	"fmt"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestCommandRunEFunc_PanicRecovery(t *testing.T) {
	// 准备一个会发生 panic 的函数
	panicFunc := func(cmd *cobra.Command, args []string) error {
		panic("test panic")
	}

	// 使用装饰器包装
	wrappedFunc := CommandRunEFunc(panicFunc)

	// 模拟 cobra.Command
	cmd := &cobra.Command{}

	// 执行包装后的函数，应该能够恢复 panic
	// 虽然 wrappedFunc 本身会返回 nil (因为 panic 后没执行到 return)，
	// 但关键是它不应该崩溃当前进程。
	assert.NotPanics(t, func() {
		_ = wrappedFunc(cmd, nil)
	})
}

func TestCommandRunEFunc_NormalExecution(t *testing.T) {
	// 准备一个正常的函数
	normalFunc := func(cmd *cobra.Command, args []string) error {
		return errors.New("normal error")
	}

	// 使用装饰器包装
	wrappedFunc := CommandRunEFunc(normalFunc)

	// 模拟 cobra.Command
	cmd := &cobra.Command{}

	// 执行包装后的函数，应该返回预期的错误
	err := wrappedFunc(cmd, nil)
	assert.Error(t, err)
	assert.Equal(t, "normal error", err.Error())
}

func TestCommandRunEFunc_ContextSetup(t *testing.T) {
	// 准备一个检查 context 的函数
	checkCtxFunc := func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if ctx == nil {
			return fmt.Errorf("context is nil")
		}
		return nil
	}

	// 使用装饰器包装
	wrappedFunc := CommandRunEFunc(checkCtxFunc)

	// 模拟 cobra.Command
	cmd := &cobra.Command{}

	// 执行包装后的函数
	err := wrappedFunc(cmd, nil)
	assert.NoError(t, err)
}
