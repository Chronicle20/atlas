package craft

import (
	"atlas-maker/character"
	"atlas-maker/compartment"
	"atlas-maker/crystalband"
	"atlas-maker/data/equipment"
	"atlas-maker/quest"
	"atlas-maker/reagent"
	"atlas-maker/recipe"
	"atlas-maker/skill"
	"context"
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

// Processor evaluates per-character recipe eligibility (FR-2.1, FR-2.2,
// FR-3.5) and, per Task 23, validates and executes a craft. Create lives
// here rather than on a separately named type because Task 21 already
// claimed the package's obvious "the craft processor" name for eligibility;
// splitting validation-only and validate-and-execute processors would
// fragment one cohesive dependency set (character/skill/compartment/quest
// plus, as of Task 23, recipe/reagent/crystalband/equipment) across two
// types for no benefit to a caller, who always wants both.
type Processor interface {
	// NewSnapshot reads characterId's EQUIP, USE, and ETC compartments once
	// each, for repeated in-memory evaluation of every candidate recipe
	// (design §4.2.2 NFR).
	NewSnapshot(characterId uint32) (Snapshot, error)
	// Evaluate checks r against characterId and snap, in the cheapest-first
	// order design §4.2.2 specifies, returning on the first failed check.
	Evaluate(characterId uint32, snap Snapshot, r recipe.Model) (Eligibility, error)
	// Create validates req against characterId's live state and, on
	// acceptance, emits the craft saga and returns its transaction id
	// (design §3.2 steps 2-4). On rejection it returns a *CraftError whose
	// Code maps 1:1 onto PRD §5 and mutates nothing (design §7 "rejection is
	// pre-mutation").
	Create(characterId uint32, req Request) (uuid.UUID, error)
	// ReleaseInFlight clears the in-flight craft guard entry Track-ed under
	// transactionId (design §7). Called by kafka/consumer/saga's terminal
	// event handler on both COMPLETED and FAILED -- the only handle a
	// terminal event carries is the transaction id, never a character id.
	// A rejected Create already releases its own guard before returning.
	ReleaseInFlight(transactionId uuid.UUID)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	cp  character.Processor
	sp  skill.Processor
	kp  compartment.Processor
	qp  quest.Processor
	rp  recipe.Processor
	rgp reagent.Processor
	cbp crystalband.Processor
	eqp equipment.Processor
	em  SagaEmitter
}

// NewProcessor builds a Processor backed by the upstream character, skill,
// compartment, and quest processors (Task 19), the recipe cache (Task 20),
// the reagent stat table and crystal-band table (Task 17/18), the equipment
// data client (Task 19), and em, the caller-supplied saga emission seam
// (Task 23; the concrete Kafka-backed implementation is Task 24's to wire).
func NewProcessor(l logrus.FieldLogger, ctx context.Context, cp character.Processor, sp skill.Processor, kp compartment.Processor, qp quest.Processor, rp recipe.Processor, rgp reagent.Processor, cbp crystalband.Processor, eqp equipment.Processor, em SagaEmitter) Processor {
	return &ProcessorImpl{l: l, ctx: ctx, cp: cp, sp: sp, kp: kp, qp: qp, rp: rp, rgp: rgp, cbp: cbp, eqp: eqp, em: em}
}

var _ Processor = (*ProcessorImpl)(nil)

// NewSnapshot implements Processor.NewSnapshot.
func (p *ProcessorImpl) NewSnapshot(characterId uint32) (Snapshot, error) {
	return NewSnapshot(p.kp, characterId)
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
// released on every path that returns an error, including one that already
// reached p.emit: p.emit itself Tracks the saga's transaction id against
// this entry before submitting it, and unwinds that Track back off again if
// the submit fails, so a repeat Release here after such a failure is a
// harmless no-op (Release on an unheld key is defined as one). On success
// the entry stays held, already Tracked, for kafka/consumer/saga's
// terminal-event handler to release via ReleaseInFlightByTransaction.
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
func (p *ProcessorImpl) ReleaseInFlight(transactionId uuid.UUID) {
	t := tenant.MustFromContext(p.ctx)
	ReleaseInFlightByTransaction(t.Id(), transactionId)
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

	b.AddStep("record_manifest", saga.Pending, saga.RecordCraftManifest, craftManifest(characterId, req.Mode, r, plan, req.GemItemIds, snap))

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

	return p.emit(characterId, b, logrus.Fields{
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

	b.AddStep("record_manifest", saga.Pending, saga.RecordCraftManifest, saga.CraftManifestPayload{
		CharacterId:    characterId,
		Mode:           uint32(req.Mode),
		CrystalItemId:  uint32(reward.ItemId),
		LeftoverItemId: uint32(req.LeftoverItemId),
		MesoCost:       r.Meso(),
	})

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

	return p.emit(characterId, b, logrus.Fields{
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

	b.AddStep("record_manifest", saga.Pending, saga.RecordCraftManifest, saga.CraftManifestPayload{
		CharacterId:        characterId,
		Mode:               uint32(req.Mode),
		DisassembledItemId: uint32(req.EquipItemId),
		Crystals:           []saga.CraftManifestItem{{ItemId: uint32(crystalId), Count: count}},
		MesoCost:           uint32(DisassembleMesoCharge),
	})

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

	return p.emit(characterId, b, logrus.Fields{
		"characterId":  characterId,
		"mode":         req.Mode,
		"disassembled": req.EquipItemId,
		"mesoDelta":    -int32(DisassembleMesoCharge),
		"producedItem": crystalId,
	})
}

// craftManifest builds modes 1/2's CraftManifestPayload from the resolved
// Plan and recipe -- never from Request, since BuildCreatePlan silently
// drops an unheld gem and an unheld catalyst (FR-3.2); a Request-derived
// manifest would report consumption that never happened (task-285
// Task 26a). Materials aggregate by template id (craftManifestMaterials);
// GemItemIds is the applied, hold-filtered set (AppliedGems, F5); a catalyst
// is reported only when the plan actually holds a RoleCatalyst consumption.
func craftManifest(characterId uint32, mode Mode, r recipe.Model, plan Plan, gemItemIds []item.Id, snap Snapshot) saga.CraftManifestPayload {
	applied := AppliedGems(snap, gemItemIds)
	gemIds := make([]uint32, len(applied))
	for i, id := range applied {
		gemIds[i] = uint32(id)
	}
	catalystUsed, catalystItemId := craftManifestCatalyst(plan)

	return saga.CraftManifestPayload{
		CharacterId:    characterId,
		Mode:           uint32(mode),
		TargetItemId:   uint32(r.Id()),
		ItemNum:        r.ItemNum(),
		Materials:      craftManifestMaterials(plan),
		GemItemIds:     gemIds,
		CatalystUsed:   catalystUsed,
		CatalystItemId: catalystItemId,
		MesoCost:       r.Meso(),
	}
}

// craftManifestMaterials aggregates plan's RoleMaterial consumptions by
// template id: resolveConsumption emits one Consumption per slot touched,
// but the wire renders one line per material id, not per slot
// (manifest-carrier-derivation.md §1, §2 F4).
func craftManifestMaterials(plan Plan) []saga.CraftManifestItem {
	var order []item.Id
	totals := make(map[item.Id]uint32, len(plan.Consumptions))
	for _, c := range plan.Consumptions {
		if c.Role != RoleMaterial {
			continue
		}
		if _, seen := totals[c.TemplateId]; !seen {
			order = append(order, c.TemplateId)
		}
		totals[c.TemplateId] += c.Quantity
	}
	if len(order) == 0 {
		return nil
	}
	out := make([]saga.CraftManifestItem, 0, len(order))
	for _, id := range order {
		out = append(out, saga.CraftManifestItem{ItemId: uint32(id), Count: totals[id]})
	}
	return out
}

// craftManifestCatalyst reports whether plan holds a RoleCatalyst
// consumption and, if so, its template id -- true only when a catalyst
// consumption is actually in the plan, matching the fact that
// BuildCreatePlan resolves no catalyst consumption when it is unheld.
func craftManifestCatalyst(plan Plan) (bool, uint32) {
	for _, c := range plan.Consumptions {
		if c.Role == RoleCatalyst {
			return true, uint32(c.TemplateId)
		}
	}
	return false, 0
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

// emit builds b, Tracks its transaction id against characterId's held
// craftGuard entry, and only then submits it through the Emitter. Tracking
// before the produce (rather than after, as an earlier revision did) closes
// a release-before-track race: NewBuilder assigns TransactionId at Build
// time (libs/atlas-saga/builder.go), so the id already exists here and a
// synchronous, broker-acknowledged Emit can otherwise let this same service
// consume the saga's terminal event -- and call ReleaseByTransactionId as a
// no-op, because nothing was Tracked yet -- before the emitting goroutine
// ever reaches Track, leaving a mapping nothing will ever release. Tracking
// first means the terminal-event consumer always finds a mapping to
// release, however fast it runs.
//
// If Emit then fails, the craft was never accepted and no terminal event
// will ever arrive to release the entry via ReleaseByTransactionId, so this
// unwinds by releasing the whole (tenant, character) entry directly --
// Release also drops the just-added byTransaction index, leaving no stale
// mapping behind. Create's own error path additionally calls Release, but
// that is a harmless no-op by then (Release on an unheld key is defined as
// a no-op) and still required, since not every error returned by
// p.create() (see p.emit's own callers) reaches this method's Track/Emit at
// all.
func (p *ProcessorImpl) emit(characterId uint32, b *saga.Builder, fields logrus.Fields) (uuid.UUID, error) {
	s := b.Build()
	t := tenant.MustFromContext(p.ctx)
	craftGuard.Track(t.Id(), characterId, s.TransactionId)
	if err := p.em.Emit(s); err != nil {
		craftGuard.Release(t.Id(), characterId)
		return uuid.Nil, err
	}
	p.l.WithFields(fields).
		WithField("tenantId", t.Id()).
		WithField("transactionId", s.TransactionId).
		Info("Accepted maker craft; craft saga emitted.")
	return s.TransactionId, nil
}
