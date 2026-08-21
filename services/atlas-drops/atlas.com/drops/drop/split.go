package drop

import (
	"atlas-drops/party"
	"sort"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

// Recipient is one character's share of a split meso drop.
type Recipient struct {
	CharacterId uint32
	Amount      uint32
	Picker      bool
}

// splitMeso divides meso evenly among the picker and every online party member
// whose recorded location matches the drop's field on all four dimensions.
// The picker is always included, even when their own party-member record is
// stale or reports offline, so the recipient set is never empty. members == nil
// (no party, or a failed party lookup) collapses to a single full-amount award
// to the picker — the degrade is the empty input, not a special case.
//
// Integer division: the remainder is discarded, nobody receives it. Recipients
// are returned sorted by character id with exactly one Picker: true.
func splitMeso(f field.Model, meso uint32, pickerId uint32, members []party.MemberModel) []Recipient {
	ids := []uint32{pickerId}
	seen := map[uint32]bool{pickerId: true}
	for _, m := range members {
		if seen[m.Id()] {
			continue
		}
		if !m.Online() {
			continue
		}
		mf := m.Field()
		if mf.WorldId() != f.WorldId() || mf.ChannelId() != f.ChannelId() || mf.MapId() != f.MapId() || mf.Instance() != f.Instance() {
			continue
		}
		seen[m.Id()] = true
		ids = append(ids, m.Id())
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	share := meso / uint32(len(ids))
	rs := make([]Recipient, 0, len(ids))
	for _, id := range ids {
		rs = append(rs, Recipient{CharacterId: id, Amount: share, Picker: id == pickerId})
	}
	return rs
}
