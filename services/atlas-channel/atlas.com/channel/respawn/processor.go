package respawn

import (
	"atlas-channel/character"
	map_ "atlas-channel/data/map"
	channelInventory "atlas-channel/inventory"
	"atlas-channel/saga"
	"context"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	inventoryConst "github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/item"
	"github.com/Chronicle20/atlas-constants/job"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Processor interface defines operations for character respawn
type Processor interface {
	// Respawn handles character death and respawn logic
	Respawn(ch channel.Model, characterId uint32, currentMapId _map.Id) error
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	cp  character.Processor
	ip  channelInventory.Processor
	mp  map_.Processor
	sp  saga.Processor
}

// NewProcessor creates a new respawn processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		cp:  character.NewProcessor(l, ctx),
		ip:  channelInventory.NewProcessor(l, ctx),
		mp:  map_.NewProcessor(l, ctx),
		sp:  saga.NewProcessor(l, ctx),
	}
}

// Respawn handles character death and respawn logic
func (p *ProcessorImpl) Respawn(ch channel.Model, characterId uint32, currentMapId _map.Id) error {
	p.l.Debugf("Processing respawn for character [%d] on map [%d].", characterId, currentMapId)

	// Get character data
	c, err := p.cp.GetById()(characterId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to get character [%d] for respawn.", characterId)
		return err
	}

	// Get inventory data
	inv, err := p.ip.GetByCharacterId(characterId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to get inventory for character [%d].", characterId)
		return err
	}

	// Get map data for return map and field limits
	mapData, err := p.mp.GetById(currentMapId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to get map [%d] data for respawn.", currentMapId)
		return err
	}

	// Check for Wheel of Fortune in Cash inventory
	hasWheelOfFortune := false
	if a, found := inv.Cash().FindFirstByItemId(uint32(item.WheelOfFortuneId)); found && a != nil {
		hasWheelOfFortune = true
	}

	// Check for protective items
	protectiveItem, _ := p.findProtectiveItem(inv)

	// Calculate experience loss
	expLoss := p.calculateExpLoss(c, mapData, protectiveItem != nil)

	// Determine target map
	targetMapId := currentMapId
	if !hasWheelOfFortune {
		targetMapId = mapData.ReturnMapId()
	}

	// Create respawn saga
	return p.createRespawnSaga(ch, characterId, targetMapId, hasWheelOfFortune, protectiveItem, expLoss)
}

// findProtectiveItem searches for a death protection item in the inventory
// Returns the item and which inventory type it was found in
func (p *ProcessorImpl) findProtectiveItem(inv channelInventory.Model) (*uint32, inventoryConst.Type) {
	// Check Cash inventory for Safety Charm
	if a, found := inv.Cash().FindFirstByItemId(uint32(item.SafetyCharmId)); found && a != nil {
		templateId := a.TemplateId()
		return &templateId, inventoryConst.TypeValueCash
	}

	// Check ETC inventory for Easter Basket
	if a, found := inv.ETC().FindFirstByItemId(uint32(item.EasterBasketId)); found && a != nil {
		templateId := a.TemplateId()
		return &templateId, inventoryConst.TypeValueETC
	}

	// Check ETC inventory for ProtectOnDeath
	if a, found := inv.ETC().FindFirstByItemId(uint32(item.ProtectOnDeathId)); found && a != nil {
		templateId := a.TemplateId()
		return &templateId, inventoryConst.TypeValueETC
	}

	return nil, 0
}

// calculateExpLoss calculates the experience loss on death
func (p *ProcessorImpl) calculateExpLoss(c character.Model, mapData map_.Model, hasProtection bool) uint32 {
	// Beginners don't lose experience
	if job.IsBeginner(c.JobId()) {
		p.l.Debugf("Character [%d] is a beginner, no experience loss.", c.Id())
		return 0
	}

	// Map with NoExpLossOnDeath field limit
	if mapData.NoExpLossOnDeath() {
		p.l.Debugf("Map has no exp loss field limit, no experience loss for character [%d].", c.Id())
		return 0
	}

	// Has protective item
	if hasProtection {
		p.l.Debugf("Character [%d] has protective item, no experience loss.", c.Id())
		return 0
	}

	// Calculate experience loss as a percentage of current experience
	// This is a simplified calculation - ideally should use exp needed for level
	currentExp := c.Experience()
	if currentExp == 0 {
		return 0
	}

	var lossPercentage float64
	if mapData.Town() {
		// Town = 1% loss
		lossPercentage = 0.01
		p.l.Debugf("Character [%d] dying in town, 1%% experience loss.", c.Id())
	} else if c.Luck() < 50 {
		// Non-town with luck < 50 = 10% loss
		lossPercentage = 0.10
		p.l.Debugf("Character [%d] with luck [%d] < 50, 10%% experience loss.", c.Id(), c.Luck())
	} else {
		// Non-town with luck >= 50 = 5% loss
		lossPercentage = 0.05
		p.l.Debugf("Character [%d] with luck [%d] >= 50, 5%% experience loss.", c.Id(), c.Luck())
	}

	loss := uint32(float64(currentExp) * lossPercentage)
	p.l.Debugf("Character [%d] will lose [%d] experience.", c.Id(), loss)
	return loss
}

