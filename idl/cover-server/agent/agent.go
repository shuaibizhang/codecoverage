package agent

type AgentRegisterRequest struct {
	// 填写需要的字段
}

type AgentRegisterResponse struct {
	OssConfig *OssConfig `json:"oss_config"`
}

type OssConfig struct {
	AccessKeyId string `json:"access_key_id"`
	Secret      string `json:"secret"`
	Bucket      string `json:"bucket"`
	Addr        string `json:"addr"`
	UseSsl      bool   `json:"use_ssl"`
}
