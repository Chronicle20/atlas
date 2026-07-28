package handler

import (
	"atlas-channel/character"
	"atlas-channel/pet"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	pet2 "github.com/Chronicle20/atlas/libs/atlas-packet/pet/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

func PetSpawnHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := pet2.Spawn{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		slot := p.Slot()
		lead := p.Lead()

		cp := character.NewProcessor(l, ctx)
		// PetAssetEnrichmentDecorator is required: it populates the cash pet
		// asset's PetSlot (and name/level/closeness) from atlas-pets. Without it
		// PetSlot defaults to 0, so the spawned check below (PetSlot != -1) is
		// always true and every spawn request is mishandled as a despawn.
		c, err := cp.GetById(cp.InventoryDecorator, cp.PetAssetEnrichmentDecorator)(s.CharacterId())
		if err != nil {
			return
		}
		a, ok := c.Inventory().Cash().FindBySlot(slot)
		if !ok {
			return
		}
		if !a.IsPet() {
			return
		}
		spawned := a.PetSlot() != -1
		if spawned {
			_ = pet.NewProcessor(l, ctx).Despawn(s.CharacterId(), a.PetId())
		} else {
			_ = pet.NewProcessor(l, ctx).Spawn(s.CharacterId(), a.PetId(), lead)
		}
	}
}
