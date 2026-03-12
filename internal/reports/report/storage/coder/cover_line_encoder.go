package coder

import (
	"encoding/binary"
	"fmt"
	"io"
	"slices"
	"time"
)

/*
a、行覆盖率数据编码协议（.cda文件，每个文件行覆盖率数据 = 二进制header + 数组 ）
| Magic(2B) | Version(1B) | Reserved(1B) | TotalLines(4B) | Timestamp(8B) |
|             Line1(4B)                  |   .......      |   LineN(4B)   |
|                                ....                                     |       .....                                |
| Magic(2B) | Version(1B) | Reserved(1B) | TotalLines(4B) | Timestamp(8B) |
|             Line1(4B)                  |   .......      |   LineN(4B)   |

b、行覆盖数组编码方案：
1、行编码：
1）借用uint32的 1 << 31 位为是否是指令行的标识位
2）借用uint32的 1 << 30 位为是否是增量行的标识位
3）剩余 30 位为覆盖次数掩码
2、列编码：
修改编码协议，header里添加incrBitMap、instrBitMap数据长度，同时编码相关位图。
最终使用行编码，只需一次随机存取即可。同时每行自编码增量信息、是否指令行信息，方便前端解析。
*/

const (
	blockHeaderMagic = 0x2323
	blockHeaderSize  = 16 // 2+1+1+4+8
	int32Size        = 4

	// uint32 编码掩码
	MaskInstrLine  uint32 = 1 << 31    // 1 << 31 位为是否是指令行的标识位
	MaskIncrLine   uint32 = 1 << 30    // 1 << 30 位为是否是增量行的标识位
	MaskCoverCount uint32 = 0x3FFFFFFF // 其余 30 位为覆盖次数掩码
	MaxCoverCount  int32  = 0x3FFFFFFF // 最大覆盖次数 (1073741823)
)

var byteOrder = binary.BigEndian

// BlockHeader 在.cda文件中, 表示一块数据的header
type BlockHeader struct {
	Magic      uint16
	Version    uint8
	Reserved   uint8
	TotalLines uint32
	Timestamp  uint64
}

// NewBlockHeader 创建一个新的 BlockHeader
func NewBlockHeader(totalLines uint32) *BlockHeader {
	return &BlockHeader{
		Magic:      blockHeaderMagic,
		Version:    1,
		Reserved:   0,
		TotalLines: totalLines,
		Timestamp:  uint64(time.Now().Unix()),
	}
}

// PutBytes 将header写入p
func (h *BlockHeader) PutBytes(p []byte) {
	byteOrder.PutUint16(p, h.Magic)
	p[2] = h.Version
	p[3] = h.Reserved
	byteOrder.PutUint32(p[4:], h.TotalLines)
	byteOrder.PutUint64(p[8:], h.Timestamp)
}

// ReadFrom 从r读取header
func (h *BlockHeader) ReadFrom(r io.Reader) (int64, error) {
	var buf [blockHeaderSize]byte
	n, err := io.ReadFull(r, buf[:])
	if err != nil {
		return int64(n), err
	}
	h.Magic = byteOrder.Uint16(buf[:])
	if h.Magic != blockHeaderMagic {
		return 0, fmt.Errorf("invalid block magic: 0x%02x", h.Magic)
	}
	h.Version = buf[2]
	h.Reserved = buf[3]
	h.TotalLines = byteOrder.Uint32(buf[4:])
	h.Timestamp = byteOrder.Uint64(buf[8:])
	return int64(blockHeaderSize), nil
}

// EncodeDataBlock 将一个文件的所有行覆盖数据编码为[]byte
func EncodeDataBlock(header *BlockHeader, lines []uint32) []byte {
	size := blockHeaderSize + len(lines)*int32Size
	buf := make([]byte, size)
	header.PutBytes(buf)
	offset := blockHeaderSize
	for _, n := range lines {
		byteOrder.PutUint32(buf[offset:], n)
		offset += int32Size
	}
	return buf
}

// ParseDataBlock 从data中解析出一个文件的所有行覆盖数据
func ParseDataBlock(data []byte, totalLines int) []uint32 {
	offset := 0
	res := make([]uint32, totalLines)
	for i := 0; i < totalLines; i++ {
		res[i] = byteOrder.Uint32(data[offset:])
		offset += int32Size
	}
	return res
}

// 根据行覆盖率信息和diff信息，编码是否是增量覆盖率、是否是指令行等信息
type CoverLineEncoder interface {
	Encode(rawCoverLines []int32, Changedlines []uint32) ([]byte, error)
}

type defaultCoverLineEncoder struct{}

func NewCoverLineEncoder() CoverLineEncoder {
	return &defaultCoverLineEncoder{}
}

func (e *defaultCoverLineEncoder) Encode(rawCoverLines []int32, addedlines []uint32) ([]byte, error) {
	// 校验合法性
	for _, val := range rawCoverLines {
		// -1 是最小值，标识是非指令行，如果是其他负数，报错
		if val < -1 {
			return nil, fmt.Errorf("invalid cover line value (negative): %d", val)
		}
		// 转换成uint32时，前两位被借位做flag标识，所以不能超过 MaxCoverCount
		if val > MaxCoverCount {
			return nil, fmt.Errorf("cover line value %d exceeds maximum allowed %d", val, MaxCoverCount)
		}
	}

	lines := make([]uint32, len(rawCoverLines))

	for i, val := range rawCoverLines {
		var uVal uint32
		if val == -1 {
			// 非指令行
			uVal = MaskInstrLine
			// 注意：这里为了兼容 test.cda 中的历史格式，非指令行时 MaskCoverCount 区域可能包含 0xFFFFFF
			// 但我们当前的 DecodeRawCoverLine 逻辑是 val&MaskInstrLine != 0 即为 -1，不依赖低位值。
			// 如果需要 100% 二进制一致，可能需要知道当初低位填的是什么。
			// 这里的 orig=80ffffff 表明低 24 位被填为了 0xffffff
			uVal |= 0x00ffffff
		} else {
			// 指令行，设置覆盖次数
			uVal = uint32(val) & MaskCoverCount
		}

		// 设置增量标识，因为diff行号从1开始，需要给lineNum + 1
		if len(addedlines) > 0 && slices.Contains(addedlines, uint32(i+1)) {
			uVal |= MaskIncrLine
		}

		lines[i] = uVal
	}

	header := NewBlockHeader(uint32(len(lines)))
	return EncodeDataBlock(header, lines), nil
}
