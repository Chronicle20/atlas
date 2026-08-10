package shops

import (
	"atlas-npc/asset"
	"atlas-npc/character"
	"atlas-npc/commodities"
	"atlas-npc/kafka/message"
	"atlas-npc/kafka/message/shops"
	"math"
	"sort"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
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

// buyWithTokens executes the token-priced purchase path: the commodity is paid
// for with cm.TokenPrice() units of cm.TokenTemplateId() rather than mesos.
//
// It deliberately takes no discountPrice: the v83 client sends the *meso*
// price in the buy packet's final field (CShopDlg::SendBuyRequest @ 0x7566f4,
// COutPacket::Encode4(&v66, v8[6]) where v8[6] is ITEM+24 = mesoPrice), which
// is 0 for a token item. The commodity row is the only pricing authority.
//
// The token item is resolved from the row, never hardcoded — the v83
// client hardcodes the Perfect Pitch item id for its own local
// pre-check (0x41C3F0), but the server stays version- and
// vendor-agnostic.
//
// Guard order matters: the free-slot probe precedes any consumption so tokens
// are never destroyed for an item that cannot be received, mirroring the meso
// path at processor.go:462-467.
func (p *ProcessorImpl) buyWithTokens(mb *message.Buffer) func(c character.Model, cm commodities.Model, itemTemplateId uint32, quantity uint32) error {
	return func(c character.Model, cm commodities.Model, itemTemplateId uint32, quantity uint32) error {
		characterId := c.Id()

		if cm.TokenPrice() == 0 {
			p.l.Errorf("Character [%d] is attempting to buy item [%d] but no price is configured.", characterId, itemTemplateId)
			return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorGenericError))
		}
		if cm.TokenTemplateId() == 0 {
			p.l.Errorf("Character [%d] is attempting to buy item [%d] but it has a token price with no token item configured.", characterId, itemTemplateId)
			return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorGenericError))
		}

		tokenType, ok := inventory.TypeFromItemId(item.Id(cm.TokenTemplateId()))
		if !ok {
			p.l.Errorf("Character [%d] is attempting to buy item [%d] but token item [%d] is not a valid item.", characterId, itemTemplateId, cm.TokenTemplateId())
			return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorGenericError))
		}
		destinationType, ok := inventory.TypeFromItemId(item.Id(itemTemplateId))
		if !ok {
			p.l.Errorf("Character [%d] is attempting to buy item [%d] but it is not a valid item.", characterId, itemTemplateId)
			return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorGenericError))
		}

		// quantity arrives from the wire; uint32 x uint32 can wrap and produce
		// a small cost for a large purchase, which would grant items without
		// charging for them.
		total := uint64(cm.TokenPrice()) * uint64(quantity)
		if total == 0 || total > uint64(math.MaxUint32) {
			p.l.Errorf("Character [%d] is attempting to buy [%d] of item [%d] at token price [%d], which is not a valid cost.", characterId, quantity, itemTemplateId, cm.TokenPrice())
			return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorGenericError))
		}
		cost := uint32(total)

		draws, available := planTokenSpend(c.Inventory().CompartmentByType(tokenType).Assets(), cm.TokenTemplateId(), cost)
		if available < uint64(cost) {
			p.l.Errorf("Character [%d] is attempting to buy item [%d] but holds [%d] of token item [%d] and needs [%d].", characterId, itemTemplateId, available, cm.TokenTemplateId(), cost)
			return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorNeedMoreItems))
		}

		if _, err := c.Inventory().CompartmentByType(destinationType).NextFreeSlot(); err != nil {
			p.l.WithError(err).Errorf("Cannot locate free slot for character [%d].", characterId)
			return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorInventoryFull))
		}

		for _, d := range draws {
			if err := p.compP.RequestDestroyItem(mb)(characterId, tokenType, d.slot, d.quantity); err != nil {
				return err
			}
		}
		if err := p.compP.RequestCreateItem(mb)(characterId, itemTemplateId, quantity); err != nil {
			return err
		}

		p.l.Debugf("Character [%d] bought [%d] of item [%d] for [%d] of token item [%d].", characterId, quantity, itemTemplateId, cost, cm.TokenTemplateId())
		return mb.Put(shops.EnvStatusEventTopic, okEventProvider(characterId))
	}
}
