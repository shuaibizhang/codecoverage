package apollo

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/apolloconfig/agollo/v4"
	"github.com/apolloconfig/agollo/v4/env/config"
	"gopkg.in/yaml.v3"
)

type ApolloCli interface {
	GetString(key string, defaultValue string) string
	GetInt(key string, defaultValue int) int
	GetBool(key string, defaultValue bool) bool
	// UnmarshalJSON 将 JSON 格式的配置项解析到结构体
	UnmarshalJSON(key string, target interface{}) error
	// UnmarshalYAML 将 YAML 格式的配置项解析到结构体
	UnmarshalYAML(key string, target interface{}) error
}

type apolloClient struct {
	client agollo.Client
	ns     string
}

func NewApolloClient(cfg ApolloConfig) (ApolloCli, error) {
	c := &config.AppConfig{
		AppID:          cfg.AppID,
		Cluster:        cfg.Cluster,
		NamespaceName:  cfg.Namespace,
		IP:             cfg.Addr,
		IsBackupConfig: true,
	}

	client, err := agollo.StartWithConfig(func() (*config.AppConfig, error) {
		return c, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start apollo client: %w", err)
	}

	return &apolloClient{
		client: client,
		ns:     cfg.Namespace,
	}, nil
}

func (a *apolloClient) GetString(key string, defaultValue string) string {
	val := a.client.GetConfig(a.ns).GetStringValue(key, defaultValue)
	return val
}

func (a *apolloClient) GetInt(key string, defaultValue int) int {
	valStr := a.client.GetConfig(a.ns).GetValue(key)
	if valStr == "" {
		return defaultValue
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultValue
	}
	return val
}

func (a *apolloClient) GetBool(key string, defaultValue bool) bool {
	valStr := a.client.GetConfig(a.ns).GetValue(key)
	if valStr == "" {
		return defaultValue
	}
	val, err := strconv.ParseBool(valStr)
	if err != nil {
		return defaultValue
	}
	return val
}

func (a *apolloClient) UnmarshalJSON(key string, target interface{}) error {
	valStr := a.client.GetConfig(a.ns).GetValue(key)
	if valStr == "" {
		return fmt.Errorf("config key %s not found", key)
	}
	return json.Unmarshal([]byte(valStr), target)
}

func (a *apolloClient) UnmarshalYAML(key string, target interface{}) error {
	valStr := a.client.GetConfig(a.ns).GetValue(key)
	if valStr == "" {
		return fmt.Errorf("config key %s not found", key)
	}
	return yaml.Unmarshal([]byte(valStr), target)
}

func (a *apolloClient) GetConfig(key string) (string, error) {
	val := a.client.GetConfig(a.ns).GetValue(key)
	if val == "" {
		return "", fmt.Errorf("config key %s not found in namespace %s", key, a.ns)
	}
	return val, nil
}
