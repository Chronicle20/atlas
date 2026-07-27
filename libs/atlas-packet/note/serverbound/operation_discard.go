package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// noteDiscardSpecialFlag reports the memo flag value that CMemoListDlg::SetRet
// treats as a "gift/reward" (special) entry — one that claims a free ETC slot
// and carries trailing reward field(s) on the wire. IDA-verified per version
// (SetRet: v48 0x534dc4, v61 0x5ad50c, v72 0x5fb443, v79 0x619f32, v83 0x64aa57,
// v84 0x6606a0, v87 0x684843, v95 0x624280, jms 0x6c2d43): GMS <= v61 compare
// against 2, all later GMS and JMS compare against 3.
func noteDiscardSpecialFlag(t tenant.Model) byte {
	if t.Region() == "GMS" && t.MajorVersion() <= 61 {
		return 2
	}
	return 3
}

// noteDiscardExtraCount reports how many trailing int32 fields a special
// (flag == noteDiscardSpecialFlag) discard entry carries. Every GMS SetRet
// writes exactly one (reward/itemId/mesos/value); jms_v185 (0x6c2d43) writes
// two. IDA-verified.
func noteDiscardExtraCount(t tenant.Model) int {
	if t.Region() == "JMS" {
		return 2
	}
	return 1
}

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

// Extra1 is the first (and, on GMS, only) trailing int32 a "special" discard
// entry carries once granted a free ETC slot — the claimed reward/itemId/mesos
// (CMemoListDlg::SetRet, e.g. v83 0x64aa57 Encode4(v28)). Zero on normal entries.
func (e DiscardEntry) Extra1() uint32 {
	return e.extra1
}

// Extra2 is jms_v185's second trailing int32 on a special discard entry
// (CMemoListDlg::SetRet@0x6c2d43: Encode4(v25, a2.p) at 0x6c2faf). Zero on every
// GMS version (which write a single extra field) and on normal entries.
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

// SpecialCount is the third header byte CMemoListDlg::SetRet writes on EVERY
// version (IDA-verified v48-v95 + jms): the number of memos in the client's
// local list whose flag equals noteDiscardSpecialFlag, counted before the
// free-slot budget is applied. (An earlier revision wrongly modeled this as a
// jms-only field, which mis-aligned the GMS discard body by one byte.)
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

// wireEntryCount is the number of memo entries actually present on the wire.
// CMemoListDlg::SetRet OMITS a special (flag==noteDiscardSpecialFlag) entry
// entirely once the free-slot budget (emptySlotCount) is exhausted — the loop
// sets a "not enough slot" flag and writes no bytes for it. So the wire carries
// totalCount minus the special entries that overflowed the budget. IDA-verified
// identical on every GMS version and jms.
func (m OperationDiscard) wireEntryCount() byte {
	if m.specialCount > m.emptySlotCount {
		return m.count - (m.specialCount - m.emptySlotCount)
	}
	return m.count
}

func (m OperationDiscard) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	t := tenant.MustFromContext(ctx)
	special := noteDiscardSpecialFlag(t)
	extraCount := noteDiscardExtraCount(t)
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.count)
		w.WriteByte(m.specialCount)
		w.WriteByte(m.emptySlotCount)
		for _, e := range m.entries {
			w.WriteInt(e.id)
			w.WriteByte(e.flag)
			if e.flag == special {
				w.WriteInt(e.extra1)
				if extraCount == 2 {
					w.WriteInt(e.extra2)
				}
			}
		}
		return w.Bytes()
	}
}

func (m *OperationDiscard) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	special := noteDiscardSpecialFlag(t)
	extraCount := noteDiscardExtraCount(t)
	return func(r *request.Reader, options map[string]interface{}) {
		m.count = r.ReadByte()
		m.specialCount = r.ReadByte()
		m.emptySlotCount = r.ReadByte()

		wireEntries := m.wireEntryCount()
		m.entries = make([]DiscardEntry, 0, wireEntries)
		for i := byte(0); i < wireEntries; i++ {
			e := DiscardEntry{
				id:   r.ReadUint32(),
				flag: r.ReadByte(),
			}
			if e.flag == special {
				e.extra1 = r.ReadUint32()
				if extraCount == 2 {
					e.extra2 = r.ReadUint32()
				}
			}
			m.entries = append(m.entries, e)
		}
	}
}
