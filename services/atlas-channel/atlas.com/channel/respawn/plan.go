package respawn

import (
	"atlas-channel/asset"
	"atlas-channel/character"
	map_ "atlas-channel/data/map"
	channelInventory "atlas-channel/inventory"
	"atlas-channel/saga"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

// mapFacts is the subset of the map's data the respawn decision needs.
// data/map.Model has no exported constructor, so depending on the three facts
// instead of the model keeps planRespawn testable without adding a
// test-only constructor to the map package.
type mapFacts struct {
	ReturnMapId      _map.Id
	Town             bool
	NoExpLossOnDeath bool
}

func mapFactsOf(m map_.Model) mapFacts {
	return mapFacts{
		ReturnMapId:      m.ReturnMapId(),
		Town:             m.Town(),
		NoExpLossOnDeath: m.NoExpLossOnDeath(),
	}
}

// respawnPlan is the pure outcome of a death: where the character comes back
// and which assets a charge is spent from. Nil asset pointers mean "nothing
// consumed".
type respawnPlan struct {
	TargetMapId _map.Id
	Wheel       *asset.Model
	Protective  *asset.Model
	ExpLoss     uint32
}

// findWheelOfFortune returns the Cash-inventory Wheel of Destiny with at least
// one charge left, or nil. Charges live in the asset's quantity: the client
// gates its own revive dialog on CWvsContext::GetItemCount(5510000) > 0, and
// the client's WZ model for death items (CItemInfo::PROTECTONDIEITEM — nItemID,
// nRecoveryRate) carries no use-count field.
func findWheelOfFortune(inv channelInventory.Model) *asset.Model {
	a, found := inv.Cash().FindFirstByItemId(uint32(item.WheelOfFortuneId))
	if !found || a == nil || a.Quantity() < 1 {
		return nil
	}
	return a
}

// planRespawn decides the respawn outcome. useDeathItem is the client's
// Change.Premium() byte: CUIRevive::OnButtonClicked calls Revive(1) for OK and
// Revive(0) for Cancel, so a zero here means the player declined the wheel and
// their charge must survive.
func planRespawn(c character.Model, inv channelInventory.Model, mf mapFacts, currentMapId _map.Id, useDeathItem bool) respawnPlan {
	p := respawnPlan{TargetMapId: mf.ReturnMapId}

	if useDeathItem {
		if w := findWheelOfFortune(inv); w != nil {
			p.Wheel = w
			p.TargetMapId = currentMapId
		}
	}

	p.Protective = findProtectiveItem(inv)
	p.ExpLoss = calculateExpLoss(c, mf, p.Protective != nil)
	return p
}

// respawnSagaSteps builds the ordered step list for a death. Order is
// load-bearing: both consume steps precede warp_to_spawn, so a failed
// decrement aborts the saga before the character is moved and cannot grant a
// free in-map respawn.
func respawnSagaSteps(f field.Model, characterId uint32, rp respawnPlan, now time.Time) []saga.Step {
	steps := make([]saga.Step, 0)

	if rp.Wheel != nil {
		steps = append(steps, saga.Step{
			StepId: "consume_wheel_of_fortune",
			Status: saga.Pending,
			Action: saga.DestroyAsset,
			Payload: saga.DestroyAssetPayload{
				CharacterId: characterId,
				TemplateId:  rp.Wheel.TemplateId(),
				Quantity:    1,
				RemoveAll:   false,
			},
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	if rp.Protective != nil {
		steps = append(steps, saga.Step{
			StepId: "consume_protective_item",
			Status: saga.Pending,
			Action: saga.DestroyAsset,
			Payload: saga.DestroyAssetPayload{
				CharacterId: characterId,
				TemplateId:  rp.Protective.TemplateId(),
				Quantity:    1,
				RemoveAll:   false,
			},
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	// Step: Set HP to 50
	steps = append(steps, saga.Step{
		StepId: "set_hp",
		Status: saga.Pending,
		Action: saga.SetHP,
		Payload: saga.SetHPPayload{
			CharacterId: characterId,
			WorldId:     f.WorldId(),
			ChannelId:   f.ChannelId(),
			Amount:      50,
		},
		CreatedAt: now,
		UpdatedAt: now,
	})

	// Step: Deduct experience if applicable
	if rp.ExpLoss > 0 {
		steps = append(steps, saga.Step{
			StepId: "deduct_experience",
			Status: saga.Pending,
			Action: saga.DeductExperience,
			Payload: saga.DeductExperiencePayload{
				CharacterId: characterId,
				WorldId:     f.WorldId(),
				ChannelId:   f.ChannelId(),
				Amount:      rp.ExpLoss,
			},
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	// Step: Cancel all buffs
	steps = append(steps, saga.Step{
		StepId: "cancel_all_buffs",
		Status: saga.Pending,
		Action: saga.CancelAllBuffs,
		Payload: saga.CancelAllBuffsPayload{
			CharacterId: characterId,
			WorldId:     f.WorldId(),
			ChannelId:   f.ChannelId(),
		},
		CreatedAt: now,
		UpdatedAt: now,
	})

	// Step: Warp to target map (spawn point)
	steps = append(steps, saga.Step{
		StepId: "warp_to_spawn",
		Status: saga.Pending,
		Action: saga.WarpToPortal,
		Payload: saga.WarpToPortalPayload{
			CharacterId: characterId,
			WorldId:     f.WorldId(),
			ChannelId:   f.ChannelId(),
			MapId:       rp.TargetMapId,
			PortalId:    0, // 0 = spawn point
		},
		CreatedAt: now,
		UpdatedAt: now,
	})

	return steps
}
