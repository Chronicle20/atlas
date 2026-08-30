package craft

import (
	"atlas-maker/compartment"
	"atlas-maker/crystalband"
	"atlas-maker/reagent"
	"atlas-maker/recipe"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	saga "github.com/Chronicle20/atlas/libs/atlas-saga"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Mode is MAKER_SKILL's leading nMode field (design §4.3.1).
type Mode uint32

const (
	ModeCreate            Mode = 1
	ModeCreateWithUpgrade Mode = 2
	ModeMonsterCrystal    Mode = 3
	ModeDisassemble       Mode = 4
)

// DisassembleMesoCharge is mode 4's meso charge (design §4.5.2 "AwardMesos
// for the charge").
//
// No client data backs a magnitude here. Load_MonsterCrystalLevel reads
// only {lvMin, lvMax, itemId} -- reagent-derivation.md §5.7 items 2 and 5
// record explicitly that no count node exists and that a "price" field
// present on the archive's info nodes is never read by that loader, so it
// cannot be attributed maker semantics. crystalband's own Count column
// documents the same gap for crystal quantity ("an Atlas product decision,
// NOT a derived value"). Consistent with that precedent, Task 23 declares
// the derived charge to be 0 pending a real formula, rather than inventing
// one: the AwardMesos step below still exists (so the sequence and a future
// non-zero retuning need no shape change), it simply charges nothing today.
const DisassembleMesoCharge = 0

// Request is atlas-maker's decoded, still-untrusted MAKER_SKILL body
// (design §4.3.1), plus the calling channel's world and channel scope.
// WorldId/ChannelId ride along because atlas-character exposes no live
// channel and AwardMesosPayload needs both to scope its client-visible
// effect; atlas-channel's handler already knows its own session's world and
// channel and is expected to supply them.
type Request struct {
	Mode      Mode
	WorldId   world.Id
	ChannelId channel.Id

	// Mode 1|2 (create / create with upgrade)
	TargetItemId item.Id
	UseCatalyst  bool
	GemItemIds   []item.Id

	// Mode 3 (monster crystal)
	LeftoverItemId item.Id

	// Mode 4 (disassemble)
	EquipItemId item.Id
	SlotPos     int16
}

// SagaEmitter emits a fully-built craft saga for asynchronous execution by
// atlas-saga-orchestrator (design §3.2 step 4). Task 24 supplies the
// concrete Kafka-backed implementation; this package depends only on the
// seam so every rejection path (TestEveryRejectionEmitsNoSaga) can assert
// zero emissions without a broker.
type SagaEmitter interface {
	Emit(s saga.Saga) error
}

// CompensableActions is the closed set of saga actions every craft sequence
// step in §4.5.2 uses (design §7: "every step in every sequence uses a
// compensable action"). DestroyAllAssets is deliberately absent: it
// destroys without recording what to recreate and cannot be compensated
// (libs/atlas-saga/payloads.go's own DestroyAllAssetsPayload doc).
var CompensableActions = map[saga.Action]bool{
	saga.AwardMesos:           true,
	saga.AwardAsset:           true,
	saga.AwardCraftedAsset:    true,
	saga.DestroyAssetFromSlot: true,
}

// Create implements Processor.Create. It takes the in-flight guard first --
// before any validation work -- so a second MAKER_SKILL arriving while the
// first is still resolving is rejected immediately (design §7), then
// dispatches to the mode-specific validate-and-build path. The guard is
// released on every path that returns an error (nothing was emitted, so
// nothing is left to wait on); on success it is left held for the saga's
// terminal event to release via ReleaseInFlight.
func (p *ProcessorImpl) Create(characterId uint32, req Request) (uuid.UUID, error) {
	t := tenant.MustFromContext(p.ctx)
	if !craftGuard.TryAcquire(t.Id(), characterId) {
		return uuid.Nil, ErrCraftInProgress
	}

	txId, err := p.create(characterId, req)
	if err != nil {
		craftGuard.Release(t.Id(), characterId)
		return uuid.Nil, err
	}
	return txId, nil
}

// ReleaseInFlight implements Processor.ReleaseInFlight.
func (p *ProcessorImpl) ReleaseInFlight(characterId uint32) {
	t := tenant.MustFromContext(p.ctx)
	craftGuard.Release(t.Id(), characterId)
}

func (p *ProcessorImpl) create(characterId uint32, req Request) (uuid.UUID, error) {
	switch req.Mode {
	case ModeCreate, ModeCreateWithUpgrade:
		return p.createOrUpgrade(characterId, req)
	case ModeMonsterCrystal:
		return p.crystal(characterId, req)
	case ModeDisassemble:
		return p.disassemble(characterId, req)
	default:
		return uuid.Nil, CraftError{Code: CodeInvalidMode, Status: http.StatusUnprocessableEntity}
	}
}

// createOrUpgrade builds modes 1 and 2's sequence (design §4.5.2): AwardMesos
// (negative, the recipe's meso) -> one DestroyAssetFromSlot per resolved
// slot for each material, gem, and the catalyst -> AwardCraftedAsset for an
// equip output, or plain AwardAsset otherwise.
func (p *ProcessorImpl) createOrUpgrade(characterId uint32, req Request) (uuid.UUID, error) {
	r, err := p.rp.GetById(req.TargetItemId)
	if err != nil {
		if errors.Is(err, recipe.ErrNotFound) {
			return uuid.Nil, ErrRecipeNotFound
		}
		return uuid.Nil, err
	}

	snap, err := p.NewSnapshot(characterId)
	if err != nil {
		return uuid.Nil, err
	}

	elig, err := p.Evaluate(characterId, snap, r)
	if err != nil {
		return uuid.Nil, err
	}
	if !elig.Eligible {
		return uuid.Nil, reasonToCraftError(elig.Reason)
	}

	plan := BuildCreatePlan(snap, r, req.GemItemIds, req.UseCatalyst)

	b := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("MAKER_SKILL")

	b.AddStep("deduct_meso", saga.Pending, saga.AwardMesos, saga.AwardMesosPayload{
		CharacterId: characterId,
		WorldId:     req.WorldId,
		ChannelId:   req.ChannelId,
		ActorType:   "SYSTEM",
		Amount:      -int32(r.Meso()),
		ShowEffect:  true,
	})

	appendDestroySteps(b, characterId, plan)

	invType, isKnown := inventory.TypeFromItemId(r.Id())
	if isKnown && invType == inventory.TypeValueEquip {
		stats := reagentStats(p.rgp, AppliedGems(snap, req.GemItemIds))
		stats.CharacterId = characterId
		stats.TemplateId = uint32(r.Id())
		stats.Quantity = 1
		stats.Slots = uint16(r.Tuc())
		stats.ShowEffect = true
		b.AddStep("award_item", saga.Pending, saga.AwardCraftedAsset, stats)
	} else {
		b.AddStep("award_item", saga.Pending, saga.AwardAsset, saga.AwardItemActionPayload{
			CharacterId: characterId,
			Item:        saga.ItemPayload{TemplateId: uint32(r.Id()), Quantity: r.ItemNum()},
			ShowEffect:  true,
		})
	}

	return p.emit(b, logrus.Fields{
		"characterId":  characterId,
		"mode":         req.Mode,
		"recipeId":     r.Id(),
		"mesoDelta":    -int32(r.Meso()),
		"producedItem": r.Id(),
	})
}

// crystal builds mode 3's sequence (design §4.5.2): AwardMesos (negative,
// recipe meso) -> DestroyAssetFromSlot for the leftover -> AwardAsset for
// the weighted randomReward draw.
func (p *ProcessorImpl) crystal(characterId uint32, req Request) (uuid.UUID, error) {
	r, err := p.rp.GetByLeftover(req.LeftoverItemId)
	if err != nil {
		if errors.Is(err, recipe.ErrNoCrystalMapping) {
			return uuid.Nil, CraftError{Code: CodeNoCrystalMapping, Status: http.StatusUnprocessableEntity}
		}
		return uuid.Nil, err
	}

	snap, err := p.NewSnapshot(characterId)
	if err != nil {
		return uuid.Nil, err
	}

	elig, err := p.Evaluate(characterId, snap, r)
	if err != nil {
		return uuid.Nil, err
	}
	if !elig.Eligible {
		return uuid.Nil, reasonToCraftError(elig.Reason)
	}

	// OQ-7 extension: eligibility's generic material check above only
	// proved the archive's group-0 `count` (1) is held. The actual
	// consumption is LeftoverConsumeQuantity (100, see plan.go), so a
	// character holding fewer than 100 must still be rejected here -- an
	// "eligible" craft that then destroys less than 100 would reproduce
	// exactly the client-log/inventory mismatch OQ-7 exists to prevent.
	if snap.Held(req.LeftoverItemId) < LeftoverConsumeQuantity {
		return uuid.Nil, CraftError{Code: CodeInsufficientMaterials, Status: http.StatusUnprocessableEntity}
	}

	reward, err := Draw(r.RandomRewards())
	if err != nil {
		return uuid.Nil, err
	}

	plan := BuildCrystalPlan(snap, req.LeftoverItemId)

	b := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("MAKER_SKILL")

	b.AddStep("deduct_meso", saga.Pending, saga.AwardMesos, saga.AwardMesosPayload{
		CharacterId: characterId,
		WorldId:     req.WorldId,
		ChannelId:   req.ChannelId,
		ActorType:   "SYSTEM",
		Amount:      -int32(r.Meso()),
		ShowEffect:  true,
	})

	appendDestroySteps(b, characterId, plan)

	b.AddStep("award_crystal", saga.Pending, saga.AwardAsset, saga.AwardItemActionPayload{
		CharacterId: characterId,
		Item:        saga.ItemPayload{TemplateId: uint32(reward.ItemId), Quantity: reward.ItemNum},
		ShowEffect:  true,
	})

	return p.emit(b, logrus.Fields{
		"characterId":  characterId,
		"mode":         req.Mode,
		"recipeId":     r.Id(),
		"mesoDelta":    -int32(r.Meso()),
		"producedItem": reward.ItemId,
	})
}

// disassemble builds mode 4's sequence (design §4.5.2): verify the claimed
// equip is genuinely in the named EQUIP slot, then DestroyAssetFromSlot ->
// AwardAsset per derived crystal -> AwardMesos for the charge. There is no
// recipe.Model for this mode, so it validates independently of
// Processor.Evaluate rather than routing through it.
func (p *ProcessorImpl) disassemble(characterId uint32, req Request) (uuid.UUID, error) {
	snap, err := p.NewSnapshot(characterId)
	if err != nil {
		return uuid.Nil, err
	}

	// FR-3.5: every craft operation is gated on a Maker skill variant at
	// level >= 1, even disassembly.
	skills, err := p.sp.GetByCharacterId(characterId)
	if err != nil {
		return uuid.Nil, err
	}
	if resolveMakerLevel(skills) < 1 {
		return uuid.Nil, CraftError{Code: CodeSkillLevelTooLow, Status: http.StatusUnprocessableEntity}
	}

	// Never trust the client's slot/id pair: verify the claimed equip is
	// genuinely at that slot before touching anything.
	held := false
	for _, sh := range snap.Slots(req.EquipItemId) {
		if sh.Slot == req.SlotPos {
			held = true
			break
		}
	}
	if !held {
		return uuid.Nil, CraftError{Code: CodeEquipNotFound, Status: http.StatusUnprocessableEntity}
	}

	eq, err := p.eqp.GetById(req.EquipItemId)
	if err != nil {
		return uuid.Nil, err
	}

	crystalId, count, err := p.cbp.CrystalForLevel(eq.ReqLevel())
	if err != nil {
		if errors.Is(err, crystalband.ErrNotFound) {
			return uuid.Nil, CraftError{Code: CodeNoCrystalMapping, Status: http.StatusUnprocessableEntity}
		}
		return uuid.Nil, err
	}

	// FR-3.6: reject before any mutation when the crystal award has no free
	// slot.
	accommodated, err := p.kp.CanAccommodate(characterId, []compartment.AccommodationItem{{ItemId: crystalId, Quantity: count}})
	if err != nil {
		return uuid.Nil, err
	}
	if !accommodated {
		return uuid.Nil, CraftError{Code: CodeInventoryFull, Status: http.StatusUnprocessableEntity}
	}

	invType, _ := inventory.TypeFromItemId(req.EquipItemId)

	b := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("MAKER_SKILL")

	b.AddStep("destroy_equip", saga.Pending, saga.DestroyAssetFromSlot, saga.DestroyAssetFromSlotPayload{
		CharacterId:   characterId,
		InventoryType: byte(invType),
		Slot:          req.SlotPos,
		Quantity:      1,
		TemplateId:    uint32(req.EquipItemId),
	})

	b.AddStep("award_crystal", saga.Pending, saga.AwardAsset, saga.AwardItemActionPayload{
		CharacterId: characterId,
		Item:        saga.ItemPayload{TemplateId: uint32(crystalId), Quantity: count},
		ShowEffect:  true,
	})

	b.AddStep("charge_meso", saga.Pending, saga.AwardMesos, saga.AwardMesosPayload{
		CharacterId: characterId,
		WorldId:     req.WorldId,
		ChannelId:   req.ChannelId,
		ActorType:   "SYSTEM",
		Amount:      -int32(DisassembleMesoCharge),
		ShowEffect:  true,
	})

	return p.emit(b, logrus.Fields{
		"characterId":  characterId,
		"mode":         req.Mode,
		"disassembled": req.EquipItemId,
		"mesoDelta":    -int32(DisassembleMesoCharge),
		"producedItem": crystalId,
	})
}

// appendDestroySteps adds one DestroyAssetFromSlot step per plan entry, in
// plan order (already ascending-slot from resolveConsumption).
func appendDestroySteps(b *saga.Builder, characterId uint32, plan Plan) {
	for i, c := range plan.Consumptions {
		b.AddStep(fmt.Sprintf("destroy_%d", i), saga.Pending, saga.DestroyAssetFromSlot, saga.DestroyAssetFromSlotPayload{
			CharacterId:   characterId,
			InventoryType: byte(c.InventoryType),
			Slot:          c.Slot,
			Quantity:      c.Quantity,
			TemplateId:    uint32(c.TemplateId),
		})
	}
}

// addStat sums an AwardCraftedAssetPayload field's uint16 with a reagent's
// signed delta, clamping at 0 rather than underflowing.
func addStat(base uint16, delta int16) uint16 {
	v := int32(base) + int32(delta)
	if v < 0 {
		return 0
	}
	return uint16(v)
}

// reagentStats sums every gemIds entry's reagent.Model.Stat() contribution
// into an AwardCraftedAssetPayload's explicit stat block (FR-3.1, FR-3.2).
// A gem id reagent.Processor has no row for (not a gem, or genuinely
// unseeded) is dropped rather than failing the craft (reagent.ErrNotFound's
// own doc: "an unknown reagent is dropped, a failed read is not").
// incReqLevel, randOption, and randStat have no corresponding
// AwardCraftedAssetPayload field (reagent/builder.go: "applying them is the
// caller's concern, not this package's") and are intentionally not applied
// here.
func reagentStats(rgp reagent.Processor, gemIds []item.Id) saga.AwardCraftedAssetPayload {
	var out saga.AwardCraftedAssetPayload
	for _, id := range gemIds {
		m, err := rgp.GetByItemId(id)
		if err != nil {
			continue
		}
		v := m.Value()
		switch m.Stat() {
		case "incPAD":
			out.WeaponAttack = addStat(out.WeaponAttack, v)
		case "incMAD":
			out.MagicAttack = addStat(out.MagicAttack, v)
		case "incACC":
			out.Accuracy = addStat(out.Accuracy, v)
		case "incEVA":
			out.Avoidability = addStat(out.Avoidability, v)
		case "incSpeed":
			out.Speed = addStat(out.Speed, v)
		case "incJump":
			out.Jump = addStat(out.Jump, v)
		case "incMaxHP":
			out.HP = addStat(out.HP, v)
		case "incMaxMP":
			out.MP = addStat(out.MP, v)
		case "incSTR":
			out.Strength = addStat(out.Strength, v)
		case "incINT":
			out.Intelligence = addStat(out.Intelligence, v)
		case "incLUK":
			out.Luck = addStat(out.Luck, v)
		case "incDEX":
			out.Dexterity = addStat(out.Dexterity, v)
		}
	}
	return out
}

// emit builds b, submits it through the Emitter, and logs the PRD §8
// observability line on acceptance -- tenant, character, mode, recipe id,
// materials consumed, meso delta, and produced item, correlated by saga id.
func (p *ProcessorImpl) emit(b *saga.Builder, fields logrus.Fields) (uuid.UUID, error) {
	s := b.Build()
	if err := p.em.Emit(s); err != nil {
		return uuid.Nil, err
	}
	t := tenant.MustFromContext(p.ctx)
	p.l.WithFields(fields).
		WithField("tenantId", t.Id()).
		WithField("transactionId", s.TransactionId).
		Info("Accepted maker craft; craft saga emitted.")
	return s.TransactionId, nil
}
