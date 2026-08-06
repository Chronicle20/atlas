package shops

import (
	"atlas-npc/asset"
	"sort"
)

// tokenDraw is one slot-scoped withdrawal from a token stack. The compartment
// command contract is slot-based (compartment/producer.go:32-44), so a spend
// that straddles stacks becomes one draw — and one DESTROY command — per slot.
type tokenDraw struct {
	slot     int16
	quantity uint32
}

// planTokenSpend computes how to withdraw cost units of tokenTemplateId from
// as, drawing from the lowest slot first, and reports the total quantity held
// across every matching slot.
//
// The returned plan is only valid to execute when available >= cost; when the
// character is short, the draws describe everything they hold and the caller
// must refuse instead of executing them. available is uint64 because summing
// uint32 stack quantities can itself overflow uint32.
func planTokenSpend(as []asset.Model, tokenTemplateId uint32, cost uint32) ([]tokenDraw, uint64) {
	matching := make([]asset.Model, 0, len(as))
	for _, a := range as {
		if a.TemplateId() != tokenTemplateId || a.Quantity() == 0 {
			continue
		}
		matching = append(matching, a)
	}
	sort.Slice(matching, func(i, j int) bool {
		return matching[i].Slot() < matching[j].Slot()
	})

	var available uint64
	for _, a := range matching {
		available += uint64(a.Quantity())
	}

	draws := make([]tokenDraw, 0, len(matching))
	remaining := cost
	for _, a := range matching {
		if remaining == 0 {
			break
		}
		take := a.Quantity()
		if take > remaining {
			take = remaining
		}
		draws = append(draws, tokenDraw{slot: a.Slot(), quantity: take})
		remaining -= take
	}
	return draws, available
}
