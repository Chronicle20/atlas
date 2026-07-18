package handler

import (
	"atlas-channel/asset"
	"atlas-channel/character"
	"atlas-channel/compartment"
	"atlas-channel/data/skill/effect/statup"
	"context"
	"sort"

	compartmentMsg "atlas-channel/kafka/message/compartment"
	once "atlas-channel/kafka/once/compartment"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

// StarDraw is one slot-level consume of a chosen throwing star for the
// Shadow Stars cast cost.
type StarDraw struct {
	Slot     int16
	ItemId   uint32
	Quantity int16
}

// validateShadowStar reports whether starItemId is a throwing-star
// classification AND present (quantity > 0) in the caster's consumable assets.
func validateShadowStar(assets []asset.Model, starItemId uint32) bool {
	if !item.IsThrowingStar(item.Id(starItemId)) {
		return false
	}
	for _, a := range assets {
		if a.TemplateId() == starItemId && a.Quantity() > 0 {
			return true
		}
	}
	return false
}

// resolveStarConsume draws `count` of exactly starItemId across ascending
// consumable slots. `available` is the sum of planned draws
// (min(count, total owned)); available < count signals a shortfall.
func resolveStarConsume(assets []asset.Model, starItemId uint32, count int) (draws []StarDraw, available int) {
	matching := make([]asset.Model, 0, len(assets))
	for _, a := range assets {
		if a.TemplateId() == starItemId && a.Quantity() > 0 {
			matching = append(matching, a)
		}
	}
	if len(matching) == 0 || count <= 0 {
		return nil, 0
	}
	sort.Slice(matching, func(i, j int) bool { return matching[i].Slot() < matching[j].Slot() })

	remaining := count
	draws = make([]StarDraw, 0, len(matching))
	for _, a := range matching {
		if remaining <= 0 {
			break
		}
		draw := int(a.Quantity())
		if draw > remaining {
			draw = remaining
		}
		draws = append(draws, StarDraw{Slot: a.Slot(), ItemId: starItemId, Quantity: int16(draw)})
		remaining -= draw
		available += draw
	}
	return draws, available
}

// rewriteShadowClawStatups returns a copy of statups with the SHADOW_CLAW
// entry's amount set to starItemId, preserving any other statups. atlas-data's
// produceBuffStatAmount drops zero-value statups (its `if value != 0` guard),
// so the SHADOW_CLAW placeholder never survives the reader for 4121006 — the
// statups atlas-channel actually receives carry NO SHADOW_CLAW entry. If the
// entry is absent, one is appended so the star id always reaches the buff.
// Mirrors mount.go's tamedMountStatups for MONSTER_RIDING.
func rewriteShadowClawStatups(statups []statup.Model, starItemId uint32) []statup.Model {
	out := make([]statup.Model, 0, len(statups)+1)
	hasClaw := false
	for _, su := range statups {
		if su.Mask() == string(charconst.TemporaryStatTypeShadowClaw) {
			out = append(out, statup.NewModel(su.Mask(), int32(starItemId)))
			hasClaw = true
			continue
		}
		out = append(out, su)
	}
	if !hasClaw {
		out = append(out, statup.NewModel(string(charconst.TemporaryStatTypeShadowClaw), int32(starItemId)))
	}
	return out
}

// resolveShadowStarsCast validates the chosen star and resolves the buff
// statups + consume draws for a Shadow Stars cast. castCost is the WZ
// `bulletConsume` (200 in reference data) — the one-time bulk star charge.
// ok=false means the star is invalid (wrong classification or not owned) and the
// cast MUST abort — the returned rewritten/draws are nil. shortfall reports
// available < castCost.
func resolveShadowStarsCast(assets []asset.Model, statups []statup.Model, starItemId uint32, castCost int) (rewritten []statup.Model, draws []StarDraw, shortfall bool, ok bool) {
	if !validateShadowStar(assets, starItemId) {
		return nil, nil, false, false
	}
	draws, available := resolveStarConsume(assets, starItemId, castCost)
	rewritten = rewriteShadowClawStatups(statups, starItemId)
	return rewritten, draws, available < castCost, true
}

// loadCasterInventoryFunc is the caster-inventory load seam tests can replace.
// Production loads the character with the inventory decorator and returns the
// consumable (USE) compartment assets — the same decorated load the generic
// item-consume block in UseSkill uses.
var loadCasterInventoryFunc = func(cp character.Processor, characterId uint32) ([]asset.Model, error) {
	c, err := cp.GetById(cp.InventoryDecorator)(characterId)
	if err != nil {
		return nil, err
	}
	return c.Inventory().Consumable().Assets(), nil
}

// emitStarConsume charges the Shadow Stars cast cost by reserving then consuming
// each StarDraw from the USE compartment. Mirrors the projectile Emit path:
// register a one-time reservation-observed handler that issues the consume, then
// request the reservation. Reservation atomicity means a slot that no longer
// holds the item fails cleanly without over-consuming.
func emitStarConsume(l logrus.FieldLogger, ctx context.Context, characterId uint32, draws []StarDraw) error {
	if len(draws) == 0 {
		return nil
	}
	cpp := compartment.NewProcessor(l, ctx)
	t, err := topic.EnvProvider(l)(compartmentMsg.EnvEventTopicStatus)()
	if err != nil {
		return err
	}
	for _, draw := range draws {
		draw := draw
		txId := uuid.New()
		validator := once.ReservationValidator(txId, draw.ItemId)
		handler := reservedStarToConsume(l, cpp, characterId, txId, inventory.TypeValueUse, draw.Slot)
		if _, rerr := consumer.GetManager().RegisterHandler(t, message.AdaptHandler(message.OneTimeConfig(validator, handler))); rerr != nil {
			l.WithError(rerr).WithField("characterId", characterId).
				Errorf("Unable to register one-time consume handler for Shadow Stars reservation.")
			continue
		}
		reserves := []compartmentMsg.ItemBody{{Source: draw.Slot, ItemId: draw.ItemId, Quantity: draw.Quantity}}
		if rerr := cpp.RequestReserve(txId, characterId, inventory.TypeValueUse, reserves); rerr != nil {
			l.WithError(rerr).WithField("characterId", characterId).WithField("slot", draw.Slot).
				Errorf("Unable to emit Shadow Stars reservation request.")
		}
	}
	return nil
}

func reservedStarToConsume(l logrus.FieldLogger, cpp compartment.Processor, characterId uint32, txId uuid.UUID, invType inventory.Type, slot int16) message.Handler[compartmentMsg.StatusEvent[compartmentMsg.ReservedEventBody]] {
	return func(_ logrus.FieldLogger, _ context.Context, _ compartmentMsg.StatusEvent[compartmentMsg.ReservedEventBody]) {
		if err := cpp.Consume(txId, characterId, invType, slot); err != nil {
			l.WithError(err).WithField("characterId", characterId).WithField("slot", slot).
				Errorf("Unable to emit Shadow Stars consume command.")
		}
	}
}
