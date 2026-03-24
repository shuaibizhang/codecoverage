package controller

import (
	"encoding/json"
	"net/http"

	"github.com/shuaibizhang/codecoverage/idl/cover-server/register"
	"github.com/shuaibizhang/codecoverage/internal/oss"
	"github.com/shuaibizhang/codecoverage/logger"
)

// AgentController 处理 Agent 相关的请求
type AgentController struct {
	ossCfg oss.Config
}

func NewAgentController(ossCfg oss.Config) *AgentController {
	return &AgentController{ossCfg: ossCfg}
}

// RegisterAgent 处理来自 Agent 的注册请求
func (c *AgentController) RegisterAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req register.AgentRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Default().Error("failed to decode agent register request", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 目前注册逻辑比较简单，直接返回服务器的 OSS 配置给 Agent
	// 这样 Agent 就知道往哪里上传覆盖率数据了
	resp := register.AgentRegisterResponse{
		OssConfig: &register.OssConfig{
			Endpoint:        c.ossCfg.Endpoint,
			AccessKeyId:     c.ossCfg.AccessKeyID,
			SecretAccessKey: c.ossCfg.SecretAccessKey,
			UseSsl:          c.ossCfg.UseSSL,
			BucketName:      c.ossCfg.BucketName,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Default().Error("failed to encode agent register response", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	logger.Default().Info("agent registered successfully")
}
