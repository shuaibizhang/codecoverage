package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/shuaibizhang/codecoverage/internal/buildinfo"
	"github.com/shuaibizhang/codecoverage/internal/oss"
	"github.com/shuaibizhang/codecoverage/internal/parser"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/storage/partitionkey"
	"github.com/spf13/cobra"
)

// NewUploadCmd 上报覆盖率数据命令
func NewUploadCmd() *cobra.Command {
	uploadCmd := &cobra.Command{
		Use:   "upload",
		Short: "上报覆盖率数据到cover-server",
		Long:  "上报行覆盖率数据到cover-server",
		RunE:  CommandRunEFunc(runUpload),
	}

	// stringP支持短-，string不支持短-
	uploadCmd.Flags().StringP("language", "l", "", "选择当前项目的语言 go/java/php/c++ 等")
	uploadCmd.Flags().StringP("coverpath", "c", "", "选择待收集覆盖率数据的路径")
	uploadCmd.Flags().StringP("diffPath", "d", "", "指定与baseCommit计算出的diff文件")
	uploadCmd.Flags().String("cover-server", "", "指定覆盖率服务地址")

	return uploadCmd
}

func runUpload(cmd *cobra.Command, _ []string) error {
	cmd.Println("version:", buildinfo.Version)
	cmd.Println("build time:", buildinfo.BuildTime)
	cmd.Println("build commit:", buildinfo.Commit)

	err := upload(cmd)
	if err != nil {
		cmd.Println("上传覆盖率数据失败:", err.Error())
		return err
	}

	return nil
}

func upload(cmd *cobra.Command) error {
	opts, err := parseUploadOptions(cmd)
	if err != nil {
		return err
	}

	// 1. 获取对象存储客户端
	ossCli, err := initOSSClient()
	if err != nil {
		return fmt.Errorf("failed to init oss client: %w", err)
	}

	// 2. 获取对应语言的解析器
	p, err := parser.ParserFactory(parser.LanguageType(opts.Language))
	if err != nil {
		return fmt.Errorf("failed to create parser: %w", err)
	}

	// 3. 扫描单测覆盖率文件，解析成归一化行覆盖率数据
	partitionKey := partitionkey.NewUnitTestNormalCovKey(opts.Module, opts.Branch, opts.Commit, opts.RunID)
	uploadCmdhelper := NewUploadCmdHelper(ossCli, opts.ServerAddr, p, partitionKey)
	covNormalInfo, err := uploadCmdhelper.parseDirectory(opts.CoverPath)
	if err != nil {
		return fmt.Errorf("failed to parse directory: %w", err)
	}

	// 4. 上报覆盖率数据
	if err := uploadCmdhelper.uploadCoverage(cmd.Context(), opts, covNormalInfo); err != nil {
		cmd.PrintErrf("上报覆盖率数据失败: %v\n", err)
	}

	// 5. 上报 diff 文件
	if opts.DiffPath != "" {
		if err := uploadCmdhelper.uploadDiff(cmd.Context(), opts.DiffPath); err != nil {
			cmd.PrintErrf("上报 diff 文件失败: %v\n", err)
		}
	}

	// 6. 上报到cover_server
	if err := uploadCmdhelper.UploadToCoverServer(cmd.Context(), covNormalInfo); err != nil {
		cmd.PrintErrf("上报到cover_server失败: %v\n", err)
	}

	return nil
}

type uploadOptions struct {
	Language   string
	CoverPath  string
	DiffPath   string
	ServerAddr string
	Module     string
	Branch     string
	Commit     string
	BaseCommit string
	RunID      string
}

func parseUploadOptions(cmd *cobra.Command) (*uploadOptions, error) {
	covPath := cmd.Flag("coverpath").Value.String()
	if covPath == "" {
		return nil, errors.New("cover path is required")
	}

	return &uploadOptions{
		Language:   cmd.Flag("language").Value.String(),
		CoverPath:  covPath,
		DiffPath:   cmd.Flag("diffPath").Value.String(),
		ServerAddr: cmd.Flag("cover-server").Value.String(),
		Module:     strings.TrimSpace(os.Getenv("MODULE")),
		Branch:     strings.TrimSpace(os.Getenv("BRANCH")),
		Commit:     strings.TrimSpace(os.Getenv("COMMIT")),
		BaseCommit: strings.TrimSpace(os.Getenv("BASE_COMMIT")),
		RunID:      strings.TrimSpace(os.Getenv("RUN_ID")),
	}, nil
}

