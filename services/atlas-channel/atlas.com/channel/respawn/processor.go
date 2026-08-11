package respawn

import (
	"atlas-channel/asset"
	"atlas-channel/character"
	map_ "atlas-channel/data/map"
	channelInventory "atlas-channel/inventory"
	"atlas-channel/saga"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character"
	charcb "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
)

// Processor interface defines operations for character respawn
type Processor interface {
	// Respawn handles character death and respawn logic. useDeathItem is the
	// client's Change.Premium() byte — 1 from the revive dialog's OK button,
	// 0 from Cancel. A Cancel must not spend the player's wheel.
	Respawn(f field.Model, characterId uint32, useDeathItem bool) error
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	wp  writer.Producer
	cp  character.Processor
	ip  channelInventory.Processor
	mp  map_.Processor
	sp  saga.Processor
}

// NewProcessor creates a new respawn processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		wp:  wp,
		cp:  character.NewProcessor(l, ctx),
		ip:  channelInventory.NewProcessor(l, ctx),
		mp:  map_.NewProcessor(l, ctx),
		sp:  saga.NewProcessor(l, ctx),
	}
}

var _ Processor = (*ProcessorImpl)(nil)

// Respawn handles character death and respawn logic
func (p *ProcessorImpl) Respawn(f field.Model, characterId uint32, useDeathItem bool) error {
	currentMapId := f.MapId()
	p.l.Debugf("Processing respawn for character [%d] on map [%d]. useDeathItem [%t].", characterId, currentMapId, useDeathItem)

	c, err := p.cp.GetById()(characterId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to get character [%d] for respawn.", characterId)
		return err
	}
	inv, err := p.ip.GetByCharacterId(characterId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to get inventory for character [%d].", characterId)
		return err
	}
	mapData, err := p.mp.GetById(currentMapId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to get map [%d] data for respawn.", currentMapId)
		return err
	}

	rp := planRespawn(c, inv, mapFactsOf(mapData), currentMapId, useDeathItem)
	if err = p.createRespawnSaga(f, characterId, rp); err != nil {
		return err
	}
	// A failed broadcast is logged, never fatal — the revive has already been
	// committed to the saga at this point.
	p.announceProtectOnDie(f, characterId, rp.Protective)
	p.announceUpgradeTombItemUse(f, characterId, rp.Wheel)
	return nil
}

// announceUpgradeTombItemUse tells the reviving player they spent a Wheel of
// Destiny and how many are left. The v83 client's CUser::OnEffect mode-21 arm
// (@0x9387d0) reads one byte and CHATLOG_ADDs
// SP_5241 "You have used 1 Wheel of Destiny in order to revive at the current
// map. (%d left)" — a first-person chat line, so it is sent to the owner only.
// CUserPool::OnUserRemotePacket routes the foreign variant into the very same
// arm on the remote user object, which would put that first-person sentence in
// every bystander's chat log; no foreign broadcast is emitted.
func (p *ProcessorImpl) announceUpgradeTombItemUse(f field.Model, characterId uint32, a *asset.Model) {
	if a == nil {
		return
	}
	remaining := usesRemaining(a)

	p.l.Debugf("Character [%d] consumed a wheel of destiny [%d]: [%d] uses remaining.", characterId, a.TemplateId(), remaining)

	err := session.NewProcessor(p.l, p.ctx).IfPresentByCharacterId(f.Channel())(characterId,
		session.Announce(p.l)(p.ctx)(p.wp)(charcb.CharacterEffectWriter)(
			charpkt.CharacterUpgradeTombItemUseEffectBody(remaining)))
	if err != nil {
		p.l.WithError(err).Errorf("Unable to announce wheel of destiny use to character [%d].", characterId)
	}
}

