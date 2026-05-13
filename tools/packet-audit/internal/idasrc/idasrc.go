package idasrc

import "context"

type Direction int

const (
	DirClientbound Direction = iota
	DirServerbound
)

type Primitive int

const (
	Decode1 Primitive = iota // ReadByte/WriteByte
	Decode2                  // ReadShort/WriteShort
	Decode4                  // ReadInt/WriteInt
	Decode8                  // ReadLong/WriteLong
	DecodeStr                // ReadAsciiString/WriteAsciiString
	DecodeBuf                // ReadBytes/WriteBytes
)

func (p Primitive) String() string {
	switch p {
	case Decode1:
		return "byte"
	case Decode2:
		return "int16"
	case Decode4:
		return "int32"
	case Decode8:
		return "int64"
	case DecodeStr:
		return "string"
	case DecodeBuf:
		return "bytes"
	}
	return "unknown"
}

type FieldCall struct {
	Op      Primitive
	Comment string
	Guard   string // free-form expression text; "" if unconditional
}

type Fields struct {
	Function  string
	Address   string
	Direction Direction
	Calls     []FieldCall
}

type Source interface {
	Resolve(ctx context.Context, fname string) (Fields, error)
}
