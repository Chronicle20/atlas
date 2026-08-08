package handler

import (
	"atlas-channel/character"
	"atlas-channel/compartment"
	"atlas-channel/consumable"
	consumabledata "atlas-channel/data/consumable"
	equipmentdata "atlas-channel/data/equipment"
	"atlas-channel/pet"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	character2 "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	inventory2 "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	petskill "github.com/Chronicle20/atlas/libs/atlas-constants/pet/skill"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	pet2 "github.com/Chronicle20/atlas/libs/atlas-packet/pet/serverbound"
	statpkt "github.com/Chronicle20/atlas/libs/atlas-packet/stat/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-rest/degrade"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// skillGate option values (tenant handler config, design §3.2): the FR-3 gate
// mirrors exactly the gate the tenant's client family enforces.
const (
	skillGateEquipAbility = "equipAbility" // GMS: worn pet-ability equip
	skillGatePetSkillFlag = "petSkillFlag" // JMS: pouch-taught pet flag
)

func PetItemUseHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := pet2.ItemUse{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		// p.BuffSkill() is a damage-mitigation code from CUserLocal::SetDamaged
		// (0=normal, 1=Power Guard, 2=Meso Guard, 4/8=other mitigation arms), not
		// a skill id, and is decoded as a bool by the current codec even though
		// the wire byte is wider — log it as the bool it is; never gate on it.
		reject := func(reason string) {
			l.WithFields(logrus.Fields{
				"characterId": s.CharacterId(),
				"petId":       p.PetId(),
				"itemId":      p.ItemId(),
				"slot":        p.Source(),
				"buffSkill":   p.BuffSkill(),
				"reason":      reason,
			}).Warnf("Rejecting pet auto-pot request from character [%d]: [%s].", s.CharacterId(), reason)
			if err := enableActions(l)(ctx)(wp)(s); err != nil {
				l.WithError(err).Errorf("Unable to write [%s] for character [%d].", statpkt.StatChangedWriter, s.CharacterId())
			}
		}

		gate, _ := readerOptions["skillGate"].(string)
		if gate != skillGateEquipAbility && gate != skillGatePetSkillFlag {
			// Fail closed and loud: a template gap must never be permissive.
			reject("skill_gate_unconfigured")
			return
		}

		// gms_48 has no petId on the wire: the lookup branches on whether this
		// version carries one at all, not merely on whether the decoded value is
		// non-zero. petId present -> resolve that pet directly (narrowing-guarded
		// below by evaluateAutoPot's ownership/spawn checks). Field absent ->
		// resolve the character's spawned pet. Field present but zero -> reject.
		// Never fall back from one to the other — that would reopen the FR-1
		// ownership hole this task exists to close.
		//
		// The wire value is the pet's CLIENT serial (GW_ItemSlotBase::liCashItemSN
		// = the cash serial for a purchased pet), not the Atlas pet id, so it is a
		// full 64 bits. It used to be range-checked against MaxUint32 on the theory
		// that the two were the same number; that check rejected every
		// cash-purchased pet outright.
		hasPetId, reason, ok := classifyPetIdInput(pet2.HasLeadingPetId(tenant.MustFromContext(ctx)), p.PetId())
		if !ok {
			reject(reason)
			return
		}

		// Parallel fetch — one round-trip of latency, not three (design §3.1).
		// model.Future carries no per-future error (Group.Wait returns only the
		// first), so each provider captures its own error and never fails the
		// group; Wait() is the happens-before barrier for the captured values.
		pg, _ := model.NewGroup(ctx)
		var (
			pm          pet.Model
			pmErr       error
			spawnedPets []pet.Model
			spawnedErr  error
			c           character.Model
			cErr        error
			ci          consumabledata.Model
			ciErr       error
		)
		if hasPetId {
			model.Submit(pg, func() (any, error) {
				pm, pmErr = pet.NewProcessor(l, ctx).GetBySerialNumber(s.CharacterId(), p.PetId())
				return nil, nil
			})
		} else {
			model.Submit(pg, func() (any, error) {
				spawnedPets, spawnedErr = pet.NewProcessor(l, ctx).GetByOwner(s.CharacterId())
				return nil, nil
			})
		}
		model.Submit(pg, func() (any, error) {
			c, cErr = character.NewProcessor(l, ctx).GetById()(s.CharacterId())
			return nil, nil
		})
		model.Submit(pg, func() (any, error) {
			ci, ciErr = consumabledata.NewProcessor(l, ctx).GetById(p.ItemId())
			return nil, nil
		})
		_ = pg.Wait()

		// Fail closed on any fetch failure (design §5) — never forward unvalidated.
		if hasPetId {
			if pmErr != nil {
				reject("pet_not_found")
				return
			}
		} else {
			if spawnedErr != nil {
				reject("pet_not_found")
				return
			}
			pm, reason, ok = resolveSpawnedPet(spawnedPets)
			if !ok {
				reject(reason)
				return
			}
		}
		if ciErr != nil {
			reject("not_consumable")
			return
		}
		if cErr != nil {
			l.WithError(cErr).Warnf("Unable to resolve character [%d] during pet auto-pot validation.", s.CharacterId())
			reject("fetch_failed")
			return
		}

		recoversHP, recoversMP := classifyRecovery(ci)

		hasHP, hasMP, ok := resolveSkillSources(l, ctx)(gate, s.CharacterId(), pm)
		if !ok {
			reject("equip_data_missing")
			return
		}

		if reason, pass := evaluateAutoPot(s.CharacterId(), c.Hp(), pm.OwnerId(), pm.Slot(), recoversHP, recoversMP, hasHP, hasMP); !pass {
			reject(reason)
			return
		}

		_ = consumable.NewProcessor(l, ctx).RequestItemConsume(s.Field(), character2.Id(s.CharacterId()), item.Id(p.ItemId()), slot.Position(p.Source()), 1, p.UpdateTime())
	}
}

