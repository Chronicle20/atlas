package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const AdminResultWriter = "AdminResult"

// AdminResult models the ADMIN_RESULT clientbound packet (CField::OnAdminResult).
//
// The client dispatches on a leading mode byte (Decode1) into a large switch; each
// mode reads a different field set (v95 @0x53bc20):
//
//	mode 4 / 5         : Decode1 (extra byte)
//	mode 6             : Decode1 (flag)
//	mode 0xB           : DecodeStr(channel) [+ DecodeStr(world) + DecodeStr(msg)]
//	mode 0x12          : Decode1
//	mode 0x15          : Decode1(flag) [+ Decode1(channel) | + Decode4(mapId)]
//	mode 0x28 / 0x29   : (no wire fields)
//	mode 0x2A / 0x2B   : Decode1
//	mode 0x33-0x39     : DecodeStr
//	mode 0x3A          : DecodeStr
//	mode 0x47 / 0x48   : DecodeStr
//
// As with SPOUSE_CHAT, the IDA export flattens the guarded switch arms into a
// single positional read order (the analyzer cannot parse switch guards). The
// flattened order DIFFERS PER VERSION (each binary's harvested flat read), so this
// codec emits each version's exact flat field sequence under a mutually-exclusive
// version guard. The first field is always the mode discriminator. A concrete send
// populates the arm for its mode; the representative fixture exercises the full
// flattened union so the round-trip closes for the modeled shape.
//
// Non-wire Delegate calls (ZXString operator+ string concat used to build the GM
// chat log line — v84 sub_476592, jms sub_4A586D) are application logic, stripped
// from those exports per the §10 report-gen discipline.
//
// Per-version post-mode flat schemas (b = Decode1, s = DecodeStr, i = Decode4):
//
//	v83 @0x5352e9: s,b,b,b,i,b,b,s,s,s,b,b,b
//	v84 @0x54156f: b,s,s,b,b,b,i,b,s,s,s,b,b,b
//	v87 @0x55cac3: b,b,b,s,s,s,b,b,b,i,b,b,s,s
//	v95 @0x53bc20: b,b,b,s,s,s,b,b,b,i,b,b,s,s,s,s
//	jms @0x57255f: b,s,s,s,b,b,b,b,b,i
//
// The model holds the union of byte/string/int payloads (indexed positionally).
// packet-audit:fname CField::OnAdminResult
type AdminResult struct {
	mode  byte
	b     []byte
	s     []string
	mapId uint32
}

// NewAdminResult builds an AdminResult carrying the union of representative fields.
// b supplies the Decode1 payload values (in flat-order), s the DecodeStr values
// (in flat-order), and mapId the single Decode4 value. Out-of-range indices encode
// as zero/empty, so a caller may supply only the fields for its mode.
func NewAdminResult(mode byte, b []byte, s []string, mapId uint32) AdminResult {
	return AdminResult{mode: mode, b: b, s: s, mapId: mapId}
}

func (m AdminResult) Mode() byte     { return m.mode }
func (m AdminResult) Bytes() []byte  { return m.b }
func (m AdminResult) Strs() []string { return m.s }
func (m AdminResult) MapId() uint32  { return m.mapId }

func (m AdminResult) Operation() string { return AdminResultWriter }
func (m AdminResult) String() string {
	return fmt.Sprintf("admin result mode [%d]", m.mode)
}

func (m AdminResult) bAt(i int) byte {
	if i >= 0 && i < len(m.b) {
		return m.b[i]
	}
	return 0
}

func (m AdminResult) sAt(i int) string {
	if i >= 0 && i < len(m.s) {
		return m.s[i]
	}
	return ""
}

