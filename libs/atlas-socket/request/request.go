package request

import (
	"fmt"
	"golang.org/x/text/transform"
	"io"
	"strings"
)
import "golang.org/x/text/encoding/japanese"

// Request -
type Request []byte

type Opcode byte

// Size -
func (p *Request) Size() int {
	return len(*p)
}

// String -
func (p *Request) String() string {
	return fmt.Sprintf("[Request] (%d) : % X", len(*p), string(*p))
}

func (p *Request) readByte(pos *int) byte {
	r := (*p)[*pos]
	*pos++
	return r
}

func (p *Request) readInt8(pos *int) int8 {
	r := int8((*p)[*pos])
	*pos++
	return r
}

func (p *Request) readBool(pos *int) bool {
	r := (*p)[*pos]
	*pos++

	if r == 0 {
		return false
	}

	return true
}

func (p *Request) readBytes(pos *int, length int) []byte {
	r := []byte((*p)[*pos : *pos+length])
	*pos += length
	return r
}

func (p *Request) readInt16(pos *int) int16 {
	return int16(p.readByte(pos)) | (int16(p.readByte(pos)) << 8)
}

func (p *Request) readInt32(pos *int) int32 {
	return int32(p.readByte(pos)) |
		int32(p.readByte(pos))<<8 |
		int32(p.readByte(pos))<<16 |
		int32(p.readByte(pos))<<24
}

func (p *Request) readInt64(pos *int) int64 {
	return int64(p.readByte(pos)) |
		int64(p.readByte(pos))<<8 |
		int64(p.readByte(pos))<<16 |
		int64(p.readByte(pos))<<24 |
		int64(p.readByte(pos))<<32 |
		int64(p.readByte(pos))<<40 |
		int64(p.readByte(pos))<<48 |
		int64(p.readByte(pos))<<56
}

func (p *Request) readUint16(pos *int) uint16 {
	return uint16(p.readByte(pos)) | (uint16(p.readByte(pos)) << 8)
}

func (p *Request) readUint32(pos *int) uint32 {
	return uint32(p.readByte(pos)) |
		uint32(p.readByte(pos))<<8 |
		uint32(p.readByte(pos))<<16 |
		uint32(p.readByte(pos))<<24
}

func (p *Request) readUint64(pos *int) uint64 {
	return uint64(p.readByte(pos)) |
		uint64(p.readByte(pos))<<8 |
		uint64(p.readByte(pos))<<16 |
		uint64(p.readByte(pos))<<24 |
		uint64(p.readByte(pos))<<32 |
		uint64(p.readByte(pos))<<40 |
		uint64(p.readByte(pos))<<48 |
		uint64(p.readByte(pos))<<56
}

func (p *Request) readString(pos *int, length int) string {
	bytes := p.readBytes(pos, length)
	d := japanese.ShiftJIS.NewDecoder()
	r := transform.NewReader(strings.NewReader(string(bytes)), d)
	dbs, err := io.ReadAll(r)
	if err != nil {
		return string(bytes)
	}
	return string(dbs)
}
