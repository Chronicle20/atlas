package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const FootholdInfoWriter = "FootholdInfo"

// FootholdEntry is one dynamic foothold-object record inside a FOOTHOLD_INFO
// packet.
//
//   - name    : the object name (DecodeStr).
//   - mode    : the foothold state-change mode (Decode4). When mode == 2 the
//     packet carries an additional moving-object block (move ints + 2 flag bytes).
//   - ids     : v95/jms only — the count-prefixed list of foothold ids
//     (Decode4 idCount, idCount × Decode4). Empty/ignored on v87.
//   - moveInts: the mode==2 moving-object int32 block. v87 carries 8 ints,
//     v95/jms carry 7 ints (the v95/jms count-loop wrapper moved one int into the
//     id-list path). nil/empty when mode != 2.
//   - reverseVertical / reverseHorizontal: the two trailing mode==2 flag bytes.
type FootholdEntry struct {
	name              string
	mode              uint32
	ids               []uint32
	moveInts          []uint32
	reverseVertical   byte
	reverseHorizontal byte
}

func NewFootholdEntry(name string, mode uint32, ids []uint32, moveInts []uint32, reverseVertical byte, reverseHorizontal byte) FootholdEntry {
	return FootholdEntry{
		name:              name,
		mode:              mode,
		ids:               ids,
		moveInts:          moveInts,
		reverseVertical:   reverseVertical,
		reverseHorizontal: reverseHorizontal,
	}
}

func (e FootholdEntry) Name() string              { return e.name }
func (e FootholdEntry) Mode() uint32              { return e.mode }
func (e FootholdEntry) Ids() []uint32             { return e.ids }
func (e FootholdEntry) MoveInts() []uint32        { return e.moveInts }
func (e FootholdEntry) ReverseVertical() byte     { return e.reverseVertical }
func (e FootholdEntry) ReverseHorizontal() byte   { return e.reverseHorizontal }

// FootholdInfo models the FOOTHOLD_INFO clientbound packet (CField::OnFootHoldInfo).
//
// The wire layout is version-divergent (v95 authoritative):
//
//   - v87 @0x560fec — a single entry, no count prefix and no id-list:
//     DecodeStr(name), Decode4(mode), then if mode == 2: 8 × Decode4 + 2 × Decode1.
//   - v95 @0x53a810 / jms @0x576a89 — Decode4(count), then count entries, each:
//     DecodeStr(name), Decode4(mode), Decode4(idCount), idCount × Decode4(id),
//     then if mode == 2: 7 × Decode4 + 2 × Decode1.
//   - GMS < 87: the CField::OnFootHoldInfo handler does not exist (VERSION-ABSENT).
//
// The CMapLoadable::FootHoldStateChange / FootHoldMove delegates are application
// logic (they apply the state to the loaded map), not wire reads, and carry no
// bytes.
//
// packet-audit:fname CField::OnFootHoldInfo
type FootholdInfo struct {
	entries []FootholdEntry
}

func NewFootholdInfo(entries []FootholdEntry) FootholdInfo {
	return FootholdInfo{entries: entries}
}

func (m FootholdInfo) Entries() []FootholdEntry { return m.entries }

func (m FootholdInfo) Operation() string { return FootholdInfoWriter }
func (m FootholdInfo) String() string {
	return fmt.Sprintf("foothold info entries [%d]", len(m.entries))
}

// usesCountForm reports whether the tenant uses the v95/jms count-prefixed
// id-list layout (true) or the v87 single-entry layout (false).
func usesCountForm(t tenant.Model) bool {
	return t.MajorAtLeast(95)
}

func (m FootholdInfo) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		if usesCountForm(t) {
			w.WriteInt(uint32(len(m.entries)))
			for _, e := range m.entries {
				w.WriteAsciiString(e.name)
				w.WriteInt(e.mode)
				w.WriteInt(uint32(len(e.ids)))
				for _, id := range e.ids {
					w.WriteInt(id)
				}
				if e.mode == 2 {
					encodeFootholdMove(w, e)
				}
			}
			return w.Bytes()
		}
		// v87 single-entry form: no count, no id-list.
		var e FootholdEntry
		if len(m.entries) > 0 {
			e = m.entries[0]
		}
		w.WriteAsciiString(e.name)
		w.WriteInt(e.mode)
		if e.mode == 2 {
			encodeFootholdMove(w, e)
		}
		return w.Bytes()
	}
}

func encodeFootholdMove(w *response.Writer, e FootholdEntry) {
	for _, v := range e.moveInts {
		w.WriteInt(v)
	}
	w.WriteByte(e.reverseVertical)
	w.WriteByte(e.reverseHorizontal)
}

func (m *FootholdInfo) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	moveIntCount := 8
	if usesCountForm(t) {
		moveIntCount = 7
	}
	return func(r *request.Reader, options map[string]interface{}) {
		if usesCountForm(t) {
			count := r.ReadUint32()
			m.entries = make([]FootholdEntry, 0, count)
			for i := uint32(0); i < count; i++ {
				var e FootholdEntry
				e.name = r.ReadAsciiString()
				e.mode = r.ReadUint32()
				idCount := r.ReadUint32()
				e.ids = make([]uint32, 0, idCount)
				for j := uint32(0); j < idCount; j++ {
					e.ids = append(e.ids, r.ReadUint32())
				}
				if e.mode == 2 {
					decodeFootholdMove(r, &e, moveIntCount)
				}
				m.entries = append(m.entries, e)
			}
			return
		}
		var e FootholdEntry
		e.name = r.ReadAsciiString()
		e.mode = r.ReadUint32()
		if e.mode == 2 {
			decodeFootholdMove(r, &e, moveIntCount)
		}
		m.entries = []FootholdEntry{e}
	}
}

func decodeFootholdMove(r *request.Reader, e *FootholdEntry, intCount int) {
	e.moveInts = make([]uint32, 0, intCount)
	for i := 0; i < intCount; i++ {
		e.moveInts = append(e.moveInts, r.ReadUint32())
	}
	e.reverseVertical = r.ReadByte()
	e.reverseHorizontal = r.ReadByte()
}