func (m AdminResult) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	t := tenant.MustFromContext(ctx)
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // Decode1: mode discriminator (all versions)
		if t.Region() == "JMS" {
			// jms: b,s,s,s,b,b,b,b,b,i
			w.WriteByte(m.bAt(0))
			w.WriteAsciiString(m.sAt(0))
			w.WriteAsciiString(m.sAt(1))
			w.WriteAsciiString(m.sAt(2))
			w.WriteByte(m.bAt(1))
			w.WriteByte(m.bAt(2))
			w.WriteByte(m.bAt(3))
			w.WriteByte(m.bAt(4))
			w.WriteByte(m.bAt(5))
			w.WriteInt(m.mapId)
		}
		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			// v95: b,b,b,s,s,s,b,b,b,i,b,b,s,s,s,s
			w.WriteByte(m.bAt(0))
			w.WriteByte(m.bAt(1))
			w.WriteByte(m.bAt(2))
			w.WriteAsciiString(m.sAt(0))
			w.WriteAsciiString(m.sAt(1))
			w.WriteAsciiString(m.sAt(2))
			w.WriteByte(m.bAt(3))
			w.WriteByte(m.bAt(4))
			w.WriteByte(m.bAt(5))
			w.WriteInt(m.mapId)
			w.WriteByte(m.bAt(6))
			w.WriteByte(m.bAt(7))
			w.WriteAsciiString(m.sAt(3))
			w.WriteAsciiString(m.sAt(4))
			w.WriteAsciiString(m.sAt(5))
			w.WriteAsciiString(m.sAt(6))
		}
		if t.Region() == "GMS" && t.MajorVersion() >= 87 && t.MajorVersion() < 95 {
			// v87: b,b,b,s,s,s,b,b,b,i,b,b,s,s
			w.WriteByte(m.bAt(0))
			w.WriteByte(m.bAt(1))
			w.WriteByte(m.bAt(2))
			w.WriteAsciiString(m.sAt(0))
			w.WriteAsciiString(m.sAt(1))
			w.WriteAsciiString(m.sAt(2))
			w.WriteByte(m.bAt(3))
			w.WriteByte(m.bAt(4))
			w.WriteByte(m.bAt(5))
			w.WriteInt(m.mapId)
			w.WriteByte(m.bAt(6))
			w.WriteByte(m.bAt(7))
			w.WriteAsciiString(m.sAt(3))
			w.WriteAsciiString(m.sAt(4))
		}
		if t.Region() == "GMS" && t.MajorVersion() >= 84 && t.MajorVersion() < 87 {
			// v84: b,s,s,b,b,b,i,b,s,s,s,b,b,b
			w.WriteByte(m.bAt(0))
			w.WriteAsciiString(m.sAt(0))
			w.WriteAsciiString(m.sAt(1))
			w.WriteByte(m.bAt(1))
			w.WriteByte(m.bAt(2))
			w.WriteByte(m.bAt(3))
			w.WriteInt(m.mapId)
			w.WriteByte(m.bAt(4))
			w.WriteAsciiString(m.sAt(2))
			w.WriteAsciiString(m.sAt(3))
			w.WriteAsciiString(m.sAt(4))
			w.WriteByte(m.bAt(5))
			w.WriteByte(m.bAt(6))
			w.WriteByte(m.bAt(7))
		}
		if t.Region() == "GMS" && t.MajorVersion() >= 83 && t.MajorVersion() < 84 {
			// v83: s,b,b,b,i,b,b,s,s,s,b,b,b
			w.WriteAsciiString(m.sAt(0))
			w.WriteByte(m.bAt(0))
			w.WriteByte(m.bAt(1))
			w.WriteByte(m.bAt(2))
			w.WriteInt(m.mapId)
			w.WriteByte(m.bAt(3))
			w.WriteByte(m.bAt(4))
			w.WriteAsciiString(m.sAt(1))
			w.WriteAsciiString(m.sAt(2))
			w.WriteAsciiString(m.sAt(3))
			w.WriteByte(m.bAt(5))
			w.WriteByte(m.bAt(6))
			w.WriteByte(m.bAt(7))
		}
		if t.Region() == "GMS" && t.MajorVersion() < 83 {
			// v79 @0x52075c: b,b,b,i,b,b,s,s,s,b,b,b (export-harvested flat order).
			w.WriteByte(m.bAt(0))        // Decode1 @0x520d9b (mode 29)
			w.WriteByte(m.bAt(1))        // Decode1 @0x520b5f (mode 19)
			w.WriteByte(m.bAt(2))        // Decode1 @0x520b71 (mode 19)
			w.WriteInt(m.mapId)          // Decode4 @0x520cd9 (mode 19 mapId)
			w.WriteByte(m.bAt(3))        // Decode1 @0x520bb5 (mode 19 LABEL_45)
			w.WriteByte(m.bAt(4))        // Decode1 @0x5207c4 (mode 16)
			w.WriteAsciiString(m.sAt(0)) // DecodeStr @0x5207d5 (mode 11)
			w.WriteAsciiString(m.sAt(1)) // DecodeStr @0x520827 (mode 11)
			w.WriteAsciiString(m.sAt(2)) // DecodeStr @0x520836 (mode 11)
			w.WriteByte(m.bAt(5))        // Decode1 @0x52096d (mode 6)
			w.WriteByte(m.bAt(6))        // Decode1 @0x520a39 (mode 5)
			w.WriteByte(m.bAt(7))        // Decode1 @0x520ad0 (mode 4)
		}
		return w.Bytes()
	}
}

