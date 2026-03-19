package commands

import "github.com/spf13/cobra"

func Execute() error {
	// 定义根命令 cover-cli xxx
	rootCmd := &cobra.Command{
		Use:   "cover-cli",
		Short: "覆盖率采集命令行工具",
		Long:  "支持short live job相关应用的覆盖率采集，支持go",
	}

	// 注册全局持久化标志，开启详细日志输出
	// 普通的标志只对当前命令有效，持久化flags对其子命令也有效
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")

	// 注册子命令，路由子命令
	// 注册上报命令
	rootCmd.AddCommand(NewUploadCmd())

	// 返回执行结果
	return rootCmd.Execute()
}