// classifyPetIdInput decides how the target pet is resolved, given whether the
// tenant's client version puts a petId on the wire at all
// (pet2.HasLeadingPetId) and what the decoder produced. Returns
// (usePetId, reason, ok); ok=false means reject with reason.
//
// The three cases are deliberately distinct, because testing `petId != 0`
// alone conflates two of them:
//   - version has no petId field (gms_48): resolve the character's spawned
//     pet. Any value the decoder left in petId is meaningless here.
//   - version has the field and the client sent one: resolve that pet by id.
//   - version has the field and the client sent 0: a real Atlas pet id is
//     never 0, so this is malformed or forged. Reject. Falling through to the
//     spawned-pet branch here is what would breach the FR-1 invariant "never
//     fall back from one resolution path to the other".
func classifyPetIdInput(hasWirePetId bool, petId uint64) (bool, string, bool) {
	if !hasWirePetId {
		return false, "", true
	}
	if petId == 0 {
		return false, "pet_not_found", false
	}
	return true, "", true
}

// resolveSpawnedPet implements the gms_48 (petId-absent) branch of pet
// resolution: the client has a single active pet slot, so "the spawned pet"
// is unambiguous. Returns pet_not_found when the character has no currently
// spawned pet. This is only ever called when the wire petId was absent
// (petId == 0) — it is never a fallback from the petId-bearing GetById path.
func resolveSpawnedPet(pets []pet.Model) (pet.Model, string, bool) {
	for _, pm := range pets {
		if pm.Slot() >= 0 {
			return pm, "", true
		}
	}
	return pet.Model{}, "pet_not_found", false
}

// classifyRecovery reports whether the consumed item's spec recovers HP and/or
// MP. HP vs MP intent is not on the wire (TryConsumePetHP/MP encode identical
// packets) — the server derives it from the item, and for dual items either
// matching skill source passes.
func classifyRecovery(ci consumabledata.Model) (bool, bool) {
	hp := false
	if v, ok := ci.GetSpec(consumabledata.SpecTypeHP); ok && v > 0 {
		hp = true
	}
	if v, ok := ci.GetSpec(consumabledata.SpecTypeHPRecovery); ok && v > 0 {
		hp = true
	}
	mp := false
	if v, ok := ci.GetSpec(consumabledata.SpecTypeMP); ok && v > 0 {
		mp = true
	}
	if v, ok := ci.GetSpec(consumabledata.SpecTypeMPRecovery); ok && v > 0 {
		mp = true
	}
	return hp, mp
}

// evaluateAutoPot runs the FR-1/FR-2/FR-3 decision on already-resolved inputs.
// Ordered cheapest-first; all failures are externally identical (unstick+warn).
func evaluateAutoPot(characterId uint32, characterHp uint16, petOwnerId uint32, petSlot int8, recoversHP, recoversMP, hasHPSource, hasMPSource bool) (string, bool) {
	if petOwnerId != characterId {
		return "pet_not_owned", false
	}
	if petSlot < 0 {
		return "pet_not_spawned", false
	}
	if characterHp == 0 {
		return "character_dead", false
	}
	if !recoversHP && !recoversMP {
		return "not_consumable", false
	}
	if recoversHP && hasHPSource {
		return "", true
	}
	if recoversMP && hasMPSource {
		return "", true
	}
	return "missing_pet_skill", false
}