func (m *AdminResult) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.b = nil
		m.s = nil
		if t.Region() == "JMS" {
			m.b = append(m.b, r.ReadByte())
			m.s = append(m.s, r.ReadAsciiString())
			m.s = append(m.s, r.ReadAsciiString())
			m.s = append(m.s, r.ReadAsciiString())
			m.b = append(m.b, r.ReadByte())
			m.b = append(m.b, r.ReadByte())
			m.b = append(m.b, r.ReadByte())
			m.b = append(m.b, r.ReadByte())
			m.b = append(m.b, r.ReadByte())
			m.mapId = r.ReadUint32()
		}
		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			m.b = append(m.b, r.ReadByte())
			m.b = append(m.b, r.ReadByte())
			m.b = append(m.b, r.ReadByte())
			m.s = append(m.s, r.ReadAsciiString())
			m.s = append(m.s, r.ReadAsciiString())
			m.s = append(m.s, r.ReadAsciiString())
			m.b = append(m.b, r.ReadByte())
			m.b = append(m.b, r.ReadByte())
			m.b = append(m.b, r.ReadByte())
			m.mapId = r.ReadUint32()
			m.b = append(m.b, r.ReadByte())
			m.b = append(m.b, r.ReadByte())
			m.s = append(m.s, r.ReadAsciiString())
			m.s = append(m.s, r.ReadAsciiString())
			m.s = append(m.s, r.ReadAsciiString())
			m.s = append(m.s, r.ReadAsciiString())
		}
		if t.Region() == "GMS" && t.MajorVersion() >= 87 && t.MajorVersion() < 95 {
			m.b = append(m.b, r.ReadByte())
			m.b = append(m.b, r.ReadByte())
			m.b = append(m.b, r.ReadByte())
			m.s = append(m.s, r.ReadAsciiString())
			m.s = append(m.s, r.ReadAsciiString())
			m.s = append(m.s, r.ReadAsciiString())
			m.b = append(m.b, r.ReadByte())
			m.b = append(m.b, r.ReadByte())
			m.b = append(m.b, r.ReadByte())
			m.mapId = r.ReadUint32()
			m.b = append(m.b, r.ReadByte())
			m.b = append(m.b, r.ReadByte())
			m.s = append(m.s, r.ReadAsciiString())
			m.s = append(m.s, r.ReadAsciiString())
		}
		if t.Region() == "GMS" && t.MajorVersion() >= 84 && t.MajorVersion() < 87 {
			m.b = append(m.b, r.ReadByte())
			m.s = append(m.s, r.ReadAsciiString())
			m.s = append(m.s, r.ReadAsciiString())
			m.b = append(m.b, r.ReadByte())
			m.b = append(m.b, r.ReadByte())
			m.b = append(m.b, r.ReadByte())
			m.mapId = r.ReadUint32()
			m.b = append(m.b, r.ReadByte())
			m.s = append(m.s, r.ReadAsciiString())
			m.s = append(m.s, r.ReadAsciiString())
			m.s = append(m.s, r.ReadAsciiString())
			m.b = append(m.b, r.ReadByte())
			m.b = append(m.b, r.ReadByte())
			m.b = append(m.b, r.ReadByte())
		}
		if t.Region() == "GMS" && t.MajorVersion() >= 83 && t.MajorVersion() < 84 {
			m.s = append(m.s, r.ReadAsciiString())
			m.b = append(m.b, r.ReadByte())
			m.b = append(m.b, r.ReadByte())
			m.b = append(m.b, r.ReadByte())
			m.mapId = r.ReadUint32()
			m.b = append(m.b, r.ReadByte())
			m.b = append(m.b, r.ReadByte())
			m.s = append(m.s, r.ReadAsciiString())
			m.s = append(m.s, r.ReadAsciiString())
			m.s = append(m.s, r.ReadAsciiString())
			m.b = append(m.b, r.ReadByte())
			m.b = append(m.b, r.ReadByte())
			m.b = append(m.b, r.ReadByte())
		}
		if t.Region() == "GMS" && t.MajorVersion() < 83 {
			// v79: b,b,b,i,b,b,s,s,s,b,b,b
			m.b = append(m.b, r.ReadByte())
			m.b = append(m.b, r.ReadByte())
			m.b = append(m.b, r.ReadByte())
			m.mapId = r.ReadUint32()
			m.b = append(m.b, r.ReadByte())
			m.b = append(m.b, r.ReadByte())
			m.s = append(m.s, r.ReadAsciiString())
			m.s = append(m.s, r.ReadAsciiString())
			m.s = append(m.s, r.ReadAsciiString())
			m.b = append(m.b, r.ReadByte())
			m.b = append(m.b, r.ReadByte())
			m.b = append(m.b, r.ReadByte())
		}
	}
}
