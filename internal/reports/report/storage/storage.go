package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/shuaibizhang/codecoverage/internal/reports/report"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/storage/coder"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/storage/datasource"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/storage/partitionkey"
	reportlock "github.com/shuaibizhang/codecoverage/internal/reports/report/storage/report-lock"
)

type storage struct {
	// 元数据数据源
	metaSource datasource.DataSource
	// 行覆盖数据数据源
	coverLineSource datasource.DataSource

	// 行覆盖数据编解码器
	coverLineEncoder coder.CoverLineEncoder
	coverLineDecoder coder.CoverLineDecoder

	// 报告数据编解码器
	reportEncoder coder.ReportEncoder
	reportDecoder coder.ReportDecoder

	// 报告数据锁
	reportLock reportlock.ReportLock
}

// NewStorage 创建存储实例
func NewStorage(metaSource, coverLineSource datasource.DataSource, reportLock reportlock.ReportLock) report.Storage {
	return &storage{
		metaSource:       metaSource,
		coverLineSource:  coverLineSource,
		reportLock:       reportLock,
		coverLineEncoder: coder.NewCoverLineEncoder(),
		coverLineDecoder: coder.NewCoverLineDecoder(nil),
		reportEncoder:    coder.NewReportEncoder(metaSource),
		reportDecoder:    coder.NewReportDecoder(metaSource),
	}
}

func (s *storage) SetCoverLine(ctx context.Context, pk partitionkey.PartitionKey, coverLines []int32, addedLines []uint32) (partitionkey.PartitionKey, error) {
	// 1、获取分布式锁
	if err := s.reportLock.Lock(ctx); err != nil {
		return nil, err
	}
	defer s.reportLock.Unlock(ctx)

	// 2、编码覆盖行
	coverLineData, err := s.coverLineEncoder.Encode(coverLines, addedLines)
	if err != nil {
		return nil, err
	}

	// 3、存储覆盖行
	// 获取当前文件末尾的偏移量
	offset, err := s.coverLineSource.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}

	if _, err := s.coverLineSource.WriteAt(coverLineData, offset); err != nil {
		return nil, err
	}

	// 4、同步数据源（如果是 OSS 数据源，则触发上传）
	// 注意：这里为了性能，不在每次设置覆盖行时都同步到 OSS，而是在 SetReport 或 Close 时统一同步
	// if err := s.coverLineSource.Sync(); err != nil {
	// 	return nil, err
	// }

	// 设置偏移量到 PartitionKey
	pk.SetOffset(offset)

	return pk, nil
}

func (s *storage) GetCoverLine(ctx context.Context, pk partitionkey.PartitionKey) ([]int32, []uint32, error) {
	// 1、读取行覆盖率头部信息 (16字节)
	headerData := make([]byte, 16)
	_, err := s.coverLineSource.ReadAt(headerData, int64(pk.Offset()))
	if err != nil {
		return nil, nil, err
	}

	// 使用 BlockHeader 解析头部，避免硬编码
	var header coder.BlockHeader
	if _, err := header.ReadFrom(bytes.NewReader(headerData)); err != nil {
		return nil, nil, err
	}

	// 2、根据解析出的 TotalLines 读取完整的块数据 (Header + Lines)
	blockSize := 16 + int64(header.TotalLines)*4
	fullData := make([]byte, blockSize)
	_, err = s.coverLineSource.ReadAt(fullData, pk.Offset())
	if err != nil {
		return nil, nil, err
	}

	// 3、解码覆盖行
	decoder := coder.NewCoverLineDecoder(fullData)
	return decoder.DecodeRawCoverLine()
}

func (s *storage) GetCoverLineWithFlag(ctx context.Context, pk partitionkey.PartitionKey) ([]uint32, error) {
	// 1、读取行覆盖率头部信息 (16字节)
	headerData := make([]byte, 16)
	_, err := s.coverLineSource.ReadAt(headerData, int64(pk.Offset()))
	if err != nil {
		return nil, err
	}

	// 使用 BlockHeader 解析头部
	var header coder.BlockHeader
	if _, err := header.ReadFrom(bytes.NewReader(headerData)); err != nil {
		return nil, err
	}

	// 2、根据解析出的 TotalLines 读取完整的块数据 (Header + Lines)
	blockSize := 16 + int64(header.TotalLines)*4
	fullData := make([]byte, blockSize)
	_, err = s.coverLineSource.ReadAt(fullData, pk.Offset())
	if err != nil {
		return nil, err
	}

	// 3、解码带标识的覆盖行
	decoder := coder.NewCoverLineDecoder(fullData)
	return decoder.DecodeCoverLine()
}

func (s *storage) SetReport(ctx context.Context, pk partitionkey.PartitionKey, report report.CoverReport) (partitionkey.PartitionKey, error) {
	// 1、获取分布式锁
	if err := s.reportLock.Lock(ctx); err != nil {
		return nil, err
	}
	defer s.reportLock.Unlock(ctx)

	// 2、编码报告
	// 注意：这里编码器会将数据写入 metaSource
	if err := s.metaSource.Truncate(0); err != nil {
		return nil, err
	}
	if _, err := s.metaSource.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	if err := s.reportEncoder.Encode(report); err != nil {
		return nil, err
	}

	// 3、同步数据源（如果是 OSS 数据源，则触发上传）
	if err := s.metaSource.Sync(); err != nil {
		return nil, err
	}
	if err := s.coverLineSource.Sync(); err != nil {
		return nil, err
	}

	// 4、返回报告句柄
	return pk, nil
}

func (s *storage) LoadReport(ctx context.Context, pk partitionkey.PartitionKey, report report.CoverReport) error {
	// 1、解码报告（从 metaSource 读取）并填充到传入的 report 对象
	return s.reportDecoder.Decode(report)
}

func (s *storage) GetMetaSource() datasource.DataSource {
	return s.metaSource
}

func (s *storage) Close() error {
	var errs []error
	if s.metaSource != nil {
		if err := s.metaSource.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if s.coverLineSource != nil {
		if err := s.coverLineSource.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to close storage: %v", errs)
	}
	return nil
}
