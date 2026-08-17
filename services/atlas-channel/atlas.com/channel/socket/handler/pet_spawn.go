package handler

import (
	"atlas-channel/character"
	"atlas-channel/pet"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"time"

	"github.com/sirupsen/logrus"

	pet2 "github.com/Chronicle20/atlas/libs/atlas-packet/pet/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

func PetSpawnHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := pet2.Spawn{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		slot := p.Slot()
		lead := p.Lead()

		// The client latches its exclusive-request lock the moment it sends
		// this, so a request that resolves to no action still owes it a
		// release. Every bail-out below unlocks.
		enableActions := func() { _ = session.EnableActions(l)(ctx)(wp)(s) }

		cp := character.NewProcessor(l, ctx)
		// PetAssetEnrichmentDecorator is required: it populates the cash pet
		// asset's PetSlot (and name/level/closeness) from atlas-pets. Without it
		// PetSlot defaults to 0, so the spawned check below (PetSlot != -1) is
		// always true and every spawn request is mishandled as a despawn.
		c, err := cp.GetById(cp.InventoryDecorator, cp.PetAssetEnrichmentDecorator)(s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to load character [%d] for a pet summon.", s.CharacterId())
			enableActions()
			return
		}
		a, ok := c.Inventory().Cash().FindBySlot(slot)
		if !ok {
			l.Warnf("Character [%d] attempted to summon from empty cash slot [%d].", s.CharacterId(), slot)
			enableActions()
			return
		}
		if !a.IsPet() {
			l.Warnf("Character [%d] attempted to summon non-pet item [%d] in cash slot [%d].", s.CharacterId(), a.TemplateId(), slot)
			enableActions()
			return
		}
		spawned := a.PetSlot() != -1
		if spawned {
			_ = pet.NewProcessor(l, ctx).Despawn(s.CharacterId(), a.PetId())
			return
		}

		// A dried-up pet cannot be summoned: atlas-pets rejects the command
		// with ErrPetExpired and, because the rejection aborts the transactional
		// emit, produces NO status event. Nothing would reach the client, and
		// the exclusive-request lock it latched on send (CWvsContext::
		// SendActivatePetRequest sets m_bExclRequestSent right after SendPacket)
		// would never clear — the client freezes.
		//
		// An up-to-date client does not send this for a dried-up pet at all; it
		// takes the IsDead arm instead. This gate covers the window where it
		// does anyway: a pet that expired while the item block the client holds
		// still said otherwise.
		if !a.PetDeadDate().IsZero() && !time.Now().Before(a.PetDeadDate()) {
			l.Warnf("Character [%d] attempted to summon pet [%d], which dried up at [%s].", s.CharacterId(), a.PetId(), a.PetDeadDate())
			enableActions()
			return
		}

		_ = pet.NewProcessor(l, ctx).Spawn(s.CharacterId(), a.PetId(), lead)
	}
}
