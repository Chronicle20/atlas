package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// discardSpecialFlag reports the memo flag CMemoListDlg::SetRet treats as a
// wedding-invitation ("special") entry — the only entry kind that claims a free
// ETC slot and carries a trailing marriage number. IDA-verified across every
// version's SetRet: the earliest GMS builds (v48 0x534dc4, v61 0x5ad50c) use 2,
// v72 onward (and jms) use 3. The 2->3 shift is an enum renumber, not a
// behavior change — both binaries ship the same wedding feature (CField_Wedding,
// OnMarriageResult, GW_MarriageRecord). This is a client-version fact resolved
// in code (a version feature flag), not tenant configuration.
func discardSpecialFlag(t tenant.Model) byte {
	if t.Region() == "GMS" && t.MajorVersion() <= 61 {
		return 2
	}
	return 3
}

// discardSplitsMarriageNumber reports whether the client sends the wedding
// invite's marriage number as TWO trailing int32 components rather than one.
// Only jms_v185 does (CMemoListDlg::SetRet@0x6c2d43 splits the invite message on
// "_" then "."); every GMS build sends a single value.
func discardSplitsMarriageNumber(t tenant.Model) bool {
	return t.Region() == "JMS"
}

// DiscardEntry is one memo the client asks to discard. A "special" entry is a
// WEDDING INVITATION (flag == discardSpecialFlag): SetRet's no-slot notice is
// the "...TO RECEIVE WEDDING INVITES" string and v95's PDB names the parsed
// value strMarriageNo. When such an invite is discarded and the client has a
// free ETC slot, it parses the marriage number out of the invite's message and
// appends it. Wedding invites are out of scope for the note feature (design
// §2.3) — Atlas never creates one, so MarriageNumber is 0 for every discard it
// actually handles; the fields exist so the codec models the full client wire.
type DiscardEntry struct {
	id             uint32
	flag           byte
	marriageNumber uint32
	unknown1       uint32
}

func (e DiscardEntry) Id() uint32 {
	return e.id
}

func (e DiscardEntry) Flag() byte {
	return e.flag
}

// MarriageNumber is the wedding-invite / marriage number a special entry carries
// (v95 SetRet's strMarriageNo). Zero on a normal (non-wedding) entry.
func (e DiscardEntry) MarriageNumber() uint32 {
	return e.marriageNumber
}

// Unknown1 is jms_v185's second marriage-number component — the client splits
// the invite message into two dot-separated int32s and sends both. Its precise
// role is not determinable from the (non-PDB) jms client, so it is left
// unnamed per convention. Zero on every GMS version (single value) and on
// normal entries.
func (e DiscardEntry) Unknown1() uint32 {
	return e.unknown1
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
// version (IDA-verified v48-v95 + jms): the number of wedding-invite memos in
// the client's local list, counted before the free-slot budget is applied. (An
// earlier revision wrongly modeled this as a jms-only field, which mis-aligned
// the GMS discard body by one byte and crashed the client on decode.)
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
// CMemoListDlg::SetRet OMITS a wedding-invite entry entirely once the free-slot
// budget (emptySlotCount) is exhausted — the loop sets a "not enough slot" flag
// and writes no bytes for it. So the wire carries totalCount minus the special
// entries that overflowed the budget. IDA-verified identical on every GMS
// version and jms.
func (m OperationDiscard) wireEntryCount() byte {
	if m.specialCount > m.emptySlotCount {
		return m.count - (m.specialCount - m.emptySlotCount)
	}
	return m.count
}

// Encode writes one entry: id, flag, and — for a wedding-invite (special)
// entry — the marriage number, plus jms's second component. The version facts
// (which flag is special, whether jms splits the number) resolve from the
// tenant on ctx.
func (e DiscardEntry) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	t := tenant.MustFromContext(ctx)
	specialFlag := discardSpecialFlag(t)
	splitMarriage := discardSplitsMarriageNumber(t)
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(e.id)
		w.WriteByte(e.flag)
		if e.flag == specialFlag {
			w.WriteInt(e.marriageNumber)
			if splitMarriage {
				w.WriteInt(e.unknown1)
			}
		}
		return w.Bytes()
	}
}

// Decode reads one entry from the discard body (see Encode).
func (e *DiscardEntry) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	specialFlag := discardSpecialFlag(t)
	splitMarriage := discardSplitsMarriageNumber(t)
	return func(r *request.Reader, options map[string]interface{}) {
		e.id = r.ReadUint32()
		e.flag = r.ReadByte()
		if e.flag == specialFlag {
			e.marriageNumber = r.ReadUint32()
			if splitMarriage {
				e.unknown1 = r.ReadUint32()
			}
		}
	}
}

func (m OperationDiscard) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.count)
		w.WriteByte(m.specialCount)
		w.WriteByte(m.emptySlotCount)
		for _, e := range m.entries {
			w.WriteByteArray(e.Encode(l, ctx)(options))
		}
		return w.Bytes()
	}
}

func (m *OperationDiscard) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.count = r.ReadByte()
		m.specialCount = r.ReadByte()
		m.emptySlotCount = r.ReadByte()

		wireEntries := m.wireEntryCount()
		m.entries = make([]DiscardEntry, 0, wireEntries)
		for i := byte(0); i < wireEntries; i++ {
			var e DiscardEntry
			e.Decode(l, ctx)(r, options)
			m.entries = append(m.entries, e)
		}
	}
}
