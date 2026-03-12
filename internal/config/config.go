package config

import (
	"fmt"
	"strings"

	"github.com/shuaibizhang/codecoverage/internal/oss"
	"github.com/spf13/viper"
)

var (
	cfg Config
)

type Config struct {
	OssConfig    oss.Config   `mapstructure:"oss"`
	GithubConfig GithubConfig `mapstructure:"github"`
}

type GithubConfig struct {
	Token string `mapstructure:"token"`
	Owner string `mapstructure:"owner"`
}

// Init 初始化配置
func Init(configPath string) error {
	v := viper.New()

	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("dev")
		v.SetConfigType("toml")
		v.AddConfigPath("./conf")
	}

	// 支持环境变量
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	if err := v.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	fmt.Printf("Config initialized successfully from: %s\n", v.ConfigFileUsed())
	return nil
}

// GetConfig 获取全局配置对象
func GetConfig() *Config {
	return &cfg
}
