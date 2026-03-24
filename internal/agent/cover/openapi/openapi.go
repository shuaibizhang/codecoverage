package openapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/shuaibizhang/codecoverage/idl/cover-server/coverage"
	"github.com/shuaibizhang/codecoverage/idl/cover-server/register"
)

type CoverAPI interface {
	RegistryAgentInfo(ctx context.Context, req *register.AgentRegisterRequest) (*register.AgentRegisterResponse, error)
	UploadSystestCoverData(ctx context.Context, req *coverage.UploadSystestCoverDataRequest) (*coverage.UploadSystestCoverDataResponse, error)
}

type coverAPI struct {
	serverAddr string
	client     *http.Client
}

func NewCoverAPI(serverAddr string) CoverAPI {
	return &coverAPI{
		serverAddr: serverAddr,
		client:     &http.Client{},
	}
}

func (c *coverAPI) RegistryAgentInfo(ctx context.Context, req *register.AgentRegisterRequest) (*register.AgentRegisterResponse, error) {
	url := fmt.Sprintf("http://%s/api/v1/register/agent", c.serverAddr)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request err: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create request err: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request err: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result register.AgentRegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response err: %w", err)
	}

	return &result, nil
}

func (c *coverAPI) UploadSystestCoverData(ctx context.Context, req *coverage.UploadSystestCoverDataRequest) (*coverage.UploadSystestCoverDataResponse, error) {
	url := fmt.Sprintf("http://%s/api/v1/systest/upload", c.serverAddr)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request err: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create request err: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request err: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result coverage.UploadSystestCoverDataResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response err: %w", err)
	}

	return &result, nil
}