func initOSSClient() (oss.OSS, error) {
	ossCfg := oss.Config{
		Endpoint:        "http://49.233.216.158/oss/",
		AccessKeyID:     "admin",
		SecretAccessKey: "password123",
		UseSSL:          false,
		BucketName:      "coverage-reports",
	}
	return oss.NewMinioOSS(ossCfg)
}

type uploadCmdHelper struct {
	ossCli          oss.OSS
	coverServerAddr string
	parser          parser.Parser
	partitionKey    partitionkey.PartitionKey
}

func NewUploadCmdHelper(ossCli oss.OSS, coverServerAddr string, p parser.Parser, partionKey partitionkey.PartitionKey) *uploadCmdHelper {
	return &uploadCmdHelper{
		ossCli:          ossCli,
		coverServerAddr: coverServerAddr,
		parser:          p,
		partitionKey:    partionKey,
	}
}

func (u *uploadCmdHelper) parseDirectory(covPath string) (*parser.CovNormalInfo, error) {
	if u.parser == nil {
		return nil, errors.New("parser is nil")
	}

	coverFiles, err := u.parser.ScanCoverageFiles(covPath)
	if err != nil {
		return nil, fmt.Errorf("scan coverage files failed: %w", err)
	}
	if len(coverFiles) == 0 {
		return nil, errors.New("no coverage files found")
	}

	// 解析覆盖率文件，获取行覆盖率信息
	covNormalInfo, err := u.parser.ParseMultiFiles(coverFiles)
	if err != nil {
		return nil, fmt.Errorf("parse coverage files failed: %w", err)
	}
	return covNormalInfo, nil
}

func (u *uploadCmdHelper) uploadCoverage(ctx context.Context, opts *uploadOptions, covNormalInfo *parser.CovNormalInfo) error {
	// 补充覆盖率元数据信息
	covNormalInfo.Module = opts.Module
	covNormalInfo.Branch = opts.Branch
	covNormalInfo.Commit = opts.Commit
	covNormalInfo.BaseCommit = opts.BaseCommit
	covNormalInfo.UnittestRunID = opts.RunID
	covNormalInfo.HostName, _ = os.Hostname()

	// 序列化
	jsonStr, err := json.Marshal(covNormalInfo)
	if err != nil {
		return fmt.Errorf("marshal coverage data failed: %w", err)
	}

	fileName := fmt.Sprintf("%s.json", u.partitionKey.RealPathPrefix())
	return u.putToOSS(ctx, fileName, jsonStr)
}

func (u *uploadCmdHelper) uploadDiff(ctx context.Context, diffPath string) error {
	diffData, err := os.ReadFile(diffPath)
	if err != nil {
		return fmt.Errorf("read diff file failed: %w", err)
	}

	fileName := fmt.Sprintf("%s.diff", u.partitionKey.RealPathPrefix())
	return u.putToOSS(ctx, fileName, diffData)
}

func (u *uploadCmdHelper) putToOSS(ctx context.Context, fileName string, data []byte) error {
	if u.ossCli == nil {
		return errors.New("oss client is nil")
	}

	return u.ossCli.PutObject(ctx, "coverage-reports", fileName, bytes.NewReader(data), int64(len(data)))
}

func (u *uploadCmdHelper) UploadToCoverServer(ctx context.Context, info *parser.CovNormalInfo) error {
	if u.coverServerAddr == "" {
		return nil // 如果未指定服务地址，则不进行上报
	}

	// 1. 构造请求
	pkStr, _ := u.partitionKey.Marshal()
	reqBody := map[string]interface{}{
		"language":                        info.Language,
		"module":                          info.Module,
		"branch":                          info.Branch,
		"commit":                          info.Commit,
		"base_commit":                     info.BaseCommit,
		"run_id":                          info.UnittestRunID,
		"normal_cover_data_partition_key": pkStr,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}
	log.Printf("Uploading report to %s, payload size: %.2f MB", u.coverServerAddr, float64(len(jsonData))/(1024*1024))

	// 2. 发送请求 (gRPC Gateway HTTP POST)
	url := fmt.Sprintf("%s/api/v1/unittest/upload", strings.TrimRight(u.coverServerAddr, "/"))
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 增加超时时间到 2 分钟，以支持大数据量上报
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call cover server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server return status %d, body: %s", resp.StatusCode, string(body))
	}

	// 3. 解析响应
	var uploadResp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !uploadResp.Success {
		return fmt.Errorf("server return error: %s", uploadResp.Message)
	}

	return nil
}
