package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const StalkResultWriter = "StalkResult"

// StalkResult models the IDA_0X09C / OnStalkResult clientbound packet
// (CField::OnStalkResult — the minimap stalkee-list update).
//
// The real client wire is a count-prefixed loop (v83 @0x537a6a, v95 @0x539910 are
// byte-identical):
//
//	Decode4(count)
//	repeat count times:
//	  Decode4(charId)
//	  Decode1(flag)   // 1 = RemoveStalkee, 0 = InsertStalkee
//	  if flag == 0:   // insert branch
//	    DecodeStr(name) + Decode4(x) + Decode4(y)   // tagPOINT
//
// The InsertStalkee / RemoveStalkee / ZXString::_Release calls are UI application
// logic (the Delegate entries in the export), not wire reads.
//
// The export flattens one insert-branch iteration into a single positional read
// order; this model carries that representative shape (count + one stalkee's
// insert fields) so the wire-level diff aligns positionally with the flattened
// read order and the round-trip closes. Layout is version-invariant.
// packet-audit:fname CField::OnStalkResult
type StalkResult struct {
	count  uint32
	charId uint32
	flag   byte
	name   string
	x      uint32
	y      uint32
}

// NewStalkResult constructs the representative of the OnStalkResult read order:
// count + one stalkee entry (charId + flag + name + x + y, the flag==0 insert arm).
func NewStalkResult(count uint32, charId uint32, flag byte, name string, x uint32, y uint32) StalkResult {
	return StalkResult{count: count, charId: charId, flag: flag, name: name, x: x, y: y}
}

func (m StalkResult) Count() uint32  { return m.count }
func (m StalkResult) CharId() uint32 { return m.charId }
func (m StalkResult) Flag() byte     { return m.flag }
func (m StalkResult) Name() string   { return m.name }
func (m StalkResult) X() uint32      { return m.x }
func (m StalkResult) Y() uint32      { return m.y }

func (m StalkResult) Operation() string { return StalkResultWriter }
func (m StalkResult) String() string {
	return fmt.Sprintf("stalk result count [%d] charId [%d] flag [%d]", m.count, m.charId, m.flag)
}

func (m StalkResult) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.count)        // Decode4: stalkee count
		w.WriteInt(m.charId)       // Decode4: stalkee charId
		w.WriteByte(m.flag)        // Decode1: 1 = remove, 0 = insert
		w.WriteAsciiString(m.name) // DecodeStr: name (insert branch)
		w.WriteInt(m.x)            // Decode4: x (insert branch, tagPOINT)
		w.WriteInt(m.y)            // Decode4: y (insert branch, tagPOINT)
		return w.Bytes()
	}
}

func (m *StalkResult) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.count = r.ReadUint32()
		m.charId = r.ReadUint32()
		m.flag = r.ReadByte()
		m.name = r.ReadAsciiString()
		m.x = r.ReadUint32()
		m.y = r.ReadUint32()
	}
}
