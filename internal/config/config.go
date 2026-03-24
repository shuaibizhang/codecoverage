package config

import (
	"context"
	"fmt"
	"strings"

	"github.com/shuaibizhang/codecoverage/internal/config/apollo"
	"github.com/shuaibizhang/codecoverage/internal/oss"
	"github.com/shuaibizhang/codecoverage/logger"
	"github.com/shuaibizhang/codecoverage/store/db"
	"github.com/spf13/viper"
)

const TagConfig = "_config"

var (
	cfg Config
)

type Config struct {
	ServerConfig ServerConfig        `mapstructure:"server" yaml:"server"`
	OssConfig    oss.Config          `mapstructure:"oss" yaml:"oss"`
	GithubConfig GithubConfig        `mapstructure:"github" yaml:"github"`
	DbConfig     db.Config           `mapstructure:"mysql" yaml:"mysql"`
	AgentConfig  AgentConfig         `mapstructure:"agent" yaml:"agent"`
	ApolloConfig apollo.ApolloConfig `mapstructure:"apollo" yaml:"apollo"` // 新增 Apollo 配置
}

type ServerConfig struct {
	GrpcAddr string `mapstructure:"grpc_addr" yaml:"grpc_addr"`
	HttpAddr string `mapstructure:"http_addr" yaml:"http_addr"`
}

type AgentConfig struct {
	Addr            string `mapstructure:"addr" yaml:"addr"`
	CoverServerAddr string `mapstructure:"cover_server_addr" yaml:"cover_server_addr"`
}

type GithubConfig struct {
	Token string `mapstructure:"token" yaml:"token"`
	Owner string `mapstructure:"owner" yaml:"owner"`
}

// Init 初始化配置
func Init(ctx context.Context, configPath string) error {
	v := viper.New()

	// 1. 设置代码默认值 (兜底)
	v.SetDefault("apollo.enabled", false)
	v.SetDefault("mysql.host", "127.0.0.1")

	// 2. 绑定命令行参数 (最高优先级)
	// 使用 pflag 绑定，支持 --mysql.host 这种层级参数
	// pflag.String("mysql.host", "", "MySQL host")
	// pflag.String("apollo.addr", "", "Apollo server address")
	// pflag.String("apollo.app_id", "", "Apollo app id")
	// pflag.String("apollo.env", "", "Apollo environment")
	// pflag.String("apollo.cluster", "default", "Apollo cluster")
	// pflag.String("apollo.namespace", "application", "Apollo namespace")
	// pflag.Bool("apollo.enabled", false, "Enable Apollo config")
	// pflag.Parse()
	// v.BindPFlags(pflag.CommandLine)

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
	// 显式绑定，确保即使配置文件中没有这些 key，也能从环境变量读取
	v.BindEnv("github.token", "GITHUB_TOKEN")
	v.BindEnv("github.owner", "GITHUB_OWNER")
	v.BindEnv("apollo.secret", "APOLLO_SECRET")

	if err := v.ReadInConfig(); err != nil {
		fmt.Printf("Warning: failed to read local config file: %v\n", err)
	}

	// 5. 解析到全局 Config 对象
	if err := v.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("failed to unmarshal final config: %w", err)
	}

	// 6. 如果开启了 Apollo，则拉取远程配置并 Merge 到 Viper (优先级高于文件，但低于 ENV/CLI)
	if cfg.ApolloConfig.Enabled {
		logger.Default().Infof(ctx, TagConfig, "init Apollo client for app_id: %s, addr: %s", cfg.ApolloConfig.AppID, cfg.ApolloConfig.Addr)
		apolloCli, err := apollo.NewApolloClient(cfg.ApolloConfig)
		if err != nil {
			logger.Default().Errorf(context.Background(), TagConfig, "failed to create apollo client: %v", err)
			return fmt.Errorf("failed to create apollo client: %w", err)
		}
		// 从 Apollo 中获取名为 "github" 的 YAML 配置并解析到结构体
		logger.Default().Infof(ctx, TagConfig, "Fetching 'github' config from Apollo namespace: %s", cfg.ApolloConfig.Namespace)
		if err := apolloCli.UnmarshalYAML("github", &cfg.GithubConfig); err != nil {
			logger.Default().Errorf(context.Background(), TagConfig, "Warning: failed to load github config from apollo: %v", err)
		} else {
			logger.Default().Infof(context.Background(), TagConfig, "Successfully loaded github config from Apollo: token=%s, owner=%s",
				maskString(cfg.GithubConfig.Token), cfg.GithubConfig.Owner)
		}
	}

	logger.Default().Infof(ctx, TagConfig, "Config initialized successfully. Apollo enabled: %v", cfg.ApolloConfig.Enabled)
	return nil
}

func maskString(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + "****" + s[len(s)-2:]
}

// GetConfig 获取全局配置对象
func GetConfig() *Config {
	return &cfg
}