// announceProtectOnDie tells the dying player that a death protection item
// absorbed the experience loss, and how many uses are left. The mode byte is
// resolved from the tenant's CharacterEffect writer options
// (PROTECT_ON_DIE_ITEM_USE) rather than hard-coded.
//
// Owner only, same reasoning as announceUpgradeTombItemUse: the mode-6 arm of
// CUser::OnEffect (@0x937e8d) has no visual — it only CHATLOG_ADDs the
// first-person SP_2966 "The EXP did not drop after using the Safety Charm
// once. (%d days, %d times left)", and CUserPool::OnUserRemotePacket routes
// the foreign variant into that same arm on the remote user object. A
// bystander would therefore see nothing but someone else's sentence in their
// own chat log. PRD FR-5.1 asked for the foreign broadcast on the assumption
// it rendered something over the dying character; the client says it does not,
// so it is deliberately not sent.
func (p *ProcessorImpl) announceProtectOnDie(f field.Model, characterId uint32, a *asset.Model) {
	if a == nil {
		return
	}
	templateId := a.TemplateId()
	safetyCharm := item.IsSafetyCharm(item.Id(templateId))
	remaining := usesRemaining(a)
	days := expirationDays(a.Expiration(), time.Now())

	p.l.Debugf("Character [%d] consumed death protection item [%d]: [%d] uses remaining, [%d] days, safetyCharm [%t].", characterId, templateId, remaining, days, safetyCharm)

	err := session.NewProcessor(p.l, p.ctx).IfPresentByCharacterId(f.Channel())(characterId,
		session.Announce(p.l)(p.ctx)(p.wp)(charcb.CharacterEffectWriter)(
			charpkt.CharacterProtectOnDieItemUseEffectBody(safetyCharm, remaining, days, templateId)))
	if err != nil {
		p.l.WithError(err).Errorf("Unable to announce protect-on-die effect to character [%d].", characterId)
	}
}

// findProtectiveItem returns the death-protection asset that suppresses the
// experience loss, or nil. Cash Safety Charm first, then the two ETC items.
func findProtectiveItem(inv channelInventory.Model) *asset.Model {
	if a, found := inv.Cash().FindFirstByItemId(uint32(item.SafetyCharmId)); found && a != nil && a.Quantity() >= 1 {
		return a
	}
	if a, found := inv.ETC().FindFirstByItemId(uint32(item.EasterBasketId)); found && a != nil && a.Quantity() >= 1 {
		return a
	}
	if a, found := inv.ETC().FindFirstByItemId(uint32(item.ProtectOnDeathId)); found && a != nil && a.Quantity() >= 1 {
		return a
	}
	return nil
}

// calculateExpLoss calculates the experience loss on death
func calculateExpLoss(c character.Model, mf mapFacts, hasProtection bool) uint32 {
	// Beginners don't lose experience
	if job.IsBeginner(c.JobId()) {
		return 0
	}

	// Map with NoExpLossOnDeath field limit
	if mf.NoExpLossOnDeath {
		return 0
	}

	// Has protective item
	if hasProtection {
		return 0
	}

	// Calculate experience loss as a percentage of current experience
	// This is a simplified calculation - ideally should use exp needed for level
	currentExp := c.Experience()
	if currentExp == 0 {
		return 0
	}

	var lossPercentage float64
	if mf.Town {
		// Town = 1% loss
		lossPercentage = 0.01
	} else if c.Luck() < 50 {
		// Non-town with luck < 50 = 10% loss
		lossPercentage = 0.10
	} else {
		// Non-town with luck >= 50 = 5% loss
		lossPercentage = 0.05
	}

	loss := uint32(float64(currentExp) * lossPercentage)
	return loss
}

// createRespawnSaga creates and submits the respawn saga
func (p *ProcessorImpl) createRespawnSaga(f field.Model, characterId uint32, rp respawnPlan) error {
	transactionId := uuid.New()
	now := time.Now()
	steps := respawnSagaSteps(f, characterId, rp, now)

	s := saga.Saga{
		TransactionId: transactionId,
		SagaType:      saga.CharacterRespawn,
		InitiatedBy:   "RESPAWN",
		Steps:         steps,
	}

	p.l.Debugf("Creating respawn saga [%s] for character [%d] with [%d] steps.", transactionId, characterId, len(steps))
	return p.sp.Create(s)
}
