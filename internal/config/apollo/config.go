package apollo

type ApolloConfig struct {
	Enabled   bool   `mapstructure:"enabled" yaml:"enabled"`
	Addr      string `mapstructure:"addr" yaml:"addr"`
	AppID     string `mapstructure:"app_id" yaml:"app_id"`
	Env       string `mapstructure:"env" yaml:"env"`
	Cluster   string `mapstructure:"cluster" yaml:"cluster"`
	Namespace string `mapstructure:"namespace" yaml:"namespace"`
	Secret    string `mapstructure:"secret" yaml:"secret"`
}
