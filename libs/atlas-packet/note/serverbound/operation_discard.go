package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// noteDiscardSpecialFlagJMS is the memo flag value (`*(entry+24)`) that marks a
// "gift/reward" memo on jms_v185's CMemoListDlg::SetRet@0x6c2d43. Only entries
// carrying this flag can have the two trailing extra int32 fields.
const noteDiscardSpecialFlagJMS = 3

type DiscardEntry struct {
	id     uint32
	flag   byte
	extra1 uint32
	extra2 uint32
}

func (e DiscardEntry) Id() uint32 {
	return e.id
}

func (e DiscardEntry) Flag() byte {
	return e.flag
}

// Extra1 is the first of two trailing int32 fields jms_v185 appends to a
// "special" (flag==3) discard entry that was granted a free ETC slot
// (CMemoListDlg::SetRet@0x6c2d43: `COutPacket::Encode4(v25, v28)`). Zero on
// every other version/entry shape.
func (e DiscardEntry) Extra1() uint32 {
	return e.extra1
}

// Extra2 is the second of jms_v185's two trailing int32 fields
// (`COutPacket::Encode4(v25, a2.p)` at 0x6c2faf). Zero on every other
// version/entry shape.
func (e DiscardEntry) Extra2() uint32 {
	return e.extra2
}

// packet-audit:fname CMemoListDlg::SetRet
type OperationDiscard struct {
	count          byte
	specialCount   byte
	emptySlotCount byte
	entries        []DiscardEntry
}

func (m OperationDiscard) Count() byte {
	return m.count
}

// SpecialCount is jms_v185's third header byte — the number of memos in the
// client's local list whose flag equals noteDiscardSpecialFlagJMS, counted
// BEFORE the free-slot budget is applied (CMemoListDlg::SetRet@0x6c2d43,
// 0x6c2e0c-0x6c2e1d). Zero (and not written on the wire) on every other
// verified version, whose SetRet shape has no such field.
func (m OperationDiscard) SpecialCount() byte {
	return m.specialCount
}

func (m OperationDiscard) EmptySlotCount() byte {
	return m.emptySlotCount
}

func (m OperationDiscard) Entries() []DiscardEntry {
	return m.entries
}

func (m OperationDiscard) Operation() string {
	return "OperationDiscard"
}

func (m OperationDiscard) String() string {
	return fmt.Sprintf("count [%d] specialCount [%d] emptySlotCount [%d] entries [%d]", m.count, m.specialCount, m.emptySlotCount, len(m.entries))
}

func (m OperationDiscard) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	t := tenant.MustFromContext(ctx)
	// jms_v185 CMemoListDlg::SetRet@0x6c2d43 writes an extra header byte
	// (specialCount, 0x6c2e1d) between totalCount and emptySlots, and a
	// "special" (flag==3) entry that was granted a free slot writes two
	// trailing int32 fields (0x6c2f81-0x6c2faf) that no other verified
	// version's SetRet shape has. gms_v83/v84/v87/v95 keep the plain
	// count+emptySlotCount+entry(id,flag) shape verified before this change.
	isJMS := t.Region() == "JMS"
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.count)
		if isJMS {
			w.WriteByte(m.specialCount)
		}
		w.WriteByte(m.emptySlotCount)
		for _, e := range m.entries {
			w.WriteInt(e.id)
			w.WriteByte(e.flag)
			if isJMS && e.flag == noteDiscardSpecialFlagJMS {
				w.WriteInt(e.extra1)
				w.WriteInt(e.extra2)
			}
		}
		return w.Bytes()
	}
}

func (m *OperationDiscard) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	isJMS := t.Region() == "JMS"
	return func(r *request.Reader, options map[string]interface{}) {
		m.count = r.ReadByte()
		if isJMS {
			m.specialCount = r.ReadByte()
		}
		m.emptySlotCount = r.ReadByte()

		// jms_v185 only: the client OMITS a special (flag==3) entry from the
		// wire entirely once its free-slot budget (emptySlotCount) is
		// exhausted (0x6c2e88: `if (*n <= 0) { v32 = 1; }` — no Encode calls
		// on that branch). Every special entry that DOES reach the wire was
		// therefore written while budget remained, so it always carries the
		// two trailing extra fields — no running-budget bookkeeping is
		// needed on decode, only the resulting wire-entry count:
		//   wireEntries = totalCount - max(0, specialCount - emptySlotCount)
		wireEntries := m.count
		if isJMS {
			skipped := byte(0)
			if m.specialCount > m.emptySlotCount {
				skipped = m.specialCount - m.emptySlotCount
			}
			wireEntries = m.count - skipped
		}

		m.entries = make([]DiscardEntry, 0, wireEntries)
		for i := byte(0); i < wireEntries; i++ {
			e := DiscardEntry{
				id:   r.ReadUint32(),
				flag: r.ReadByte(),
			}
			if isJMS && e.flag == noteDiscardSpecialFlagJMS {
				e.extra1 = r.ReadUint32()
				e.extra2 = r.ReadUint32()
			}
			m.entries = append(m.entries, e)
		}
	}
}
