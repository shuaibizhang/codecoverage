package config

import (
	"fmt"
	"strings"

	"github.com/shuaibizhang/codecoverage/internal/config/apollo"
	"github.com/shuaibizhang/codecoverage/internal/oss"
	"github.com/shuaibizhang/codecoverage/store/db"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	cfg Config
)

type Config struct {
	OssConfig    oss.Config          `mapstructure:"oss" yaml:"oss"`
	GithubConfig GithubConfig        `mapstructure:"github" yaml:"github"`
	DbConfig     db.Config           `mapstructure:"mysql" yaml:"mysql"`
	AgentConfig  AgentConfig         `mapstructure:"agent" yaml:"agent"`
	ApolloConfig apollo.ApolloConfig `mapstructure:"apollo" yaml:"apollo"` // 新增 Apollo 配置
}

type AgentConfig struct {
	Addr string `mapstructure:"addr" yaml:"addr"`
}

type GithubConfig struct {
	Token string `mapstructure:"token" yaml:"token"`
	Owner string `mapstructure:"owner" yaml:"owner"`
}

// Init 初始化配置
func Init(configPath string) error {
	v := viper.New()

	// 1. 设置代码默认值 (兜底)
	v.SetDefault("apollo.enabled", false)
	v.SetDefault("mysql.host", "127.0.0.1")

	// 2. 绑定命令行参数 (最高优先级)
	// 使用 pflag 绑定，支持 --mysql.host 这种层级参数
	pflag.String("mysql.host", "", "MySQL host")
	pflag.String("apollo.addr", "", "Apollo server address")
	pflag.String("apollo.app_id", "", "Apollo app id")
	pflag.String("apollo.env", "", "Apollo environment")
	pflag.String("apollo.cluster", "default", "Apollo cluster")
	pflag.String("apollo.namespace", "application", "Apollo namespace")
	pflag.Bool("apollo.enabled", false, "Enable Apollo config")
	pflag.Parse()
	v.BindPFlags(pflag.CommandLine)

	// 3. 加载本地配置文件 (基础配置)
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("dev")
		v.SetConfigType("toml")
		v.AddConfigPath("./conf")
	}

	// 4. 支持环境变量 (优先级高于文件)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		fmt.Printf("Warning: failed to read local config file: %v\n", err)
	}

	// 5. 解析初步配置以获取 Apollo 连接信息
	var tempCfg Config
	if err := v.Unmarshal(&tempCfg); err != nil {
		return fmt.Errorf("failed to unmarshal initial config: %w", err)
	}

	// 6. 如果开启了 Apollo，则拉取远程配置并 Merge 到 Viper (优先级高于文件，但低于 ENV/CLI)
	if tempCfg.ApolloConfig.Enabled {
		apolloCli, err := apollo.NewApolloClient(tempCfg.ApolloConfig)
		if err != nil {
			return fmt.Errorf("failed to create apollo client: %w", err)
		}
		// 从 Apollo 中获取名为 "github" 的 YAML 配置并解析到结构体
		if err := apolloCli.UnmarshalYAML("github", &cfg.GithubConfig); err != nil {
			fmt.Printf("Warning: failed to load github config from apollo: %v\n", err)
		}
	}

	// 7. 最终解析到全局 Config 对象
	if err := v.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("failed to unmarshal final config: %w", err)
	}

	fmt.Printf("Config initialized successfully. Apollo enabled: %v\n", cfg.ApolloConfig.Enabled)
	return nil
}

// GetConfig 获取全局配置对象
func GetConfig() *Config {
	return &cfg
}
