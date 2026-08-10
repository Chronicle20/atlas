package respawn

import (
	"atlas-channel/asset"
	"atlas-channel/character"
	map_ "atlas-channel/data/map"
	channelInventory "atlas-channel/inventory"
	"atlas-channel/saga"
	"atlas-channel/socket/writer"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
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
	return p.createRespawnSaga(f, characterId, rp)
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