// resolveSkillSources returns (hasHPSource, hasMPSource, ok). ok=false means
// the equip gate found worn candidates but no ability data at all — the
// deploy-ordering signal (atlas-data not re-ingested), distinct from a plain
// missing skill so operators can tell the two apart in logs.
func resolveSkillSources(l logrus.FieldLogger, ctx context.Context) func(gate string, characterId uint32, pm pet.Model) (bool, bool, bool) {
	return func(gate string, characterId uint32, pm pet.Model) (bool, bool, bool) {
		if gate == skillGatePetSkillFlag {
			return petskill.Has(pm.Flag(), petskill.ConsumeHP), petskill.Has(pm.Flag(), petskill.ConsumeMP), true
		}

		cm, err := compartment.NewProcessor(l, ctx).GetByType(characterId, inventory2.TypeValueEquip)
		if err != nil {
			degrade.Observe(l, "channel.pet_auto_pot.equip_compartment", characterId, err)
			return false, false, true // no worn equips resolvable -> plain missing_pet_skill
		}
		positions := petAbilityPositions(pm.Slot())
		var worn []wornEquip
		ep := equipmentdata.NewProcessor(l, ctx)
		for _, a := range cm.Assets() {
			pos := normalizeWornPosition(slot.Position(a.Slot()))
			if !positions[pos] {
				continue
			}
			em, err := ep.GetById(a.TemplateId())
			if err != nil {
				degrade.Observe(l.WithFields(logrus.Fields{"characterId": characterId, "slot": a.Slot()}), "channel.pet_auto_pot.equip_data", a.TemplateId(), err)
				worn = append(worn, wornEquip{position: slot.Position(a.Slot())})
				continue
			}
			worn = append(worn, wornEquip{position: slot.Position(a.Slot()), abilities: em.PetAbilities()})
		}
		if len(worn) == 0 {
			return false, false, true
		}
		hasHP, hasMP, sawData := matchPetAbilityEquips(worn, pm.Slot())
		return hasHP, hasMP, sawData
	}
}

type wornEquip struct {
	position  slot.Position
	abilities []string
}

// petAbilityPositions mirrors the client's CPet::UpdatePetAbility slot list:
// petHP(-24)/petMP(-25) apply to every pet; each pet index additionally has
// its own ability range (pet 0: -21..-29,-46; pet 1: -31..-37,-47;
// pet 2: -39..-45,-48). Positions are the canonical (non-cash-offset) values
// from libs/atlas-constants/inventory/slot.
func petAbilityPositions(petSlot int8) map[slot.Position]bool {
	res := map[slot.Position]bool{-24: true, -25: true}
	var lo, hi, ignore slot.Position
	switch petSlot {
	case 0:
		lo, hi, ignore = -29, -21, -46
	case 1:
		lo, hi, ignore = -37, -31, -47
	case 2:
		lo, hi, ignore = -45, -39, -48
	default:
		return res
	}
	for p := lo; p <= hi; p++ {
		res[p] = true
	}
	res[ignore] = true
	return res
}

// normalizeWornPosition maps the raw equip-compartment position to the
// canonical slot position: worn cash equips are stored at position-100
// (see character.Model.SetInventory), and pet equips are cash items.
func normalizeWornPosition(p slot.Position) slot.Position {
	if p < -100 {
		return p + 100
	}
	return p
}

// matchPetAbilityEquips reports (hasHP, hasMP, sawData) across the worn
// candidates already filtered to the pet's ability positions. sawData=false
// when no candidate carried any ability attributes (missing equip data).
func matchPetAbilityEquips(worn []wornEquip, petSlot int8) (bool, bool, bool) {
	positions := petAbilityPositions(petSlot)
	hasHP, hasMP, sawData := false, false, false
	for _, w := range worn {
		if !positions[normalizeWornPosition(w.position)] {
			continue
		}
		if len(w.abilities) > 0 {
			sawData = true
		}
		for _, ab := range w.abilities {
			if ab == string(petskill.ConsumeHP) {
				hasHP = true
			}
			if ab == string(petskill.ConsumeMP) {
				hasMP = true
			}
		}
	}
	return hasHP, hasMP, sawData
}
