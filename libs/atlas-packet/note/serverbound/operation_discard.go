package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// Discard-shape config keys, read from the NoteOperationHandle handler options
// (options["discard"]). CMemoListDlg::SetRet writes a per-entry "special"
// (gift/reward memo) shape whose flag sentinel and trailing token count vary by
// client version; rather than hard-code those version literals in the codec,
// each tenant template supplies them so the same code decodes every version.
const (
	// DiscardConfigKey groups the discard-shape values under the handler's
	// options map.
	DiscardConfigKey = "discard"
	// DiscardSpecialFlagKey is the memo flag CMemoListDlg::SetRet treats as a
	// gift/reward ("special") entry — the only entry kind that claims a free
	// ETC slot and carries trailing reward token(s). IDA-verified 2 on GMS
	// v48/v61, 3 on v72+ and jms.
	DiscardSpecialFlagKey = "specialFlag"
	// DiscardClaimValueCountKey is how many trailing uint32 reward tokens a
	// special entry carries — 1 on every GMS SetRet, 2 on jms_v185.
	DiscardClaimValueCountKey = "claimValueCount"
)

// discardShape holds the version-specific bits of the discard body, resolved
// from tenant config so the codec carries no per-version literals.
type discardShape struct {
	specialFlag     byte
	claimValueCount int
}

// resolveDiscardShape reads the discard-shape config from the handler options
// (options["discard"]). A missing/blank config falls back to the majority GMS
// shape (special flag 3, one claim token) so decode stays well-defined for
// plain-note batches, where no entry is special and the values are never read.
func resolveDiscardShape(options map[string]interface{}) discardShape {
	shape := discardShape{specialFlag: 3, claimValueCount: 1}
	discard, ok := options[DiscardConfigKey].(map[string]interface{})
	if !ok {
		return shape
	}
	if v, ok := discard[DiscardSpecialFlagKey].(float64); ok {
		shape.specialFlag = byte(v)
	}
	if v, ok := discard[DiscardClaimValueCountKey].(float64); ok {
		shape.claimValueCount = int(v)
	}
	return shape
}

// DiscardEntry is one memo the client asks to discard. A "special" entry — a
// gift/reward memo (flag == the version's special flag) that the client granted
// a free ETC slot — additionally echoes the reward token(s) it parsed out of
// the memo's message so the server can grant them: v95's CMemoListDlg::SetRet
// stores that value in a local named strMarriageNo (a wedding-invite number),
// and per design §1.5 it is a reward on v48, itemId on v61, mesos on v72, etc.
// Gift/reward memos are out of scope for the note feature (design §2.3) — Atlas
// never creates one, so ClaimValues is empty for every discard it actually
// handles; the field exists so the codec models the full wire and a future gift
// task has the shape and evidence ready.
type DiscardEntry struct {
	id          uint32
	flag        byte
	claimValues []uint32
}

func (e DiscardEntry) Id() uint32 {
	return e.id
}

func (e DiscardEntry) Flag() byte {
	return e.flag
}

// ClaimValues returns the reward token(s) a special (gift/reward) memo entry
// carries — nil for a normal entry. See DiscardEntry.
func (e DiscardEntry) ClaimValues() []uint32 {
	return e.claimValues
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
// local list whose flag equals the version's special flag, counted before the
// free-slot budget is applied. (An earlier revision wrongly modeled this as a
// jms-only field, which mis-aligned the GMS discard body by one byte and
// crashed the client on decode.)
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
// CMemoListDlg::SetRet OMITS a special entry entirely once the free-slot budget
// (emptySlotCount) is exhausted — the loop sets a "not enough slot" flag and
// writes no bytes for it. So the wire carries totalCount minus the special
// entries that overflowed the budget. IDA-verified identical on every GMS
// version and jms.
func (m OperationDiscard) wireEntryCount() byte {
	if m.specialCount > m.emptySlotCount {
		return m.count - (m.specialCount - m.emptySlotCount)
	}
	return m.count
}

func (m OperationDiscard) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		shape := resolveDiscardShape(options)
		w.WriteByte(m.count)
		w.WriteByte(m.specialCount)
		w.WriteByte(m.emptySlotCount)
		for _, e := range m.entries {
			w.WriteInt(e.id)
			w.WriteByte(e.flag)
			if e.flag == shape.specialFlag {
				for i := 0; i < shape.claimValueCount; i++ {
					var v uint32
					if i < len(e.claimValues) {
						v = e.claimValues[i]
					}
					w.WriteInt(v)
				}
			}
		}
		return w.Bytes()
	}
}

func (m *OperationDiscard) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		shape := resolveDiscardShape(options)
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
			if e.flag == shape.specialFlag && shape.claimValueCount > 0 {
				e.claimValues = make([]uint32, shape.claimValueCount)
				for j := 0; j < shape.claimValueCount; j++ {
					e.claimValues[j] = r.ReadUint32()
				}
			}
			m.entries = append(m.entries, e)
		}
	}
}
