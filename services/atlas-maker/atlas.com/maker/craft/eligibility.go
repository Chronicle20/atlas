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

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	skillconst "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// Reason is an eligibility failure code, mapping 1:1 onto PRD §5's stable
// error codes (docs/tasks/task-285-maker-skill-crafting/prd.md §5;
// missing_prerequisite_quest added by design C-5). It is empty when
// Eligible is true.
type Reason string

const (
	ReasonLevelTooLow              Reason = "level_too_low"
	ReasonSkillLevelTooLow         Reason = "skill_level_too_low"
	ReasonInsufficientMaterials    Reason = "insufficient_materials"
	ReasonMissingPrerequisiteItem  Reason = "missing_prerequisite_item"
	ReasonMissingPrerequisiteQuest Reason = "missing_prerequisite_quest"
	ReasonInsufficientMesos        Reason = "insufficient_mesos"
	ReasonInventoryFull            Reason = "inventory_full"
)

// Eligibility is the result of evaluating one recipe for one character
// (design §4.2.2). Reason is populated only when Eligible is false.
type Eligibility struct {
	Eligible bool
	Reason   Reason
}

func eligible() Eligibility {
	return Eligibility{Eligible: true}
}

func ineligible(reason Reason) Eligibility {
	return Eligibility{Eligible: false, Reason: reason}
}

// makerSkills is the four Maker skill variants (libs/atlas-constants/skill),
// the only skill ids design §4.2.2 recognizes as satisfying a recipe's
// reqSkillLevel. A character has at most one, but resolveMakerLevel does not
// assume that.
var makerSkills = []skillconst.Identity{
	skillconst.BeginnerMaker,
	skillconst.NoblesseMaker,
	skillconst.LegendMaker,
	skillconst.EvanMaker,
}

func isMakerSkill(id uint32) bool {
	for _, s := range makerSkills {
		if uint32(s) == id {
			return true
		}
	}
	return false
}

// resolveMakerLevel returns the maximum level across whichever Maker
// variant the character has learned, or 0 if it has none.
func resolveMakerLevel(skills []skill.Model) byte {
	var level byte
	for _, s := range skills {
		if isMakerSkill(s.Id()) && s.Level() > level {
			level = s.Level()
		}
	}
	return level
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
	// ReleaseInFlight clears characterId's in-flight craft guard (design
	// §7). Intended for the saga terminal-event consumer Task 24 wires; a
	// rejected Create already releases its own guard before returning.
	ReleaseInFlight(characterId uint32)
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

func (p *ProcessorImpl) NewSnapshot(characterId uint32) (Snapshot, error) {
	return NewSnapshot(p.kp, characterId)
}

func (p *ProcessorImpl) Evaluate(characterId uint32, snap Snapshot, r recipe.Model) (Eligibility, error) {
	c, err := p.cp.GetById(characterId)
	if err != nil {
		return Eligibility{}, err
	}

	// 1. reqLevel / reqSkillLevel. FR-3.5 floors the required skill level at
	// 1 for every craft, even a recipe whose reqSkillLevel is 0.
	if c.Level() < byte(r.ReqLevel()) {
		return ineligible(ReasonLevelTooLow), nil
	}
	skills, err := p.sp.GetByCharacterId(characterId)
	if err != nil {
		return Eligibility{}, err
	}
	makerLevel := resolveMakerLevel(skills)
	requiredSkillLevel := r.ReqSkillLevel()
	if requiredSkillLevel < 1 {
		requiredSkillLevel = 1
	}
	if uint32(makerLevel) < requiredSkillLevel {
		return ineligible(ReasonSkillLevelTooLow), nil
	}

	// 2. reqItem / reqEquip against the snapshot.
	if r.ReqItem() != 0 && snap.Held(r.ReqItem()) == 0 {
		return ineligible(ReasonMissingPrerequisiteItem), nil
	}
	if r.ReqEquip() != 0 && !snap.Equipped(r.ReqEquip()) {
		return ineligible(ReasonMissingPrerequisiteItem), nil
	}

	// 3. reqQuest against atlas-quests -- only for the few recipes that
	// carry one (C-5); reading quests for every recipe is the cost this
	// avoids.
	if reqs := r.QuestRequirements(); len(reqs) > 0 {
		progress, err := p.qp.GetByCharacterId(characterId)
		if err != nil {
			return Eligibility{}, err
		}
		states := make(map[uint32]byte, len(progress))
		for _, pr := range progress {
			states[pr.QuestId()] = pr.State()
		}
		for _, req := range reqs {
			if uint32(states[req.QuestId]) < req.State {
				return ineligible(ReasonMissingPrerequisiteQuest), nil
			}
		}
	}

	// 4. Every recipe material at its count, summed across slots.
	for _, mat := range r.Materials() {
		if snap.Held(mat.ItemId) < mat.Count {
			return ineligible(ReasonInsufficientMaterials), nil
		}
	}

	// 5. meso vs the character's mesos.
	if c.Meso() < r.Meso() {
		return ineligible(ReasonInsufficientMesos), nil
	}

	// 6. Award accommodation (FR-3.6), computed last since it is the most
	// expensive remaining check and only matters once every other
	// condition already passed.
	accommodated, err := p.kp.CanAccommodate(characterId, awardsOf(r))
	if err != nil {
		return Eligibility{}, err
	}
	if !accommodated {
		return ineligible(ReasonInventoryFull), nil
	}

	return eligible(), nil
}

// awardsOf builds the CanAccommodate request for r's award: the recipe's own
// produced item at itemNum copies when there is no randomReward, or every
// possible randomReward entry (only one of which is actually drawn at craft
// time) so a free slot is confirmed regardless of which is drawn.
func awardsOf(r recipe.Model) []compartment.AccommodationItem {
	rewards := r.RandomRewards()
	if len(rewards) == 0 {
		return []compartment.AccommodationItem{{ItemId: r.Id(), Quantity: r.ItemNum()}}
	}
	items := make([]compartment.AccommodationItem, 0, len(rewards))
	for _, rw := range rewards {
		items = append(items, compartment.AccommodationItem{ItemId: rw.ItemId, Quantity: rw.ItemNum})
	}
	return items
}
