package factory

import (
	"atlas-character-factory/configuration"
	"atlas-character-factory/configuration/tenant/characters/preset"
	"atlas-character-factory/configuration/tenant/maplelife"
	"atlas-character-factory/data"
	"atlas-character-factory/saga"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// MapleLifeCreateRestModel is the player's own submission from the Maple
// Life dialog (design.md §11 A5): the class they picked (by ordinal, not
// job id -- the dialog itself never shows a job id), their look choices, and
// the level (0..10) they chose for the class's SP skill.
type MapleLifeCreateRestModel struct {
	AccountId    uint32 `json:"accountId"`
	WorldId      byte   `json:"worldId"`
	Name         string `json:"name"`
	ClassOrdinal uint32 `json:"classOrdinal"`
	Gender       byte   `json:"gender"`
	Face         uint32 `json:"face"`
	Hair         uint32 `json:"hair"`
	HairColor    uint32 `json:"hairColor"`
	SkinColor    byte   `json:"skinColor"`
	SP           byte   `json:"sp"`
}

var (
	ErrMapleLifeNotConfigured = errors.New("maple life not configured for tenant")
	ErrClassOrdinalUnknown    = errors.New("maple life class ordinal not configured")
	ErrLookInvalid            = errors.New("maple life look selection invalid")
	ErrSPInvalid              = errors.New("maple life sp selection invalid")
)

// CreateMapleLife resolves the tenant's Maple Life class table, validates the
// player's look and SP choices against it, and builds the same
// CharacterCreation saga CreateFromPreset does -- by projecting the resolved
// class plus the player's choices onto a preset.RestModel and handing it to
// buildPresetCharacterCreationSaga. The eleven characters.Templates rules
// (findCreationTemplate and friends) never apply to this path: Maple Life's
// own class table is the only source of truth (design.md §11 A5).
func (p *ProcessorImpl) CreateMapleLife(ctx context.Context, in MapleLifeCreateRestModel) (string, error) {
	pr, skillsById, err := p.resolveMapleLifePreset(ctx, in)
	if err != nil {
		return "", err
	}

	presetIn := PresetCreateRestModel{
		AccountId: in.AccountId,
		WorldId:   in.WorldId,
		Name:      in.Name,
	}

	transactionId := uuid.New()
	sg := buildPresetCharacterCreationSaga(transactionId, presetIn, pr, skillsById)
	if err := saga.NewProcessor(p.l, ctx).Create(sg); err != nil {
		p.l.WithError(err).Errorf("Unable to emit maple life character creation saga for character [%s].", in.Name)
		return "", err
	}
	return transactionId.String(), nil
}

// resolveMapleLifePreset runs every validation step against the tenant's
// Maple Life configuration and projects the result onto a preset.RestModel,
// stopping short of building or emitting the saga. Split out from
// CreateMapleLife so the conversion is directly testable against
// buildPresetCharacterCreationSaga without a Kafka round-trip.
func (p *ProcessorImpl) resolveMapleLifePreset(ctx context.Context, in MapleLifeCreateRestModel) (preset.RestModel, map[uint32]data.SkillInfo, error) {
	t := tenant.MustFromContext(ctx)
	tc, err := configuration.GetTenantConfig(t.Id())
	if err != nil {
		p.l.WithError(err).Errorf("Unable to find maple life configuration.")
		return preset.RestModel{}, nil, err
	}
	ml := tc.MapleLife
	if len(ml.Classes) == 0 {
		return preset.RestModel{}, nil, ErrMapleLifeNotConfigured
	}

	if !validGender(in.Gender) {
		return preset.RestModel{}, nil, ErrLookInvalid
	}

	entry, entryFound := findMapleLifeClass(ml.Classes, in.ClassOrdinal, in.Gender)
	if !entryFound {
		return preset.RestModel{}, nil, ErrClassOrdinalUnknown
	}

	look, lookFound := findMapleLifeLook(ml.Looks, in.Gender)
	if !lookFound {
		return preset.RestModel{}, nil, ErrMapleLifeNotConfigured
	}

	if !validOption(look.Faces, in.Face) {
		p.l.Errorf("Chosen face [%d] is not offered for maple life class [%d].", in.Face, in.ClassOrdinal)
		return preset.RestModel{}, nil, ErrLookInvalid
	}
	if !validOption(look.Hairs, in.Hair) {
		p.l.Errorf("Chosen hair [%d] is not offered for maple life class [%d].", in.Hair, in.ClassOrdinal)
		return preset.RestModel{}, nil, ErrLookInvalid
	}
	if !validOption(look.HairColors, in.HairColor) {
		p.l.Errorf("Chosen hair color [%d] is not offered for maple life class [%d].", in.HairColor, in.ClassOrdinal)
		return preset.RestModel{}, nil, ErrLookInvalid
	}
	if !validOption(look.SkinColors, uint32(in.SkinColor)) {
		p.l.Errorf("Chosen skin color [%d] is not offered for maple life class [%d].", in.SkinColor, in.ClassOrdinal)
		return preset.RestModel{}, nil, ErrLookInvalid
	}

	if entry.SpSkillId == 0 {
		if in.SP != 0 {
			return preset.RestModel{}, nil, ErrSPInvalid
		}
	} else {
		pool, ok := parseSPPool(entry.SP)
		if !ok || len(pool) == 0 || in.SP > 10 || int(in.SP) > pool[0] {
			return preset.RestModel{}, nil, ErrSPInvalid
		}
	}

	nv, err := p.nameClient.Check(ctx, in.Name, in.WorldId)
	if err != nil {
		return preset.RestModel{}, nil, err
	}
	if !nv.Valid {
		if nv.Reason == "duplicate" {
			return preset.RestModel{}, nil, ErrNameDuplicate
		}
		return preset.RestModel{}, nil, &NameInvalidError{Reason: nv.Reason, Detail: nv.Detail}
	}

	// Re-validate equipment + inventory against atlas-data, exactly as
	// CreateFromPreset does for an admin-authored preset -- a Maple Life
	// class entry is configuration data too, and can go stale the same way.
	seenSlots := map[uint32]bool{}
	for _, eq := range entry.Equipment {
		info, err := p.dataClient.GetItemById(ctx, eq.TemplateId)
		if err != nil {
			return preset.RestModel{}, nil, fmt.Errorf("%w: equipment %d", ErrPresetValidation, eq.TemplateId)
		}
		if !info.Equipable {
			return preset.RestModel{}, nil, fmt.Errorf("%w: not equippable: %d", ErrPresetValidation, eq.TemplateId)
		}
		bucket := eq.TemplateId / 10000
		if seenSlots[bucket] {
			return preset.RestModel{}, nil, fmt.Errorf("%w: equipment slot collision: %d", ErrPresetValidation, eq.TemplateId)
		}
		seenSlots[bucket] = true
	}
	for _, inv := range entry.Inventory {
		if _, err := p.dataClient.GetItemById(ctx, inv.TemplateId); err != nil {
			return preset.RestModel{}, nil, fmt.Errorf("%w: inventory %d", ErrPresetValidation, inv.TemplateId)
		}
	}

	// Only ordinals with a spSkillId ever consult atlas-data for the skill
	// -- ordinals 2/3/4 offer no SP step and never reach here with SP > 0.
	skillsById := map[uint32]data.SkillInfo{}
	var effectX int16
	if entry.SpSkillId != 0 {
		got, err := p.dataClient.GetSkillsByIds(ctx, []uint32{entry.SpSkillId})
		if err != nil {
			return preset.RestModel{}, nil, ErrAtlasDataUnreachable
		}
		for _, sk := range got {
			skillsById[sk.Id] = sk
		}
		info, ok := skillsById[entry.SpSkillId]
		if !ok {
			return preset.RestModel{}, nil, fmt.Errorf("%w: skill not found: %d", ErrPresetValidation, entry.SpSkillId)
		}
		if in.SP > 0 {
			x, ok := info.EffectXAt(in.SP)
			if !ok {
				return preset.RestModel{}, nil, fmt.Errorf("%w: sp skill %d has no effect at level %d", ErrPresetValidation, entry.SpSkillId, in.SP)
			}
			effectX = x
		}
	}

	pr := toPreset(entry, in, effectX)
	return pr, skillsById, nil
}

func findMapleLifeClass(classes []maplelife.ClassEntry, ordinal uint32, gender byte) (maplelife.ClassEntry, bool) {
	for _, c := range classes {
		if c.Ordinal == ordinal && c.Gender == gender {
			return c, true
		}
	}
	return maplelife.ClassEntry{}, false
}

func findMapleLifeLook(looks []maplelife.LookOptions, gender byte) (maplelife.LookOptions, bool) {
	for _, l := range looks {
		if l.Gender == gender {
			return l, true
		}
	}
	return maplelife.LookOptions{}, false
}

// toPreset projects a Maple Life class entry plus the player's own choices
// onto the preset shape, so buildPresetCharacterCreationSaga -- already in
// production for the admin path -- builds the saga. Three fields come from
// the player rather than the configuration, which is the whole difference
// between this and a preset: the four look values, the gender, and the level
// of the class's SP skill.
//
// effectX is the atlas-data-sourced per-level HP/MP gain bonus for
// e.SpSkillId at level in.SP (0 when the class has no SP skill, or the
// player invested none of it). Taking it as a parameter rather than
// resolving it here -- a deviation from the plan text's toPreset(e, in)
// two-argument shape -- is the CONTROLLER AMENDMENT's requirement that Task
// 22 compute the SP-skill's own HP/MP contribution: CreateMapleLife has
// already run the atlas-data lookup by the time it calls this, and this
// function stays a pure projection rather than repeating that I/O.
func toPreset(e maplelife.ClassEntry, in MapleLifeCreateRestModel, effectX int16) preset.RestModel {
	equipment := make([]preset.EquipmentEntry, 0, len(e.Equipment))
	for _, eq := range e.Equipment {
		equipment = append(equipment, preset.EquipmentEntry{TemplateId: eq.TemplateId, UseAverageStats: eq.UseAverageStats})
	}
	inventory := make([]preset.InventoryEntry, 0, len(e.Inventory))
	for _, inv := range e.Inventory {
		inventory = append(inventory, preset.InventoryEntry{TemplateId: inv.TemplateId, Quantity: inv.Quantity})
	}

	var skills []preset.SkillEntry
	if e.SpSkillId != 0 && in.SP > 0 {
		skills = []preset.SkillEntry{{SkillId: e.SpSkillId, Level: in.SP}}
	}

	// maple-life-content.md §5.3(b): the seeded Stats.Hp/Mp is the
	// skill-EXCLUDED midpoint (Task 20's job). The SP-skill's own
	// contribution is `29 x levelUps x(nSP)`, mirroring
	// ProcessLevelChange's single-batch resolve-once-per-call shape for
	// Maple Life's synthetic level 1->30 history. Only Warrior (HP) and
	// Magician (MP) ever carry a non-zero effectX; the other three classes
	// pass their seeded stats through untouched.
	stats := e.Stats
	if e.SpSkillId != 0 && in.SP > 0 && effectX != 0 {
		contribution := uint16(29 * int(effectX))
		switch e.SpSkillId {
		case uint32(skill.WarriorImprovedMaxHpIncreaseId):
			stats.Hp += contribution
		case uint32(skill.MagicianImprovedMaxMpIncreaseId):
			stats.Mp += contribution
		}
	}

	remainingSP := spendSPPool(e.SP, in.SP)

	return preset.RestModel{
		Attributes: preset.Attributes{
			JobId:     e.JobId,
			Gender:    in.Gender,
			Face:      in.Face,
			Hair:      in.Hair,
			HairColor: in.HairColor,
			SkinColor: in.SkinColor,
			MapId:     e.MapId,
			Level:     e.Level,
			Meso:      e.Meso,
			Gm:        0,
			Stats: preset.StatBlock{
				Str: stats.Str,
				Dex: stats.Dex,
				Int: stats.Int,
				Luk: stats.Luk,
				Hp:  stats.Hp,
				Mp:  stats.Mp,
			},
			Equipment: equipment,
			Inventory: inventory,
			Skills:    skills,
			AP:        e.AP,
			SP:        remainingSP,
		},
	}
}

// parseSPPool decodes the ten-book SP string atlas-character persists
// (character/entity.go:58, "0,0,0,0,0,0,0,0,0,0") into its per-book ints.
// Maple Life only ever reads/spends slot 0.
func parseSPPool(sp string) ([]int, bool) {
	parts := strings.Split(sp, ",")
	out := make([]int, len(parts))
	for i, part := range parts {
		v, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return nil, false
		}
		out[i] = v
	}
	return out, true
}

// spendSPPool spends nSP out of slot 0 of the ten-book SP string, leaving the
// other nine books untouched, and re-encodes it in the same form. An
// unparseable pool (should not happen -- entry.SP is validated seed data)
// passes through unchanged rather than panicking.
func spendSPPool(sp string, nSP byte) string {
	pool, ok := parseSPPool(sp)
	if !ok || len(pool) == 0 {
		return sp
	}
	pool[0] -= int(nSP)
	parts := make([]string, len(pool))
	for i, v := range pool {
		parts[i] = strconv.Itoa(v)
	}
	return strings.Join(parts, ",")
}
