package craft

import (
	"atlas-maker/recipe"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// LeftoverConsumeQuantity is mode 3's leftover consumption quantity (OQ-7).
//
// All four decompiled clients render the mode-3 consumption log as
// Format(SP_293_YOU_HAVE_LOST_ITEMS_S_D, <name>, 100) -- the literal 100 is
// hard-coded in the client -- while the reference archive's group-0 recipe
// lists its leftover material with count: 1. design §5 "New open question"
// (OQ-7) decided 100, and directed Task 23 to confirm that against the
// reference server's crystal path before implementing.
//
// Confirmation outcome (Task 23): no reference-server crystallization
// source is present in this repository or its docs/ tree beyond the
// decompiled-client evidence design.md §5 already cites (a targeted search
// of docs/tasks/task-285-maker-skill-crafting/ and the repo's reference
// server material turned up no independent crystal-path implementation to
// diff against). The design's decision therefore stands unreversed on the
// only evidence available: every client build agrees on 100, so a server
// that consumed 1 would leave the client's "-100" chat log permanently
// wrong. The recipe's own `count` is ignored for group 0.
const LeftoverConsumeQuantity = 100

// Consumption is one concrete (inventoryType, slot, quantity) tuple a craft
// destroys. TemplateId always accompanies it, never left zero, so a saga
// step built from it can carry DestroyAssetFromSlotPayload.TemplateId and
// let the orchestrator's compensator re-create the asset
// (libs/atlas-saga/payloads.go:141) if a later step in the same saga fails.
type Consumption struct {
	InventoryType inventory.Type
	Slot          int16
	Quantity      uint32
	TemplateId    item.Id
}

// Plan is a craft's exact material consumption, resolved against a Snapshot
// at concrete slots in ascending order -- never a quantity read from the
// request (the NFR "never trust client quantities"). Computing it in full
// before any saga step is built is what makes design §7's "rejection is
// pre-mutation" true: a Plan that turns out to accompany a rejected craft
// is simply discarded, having touched nothing.
type Plan struct {
	Consumptions []Consumption
}

// resolveConsumption walks itemId's held slots (Snapshot.Slots, already
// ascending by construction) and destroys up to count total quantity,
// spreading across as many slots as needed -- one Consumption per slot
// actually touched, never a single entry claiming more than that slot
// holds. If the snapshot holds less than count, it destroys only what is
// actually there rather than fabricating a shortfall; a caller that must
// reject on a shortfall (a recipe's required materials) checks that via
// eligibility before ever building a Plan.
func resolveConsumption(snap Snapshot, invType inventory.Type, itemId item.Id, count uint32) []Consumption {
	if count == 0 {
		return nil
	}
	var out []Consumption
	remaining := count
	for _, sh := range snap.Slots(itemId) {
		if remaining == 0 {
			break
		}
		take := sh.Quantity
		if take > remaining {
			take = remaining
		}
		out = append(out, Consumption{InventoryType: invType, Slot: sh.Slot, Quantity: take, TemplateId: itemId})
		remaining -= take
	}
	return out
}

// gemFrequency counts how many times each gem id appears in gemItemIds. A
// duplicate entry naming a gem the character holds only once resolves,
// through resolveConsumption's own shortfall handling, to exactly one
// destroy step -- never two -- which is what makes the plan immune to a
// client repeating an id in its gem list (NFR "never trust client
// quantities").
func gemFrequency(gemItemIds []item.Id) map[item.Id]uint32 {
	freq := make(map[item.Id]uint32, len(gemItemIds))
	for _, id := range gemItemIds {
		freq[id]++
	}
	return freq
}

// BuildCreatePlan resolves mode 1|2's consumption: every recipe material at
// its exact count (eligibility already confirmed it is held), each
// distinct gem named in gemItemIds at the frequency it appears in the
// request (dropped entirely, not rejected, when unheld -- FR-3.2), and the
// recipe's catalyst exactly once when useCatalyst is set and the catalyst
// is actually held.
func BuildCreatePlan(snap Snapshot, r recipe.Model, gemItemIds []item.Id, useCatalyst bool) Plan {
	var out []Consumption

	for _, mat := range r.Materials() {
		invType, _ := inventory.TypeFromItemId(mat.ItemId)
		out = append(out, resolveConsumption(snap, invType, mat.ItemId, mat.Count)...)
	}

	// Iterate gemItemIds in the order first named, deduplicated: map
	// iteration order is random and the destroy-step order must be
	// reproducible.
	seen := make(map[item.Id]bool, len(gemItemIds))
	freq := gemFrequency(gemItemIds)
	for _, id := range gemItemIds {
		if seen[id] {
			continue
		}
		seen[id] = true
		invType, _ := inventory.TypeFromItemId(id)
		out = append(out, resolveConsumption(snap, invType, id, freq[id])...)
	}

	if useCatalyst && r.Catalyst() != 0 {
		invType, _ := inventory.TypeFromItemId(r.Catalyst())
		out = append(out, resolveConsumption(snap, invType, r.Catalyst(), 1)...)
	}

	return Plan{Consumptions: out}
}

// BuildCrystalPlan resolves mode 3's consumption: the leftover, at
// LeftoverConsumeQuantity -- never the archive's group-0 `count` (OQ-7).
func BuildCrystalPlan(snap Snapshot, leftoverItemId item.Id) Plan {
	invType, _ := inventory.TypeFromItemId(leftoverItemId)
	return Plan{Consumptions: resolveConsumption(snap, invType, leftoverItemId, LeftoverConsumeQuantity)}
}

// AppliedGems is the subset of gemItemIds (deduplicated, order preserved)
// the character actually holds -- the gems whose reagent stat contribution
// applies to a crafted equip, and the same set BuildCreatePlan resolves
// destroy steps for.
func AppliedGems(snap Snapshot, gemItemIds []item.Id) []item.Id {
	seen := make(map[item.Id]bool, len(gemItemIds))
	var out []item.Id
	for _, id := range gemItemIds {
		if seen[id] {
			continue
		}
		seen[id] = true
		if snap.Held(id) > 0 {
			out = append(out, id)
		}
	}
	return out
}