// createRespawnSaga creates and submits the respawn saga
func (p *ProcessorImpl) createRespawnSaga(ch channel.Model, characterId uint32, targetMapId _map.Id, useWheelOfFortune bool, protectiveItemId *uint32, expLoss uint32) error {
	transactionId := uuid.New()
	now := time.Now()
	steps := make([]saga.Step[any], 0)

	// Step: Consume Wheel of Fortune if used
	if useWheelOfFortune {
		steps = append(steps, saga.Step[any]{
			StepId: "consume_wheel_of_fortune",
			Status: saga.Pending,
			Action: saga.DestroyAsset,
			Payload: saga.DestroyAssetPayload{
				CharacterId: characterId,
				TemplateId:  uint32(item.WheelOfFortuneId),
				Quantity:    1,
				RemoveAll:   false,
			},
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	// Step: Consume protective item if used
	if protectiveItemId != nil {
		steps = append(steps, saga.Step[any]{
			StepId: "consume_protective_item",
			Status: saga.Pending,
			Action: saga.DestroyAsset,
			Payload: saga.DestroyAssetPayload{
				CharacterId: characterId,
				TemplateId:  *protectiveItemId,
				Quantity:    1,
				RemoveAll:   false,
			},
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	// Step: Set HP to 50
	steps = append(steps, saga.Step[any]{
		StepId: "set_hp",
		Status: saga.Pending,
		Action: saga.SetHP,
		Payload: saga.SetHPPayload{
			CharacterId: characterId,
			WorldId:     ch.WorldId(),
			ChannelId:   ch.Id(),
			Amount:      50,
		},
		CreatedAt: now,
		UpdatedAt: now,
	})

	// Step: Deduct experience if applicable
	if expLoss > 0 {
		steps = append(steps, saga.Step[any]{
			StepId: "deduct_experience",
			Status: saga.Pending,
			Action: saga.DeductExperience,
			Payload: saga.DeductExperiencePayload{
				CharacterId: characterId,
				WorldId:     ch.WorldId(),
				ChannelId:   ch.Id(),
				Amount:      expLoss,
			},
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	// Step: Cancel all buffs
	steps = append(steps, saga.Step[any]{
		StepId: "cancel_all_buffs",
		Status: saga.Pending,
		Action: saga.CancelAllBuffs,
		Payload: saga.CancelAllBuffsPayload{
			CharacterId: characterId,
			WorldId:     ch.WorldId(),
			ChannelId:   ch.Id(),
		},
		CreatedAt: now,
		UpdatedAt: now,
	})

	// Step: Warp to target map (spawn point)
	steps = append(steps, saga.Step[any]{
		StepId: "warp_to_spawn",
		Status: saga.Pending,
		Action: saga.WarpToPortal,
		Payload: saga.WarpToPortalPayload{
			CharacterId: characterId,
			WorldId:     ch.WorldId(),
			ChannelId:   ch.Id(),
			MapId:       uint32(targetMapId),
			PortalId:    0, // 0 = spawn point
		},
		CreatedAt: now,
		UpdatedAt: now,
	})

	// Create and submit the saga
	s := saga.Saga{
		TransactionId: transactionId,
		SagaType:      saga.CharacterRespawn,
		InitiatedBy:   "RESPAWN",
		Steps:         steps,
	}

	p.l.Debugf("Creating respawn saga [%s] for character [%d] with [%d] steps.", transactionId, characterId, len(steps))
	return p.sp.Create(s)
}
