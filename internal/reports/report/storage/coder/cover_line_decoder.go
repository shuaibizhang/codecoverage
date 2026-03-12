package coder

import (
	"bytes"
	"io"
)

type CoverLineDecoder interface {
	// 解码成原始覆盖率行 （int32数组， -1 标识非指令行，其余表示覆盖次数）
	DecodeRawCoverLine() (coverLines []int32, addedlines []uint32, err error)
	// 解码成自包含覆盖率信息的覆盖率数组 (1 << 31 位为是否是指令行的标识位， 1 << 30 位为是否是增量行的标识位)
	DecodeCoverLine() ([]uint32, error)
}

type defaultCoverLineDecoder struct {
	data []byte
}

func NewCoverLineDecoder(data []byte) CoverLineDecoder {
	return &defaultCoverLineDecoder{data: data}
}

func (d *defaultCoverLineDecoder) DecodeRawCoverLine() (coverLines []int32, addedlines []uint32, err error) {
	uintLines, err := d.DecodeCoverLine()
	if err != nil {
		return nil, nil, err
	}
	coverLines, addedlines = DecodeUintLines(uintLines)
	return coverLines, addedlines, nil
}

func DecodeUintLines(uintLines []uint32) (coverLines []int32, addedlines []uint32) {
	coverLines = make([]int32, len(uintLines))
	addedlines = make([]uint32, 0)
	for i, val := range uintLines {
		// 1. 处理指令行标识 (MaskInstrLine)
		if val&MaskInstrLine != 0 {
			coverLines[i] = -1
		} else {
			// 2. 处理覆盖次数 (MaskCoverCount)
			// MaskCoverCount 掩码排除了 31 位和 30 位
			coverLines[i] = int32(val & MaskCoverCount)
		}

		// 3. 处理增量行标识 (MaskIncrLine)
		if val&MaskIncrLine != 0 {
			// 行号从 1 开始
			addedlines = append(addedlines, uint32(i+1))
		}
	}
	return coverLines, addedlines
}

func (d *defaultCoverLineDecoder) DecodeCoverLine() ([]uint32, error) {
	if len(d.data) < blockHeaderSize {
		return nil, io.ErrUnexpectedEOF
	}

	header := &BlockHeader{}
	if _, err := header.ReadFrom(bytes.NewReader(d.data)); err != nil {
		return nil, err
	}

	expectedSize := blockHeaderSize + int(header.TotalLines)*int32Size
	if len(d.data) < expectedSize {
		return nil, io.ErrUnexpectedEOF
	}

	return ParseDataBlock(d.data[blockHeaderSize:], int(header.TotalLines)), nil
}
