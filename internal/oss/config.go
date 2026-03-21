package oss

// Config MinIO 配置
type Config struct {
	Endpoint        string `mapstructure:"endpoint" yaml:"endpoint"`
	AccessKeyID     string `mapstructure:"access_key" yaml:"access_key"`
	SecretAccessKey string `mapstructure:"secret_key" yaml:"secret_key"`
	UseSSL          bool   `mapstructure:"use_ssl" yaml:"use_ssl"`
	BucketName      string `mapstructure:"bucket_name" yaml:"bucket_name"`
}
