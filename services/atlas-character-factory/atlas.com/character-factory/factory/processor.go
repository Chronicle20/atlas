package factory

import (
	"atlas-character-factory/character"
	"atlas-character-factory/configuration"
	"atlas-character-factory/configuration/tenant/characters/preset"
	"atlas-character-factory/configuration/tenant/characters/template"
	"atlas-character-factory/data"
	job2 "atlas-character-factory/job"
	"atlas-character-factory/saga"
	"context"
	"errors"
	"fmt"
	"time"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

var (
	ErrInvalidPresetId      = errors.New("invalid preset id")
	ErrPresetNotFound       = errors.New("preset not found")
	ErrAtlasDataUnreachable = errors.New("atlas-data unreachable")
	ErrPresetValidation     = errors.New("preset validation failed")
	ErrNameDuplicate        = errors.New("name duplicate")
)

// NameInvalidError is returned when the name validity check rejects the name for a
// reason other than duplication.
type NameInvalidError struct {
	Reason string
	Detail string
}

func (e *NameInvalidError) Error() string { return "invalid name: " + e.Reason }

// Processor defines the interface for character creation operations
type Processor interface {
	Create(ctx context.Context, input RestModel) (string, error)
	CreateFromPreset(ctx context.Context, in PresetCreateRestModel) (string, error)
	CheckNameValidity(ctx context.Context, name string, worldId byte) (character.NameValidityResult, error)
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l            logrus.FieldLogger
	presetClient configuration.PresetClient
	nameClient   character.NameValidityClient
	dataClient   data.Client
}

// NewProcessor creates a new Processor instance with real HTTP clients.
func NewProcessor(l logrus.FieldLogger) Processor {
	return &ProcessorImpl{
		l:            l,
		presetClient: configuration.NewPresetClient(l),
		nameClient:   character.NewNameValidityClient(l),
		dataClient:   data.NewClient(l),
	}
}

// NewProcessorWithClients is the test seam — allows injection of mocks.
func NewProcessorWithClients(l logrus.FieldLogger, pc configuration.PresetClient, nc character.NameValidityClient, dc data.Client) Processor {
	return &ProcessorImpl{l: l, presetClient: pc, nameClient: nc, dataClient: dc}
}

// Create validates and initiates character creation via saga
func (p *ProcessorImpl) Create(ctx context.Context, input RestModel) (string, error) {
	// Validate character name
	if !validName(input.Name) {
		return "", errors.New("character name must be between 1 and 12 characters and contain only valid characters")
	}

	if !validGender(input.Gender) {
		return "", errors.New("gender must be 0 or 1")
	}

	if !validJob(input.JobIndex, input.SubJobIndex) {
		return "", errors.New("must provide valid job index")
	}

	t := tenant.MustFromContext(ctx)
	tc, err := configuration.GetTenantConfig(t.Id())
	if err != nil {
		p.l.WithError(err).Errorf("Unable to find template validation configuration")
		return "", err
	}

	var found = false
	var tmpl template.RestModel
	for _, ref := range tc.Characters.Templates {
		if ref.JobIndex == input.JobIndex && ref.SubJobIndex == input.SubJobIndex && ref.Gender == input.Gender {
			found = true
			tmpl = ref
		}
	}
	if !found {
		p.l.WithError(err).Errorf("Unable to find template validation configuration")
		return "", err
	}

	if !validFace(tmpl.Faces, input.Face) {
		p.l.Errorf("Chosen face [%d] is not valid for job [%d].", input.Face, input.JobIndex)
		return "", errors.New("chosen face is not valid for job")
	}

	if !validHair(tmpl.Hairs, input.Hair) {
		p.l.Errorf("Chosen hair [%d] is not valid for job [%d].", input.Hair, input.JobIndex)
		return "", errors.New("chosen hair is not valid for job")
	}

	if !validHairColor(tmpl.HairColors, input.HairColor) {
		p.l.Errorf("Chosen hair color [%d] is not valid for job [%d].", input.HairColor, input.JobIndex)
		return "", errors.New("chosen hair color is not valid for job")
	}

	if !validSkinColor(tmpl.SkinColors, uint32(input.SkinColor)) {
		p.l.Errorf("Chosen skin color [%d] is not valid for job [%d]", input.SkinColor, input.JobIndex)
		return "", errors.New("chosen skin color is not valid for job")
	}

	if !validTop(tmpl.Tops, input.Top) {
		p.l.Errorf("Chosen top [%d] is not valid for job [%d]", input.Top, input.JobIndex)
		return "", errors.New("chosen top is not valid for job")
	}

	if !validBottom(tmpl.Bottoms, input.Bottom) {
		p.l.Errorf("Chosen bottom [%d] is not valid for job [%d]", input.Bottom, input.JobIndex)
		return "", errors.New("chosen bottom is not valid for job")
	}

	if !validShoes(tmpl.Shoes, input.Shoes) {
		p.l.Errorf("Chosen shoes [%d] is not valid for job [%d]", input.Shoes, input.JobIndex)
		return "", errors.New("chosen shoes is not valid for job")
	}

	if !validWeapon(tmpl.Weapons, input.Weapon) {
		p.l.Errorf("Chosen weapon [%d] is not valid for job [%d]", input.Weapon, input.JobIndex)
		return "", errors.New("chosen weapon is not valid for job")
	}

	if input.MapId == _map.Id(0) {
		p.l.Debugf("Starting map not provided. Leveraging what is configured in the template.")
		input.MapId = _map.Id(tmpl.MapId)
	}

	// Generate transaction ID for character creation saga
	characterCreationId := uuid.New()
	p.l.Debugf("Beginning character creation saga for account [%d] in world [%d] with transaction [%s].", input.AccountId, input.WorldId, characterCreationId.String())

	// Build the unified character creation saga
	characterCreationSaga := buildCharacterCreationSaga(characterCreationId, input, tmpl)

	// Emit the character creation saga
	sagaProcessor := saga.NewProcessor(p.l, ctx)
	err = sagaProcessor.Create(characterCreationSaga)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to emit character creation saga for character [%s].", input.Name)
		return "", err
	}

	p.l.Debugf("Character creation saga [%s] emitted successfully for character [%s].", characterCreationId.String(), input.Name)
	return characterCreationId.String(), nil
}

// buildCharacterCreationSaga constructs a unified saga that creates a character and awards
// all items, equipment, and skills in a single transaction. Subsequent steps use characterId=0
// as a sentinel value; the saga orchestrator's result forwarding will inject the actual
// characterId after the CreateCharacter step completes.
func buildCharacterCreationSaga(transactionId uuid.UUID, input RestModel, tmpl template.RestModel) saga.Saga {
	// Character creation caps the orchestrator's backstop timer at 10s so the
	// client's socket is released within the login latency budget (see PRD §4.1 /
	// plan Phase 4.5). The orchestrator otherwise defaults to 30s.
	builder := saga.NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(saga.CharacterCreation).
		SetInitiatedBy(fmt.Sprintf("account_%d", input.AccountId)).
		SetTimeout(10 * time.Second)

	// Step 1: Create character
	builder.AddStep("create_character", saga.Pending, saga.CreateCharacter, saga.CharacterCreatePayload{
		AccountId:    input.AccountId,
		WorldId:      input.WorldId,
		Name:         input.Name,
		Gender:       input.Gender,
		Level:        input.Level,
		Strength:     input.Strength,
		Dexterity:    input.Dexterity,
		Intelligence: input.Intelligence,
		Luck:         input.Luck,
		JobId:        job2.JobFromIndex(input.JobIndex, input.SubJobIndex),
		Hp:           input.Hp,
		Mp:           input.Mp,
		Face:         input.Face,
		Hair:         input.Hair + input.HairColor,
		Skin:         input.SkinColor,
		Top:          input.Top,
		Bottom:       input.Bottom,
		Shoes:        input.Shoes,
		Weapon:       input.Weapon,
		MapId:        input.MapId,
	})

	// Steps 2-N: Award assets for template items (characterId=0, forwarded by orchestrator)
	for i, templateId := range tmpl.Items {
		builder.AddStep(fmt.Sprintf("award_item_%d", i), saga.Pending, saga.AwardAsset, saga.AwardItemActionPayload{
			CharacterId: 0,
			Item:        saga.ItemPayload{TemplateId: templateId, Quantity: 1},
		})
	}

	// Steps N+1-N+4: Create and equip assets for equipment (characterId=0, forwarded by orchestrator)
	equipment := []struct {
		templateId uint32
		name       string
	}{
		{input.Top, "top"},
		{input.Bottom, "bottom"},
		{input.Shoes, "shoes"},
		{input.Weapon, "weapon"},
	}
	for _, eq := range equipment {
		if eq.templateId != 0 {
			builder.AddStep(fmt.Sprintf("equip_%s", eq.name), saga.Pending, saga.CreateAndEquipAsset, saga.CreateAndEquipAssetPayload{
				CharacterId: 0,
				Item:        saga.ItemPayload{TemplateId: eq.templateId, Quantity: 1},
			})
		}
	}

	// Steps N+5-M: Create skills for starter skills (characterId=0, forwarded by orchestrator)
	for i, skillId := range tmpl.Skills {
		builder.AddStep(fmt.Sprintf("create_skill_%d", i), saga.Pending, saga.CreateSkill, saga.CreateSkillPayload{
			CharacterId: 0,
			SkillId:     skillId,
			Level:       1,
			MasterLevel: 0,
			Expiration:  time.Time{},
		})
	}

	return builder.Build()
}

// CreateFromPreset resolves a named preset from configuration, validates the
// character name, re-validates all item and skill ids against atlas-data, then
// builds and emits a CharacterCreation saga.
func (p *ProcessorImpl) CreateFromPreset(ctx context.Context, in PresetCreateRestModel) (string, error) {
	presetId, err := uuid.Parse(in.PresetId)
	if err != nil {
		return "", ErrInvalidPresetId
	}

	t := tenant.MustFromContext(ctx)
	pr, err := p.presetClient.GetById(ctx, t.Id(), presetId)
	if err != nil {
		if errors.Is(err, configuration.ErrPresetNotFound) {
			return "", ErrPresetNotFound
		}
		return "", err
	}

	nv, err := p.nameClient.Check(ctx, in.Name, in.WorldId)
	if err != nil {
		return "", err
	}
	if !nv.Valid {
		if nv.Reason == "duplicate" {
			return "", ErrNameDuplicate
		}
		return "", &NameInvalidError{Reason: nv.Reason, Detail: nv.Detail}
	}

	// Re-validate equipment + inventory against atlas-data
	seenSlots := map[uint32]bool{}
	for _, eq := range pr.Attributes.Equipment {
		info, err := p.dataClient.GetItemById(ctx, eq.TemplateId)
		if err != nil {
			return "", fmt.Errorf("%w: equipment %d", ErrPresetValidation, eq.TemplateId)
		}
		if !info.Equipable {
			return "", fmt.Errorf("%w: not equippable: %d", ErrPresetValidation, eq.TemplateId)
		}
		bucket := eq.TemplateId / 10000
		if seenSlots[bucket] {
			return "", fmt.Errorf("%w: equipment slot collision: %d", ErrPresetValidation, eq.TemplateId)
		}
		seenSlots[bucket] = true
	}
	for _, inv := range pr.Attributes.Inventory {
		if _, err := p.dataClient.GetItemById(ctx, inv.TemplateId); err != nil {
			return "", fmt.Errorf("%w: inventory %d", ErrPresetValidation, inv.TemplateId)
		}
	}

	// Batch-fetch skill MaxLevels.
	skillsById := map[uint32]data.SkillInfo{}
	if len(pr.Attributes.Skills) > 0 {
		ids := make([]uint32, 0, len(pr.Attributes.Skills))
		for _, sk := range pr.Attributes.Skills {
			ids = append(ids, sk.SkillId)
		}
		got, err := p.dataClient.GetSkillsByIds(ctx, ids)
		if err != nil {
			return "", ErrAtlasDataUnreachable
		}
		for _, sk := range got {
			skillsById[sk.Id] = sk
		}
		for _, sk := range pr.Attributes.Skills {
			if _, ok := skillsById[sk.SkillId]; !ok {
				return "", fmt.Errorf("%w: skill not found: %d", ErrPresetValidation, sk.SkillId)
			}
		}
	}

	transactionId := uuid.New()
	sg := buildPresetCharacterCreationSaga(transactionId, in, pr, skillsById)
	if err := saga.NewProcessor(p.l, ctx).Create(sg); err != nil {
		return "", err
	}
	return transactionId.String(), nil
}

// CheckNameValidity delegates the name validity check to atlas-character via the injected client.
func (p *ProcessorImpl) CheckNameValidity(ctx context.Context, name string, worldId byte) (character.NameValidityResult, error) {
	return p.nameClient.Check(ctx, name, worldId)
}

// buildPresetCharacterCreationSaga constructs a CharacterCreation saga from a preset
// configuration. Equipment goes through create_and_equip_asset steps; the legacy
// top/bottom/shoes/weapon slots are set to 0.
func buildPresetCharacterCreationSaga(
	transactionId uuid.UUID,
	in PresetCreateRestModel,
	pr preset.RestModel,
	skillsById map[uint32]data.SkillInfo,
) saga.Saga {
	a := pr.Attributes
	builder := saga.NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(saga.CharacterCreation).
		SetInitiatedBy(fmt.Sprintf("account_%d", in.AccountId)).
		SetTimeout(10 * time.Second)

	// Step 1: create_character
	builder.AddStep("create_character", saga.Pending, saga.CreateCharacter, saga.CharacterCreatePayload{
		AccountId:    in.AccountId,
		WorldId:      world.Id(in.WorldId),
		Name:         in.Name,
		Gender:       a.Gender,
		Level:        a.Level,
		Strength:     a.Stats.Str,
		Dexterity:    a.Stats.Dex,
		Intelligence: a.Stats.Int,
		Luck:         a.Stats.Luk,
		JobId:        job.Id(a.JobId),
		Hp:           a.Stats.Hp,
		Mp:           a.Stats.Mp,
		Face:         a.Face,
		Hair:         a.Hair + a.HairColor,
		Skin:         a.SkinColor,
		Top:          0,
		Bottom:       0,
		Shoes:        0,
		Weapon:       0,
		MapId:        _map.Id(a.MapId),
		Gm:           a.Gm,
		Meso:         a.Meso,
	})

	// Steps 2..N+1: award_asset for each inventory item
	for i, inv := range a.Inventory {
		builder.AddStep(fmt.Sprintf("award_asset_%d", i), saga.Pending, saga.AwardAsset, saga.AwardItemActionPayload{
			CharacterId: 0,
			Item:        saga.ItemPayload{TemplateId: inv.TemplateId, Quantity: inv.Quantity},
		})
	}

	// Steps N+2..N+M+1: create_and_equip_asset for each equipment entry
	for i, eq := range a.Equipment {
		builder.AddStep(fmt.Sprintf("create_and_equip_asset_%d", i), saga.Pending, saga.CreateAndEquipAsset, saga.CreateAndEquipAssetPayload{
			CharacterId:     0,
			Item:            saga.ItemPayload{TemplateId: eq.TemplateId, Quantity: 1},
			UseAverageStats: eq.UseAverageStats,
		})
	}

	// Steps N+M+2..end: create_skill for each skill entry
	for i, sk := range a.Skills {
		master := skillsById[sk.SkillId].MaxLevel
		builder.AddStep(fmt.Sprintf("create_skill_%d", i), saga.Pending, saga.CreateSkill, saga.CreateSkillPayload{
			CharacterId: 0,
			SkillId:     sk.SkillId,
			Level:       sk.Level,
			MasterLevel: master,
			Expiration:  time.Time{},
		})
	}

	return builder.Build()
}

func validWeapon(weapons []uint32, weapon uint32) bool {
	return validOption(weapons, weapon)
}

func validShoes(shoes []uint32, shoe uint32) bool {
	return validOption(shoes, shoe)
}

func validBottom(bottoms []uint32, bottom uint32) bool {
	return validOption(bottoms, bottom)
}

func validTop(tops []uint32, top uint32) bool {
	return validOption(tops, top)
}

func validSkinColor(colors []uint32, color uint32) bool {
	return validOption(colors, color)
}

func validHairColor(colors []uint32, color uint32) bool {
	return validOption(colors, color)
}

func validHair(hairs []uint32, hair uint32) bool {
	return validOption(hairs, hair)
}

func validOption(options []uint32, selection uint32) bool {
	if selection == 0 {
		return true
	}

	for _, option := range options {
		if option == selection {
			return true
		}
	}
	return false
}

func validFace(faces []uint32, face uint32) bool {
	return validOption(faces, face)
}

func validJob(_ uint32, _ uint32) bool {
	return true
}

func validGender(gender byte) bool {
	return gender == 0 || gender == 1
}

func validName(name string) bool {
	if len(name) < 1 || len(name) > 12 {
		return false
	}

	// Check for valid characters (alphanumeric and common symbols)
	for _, char := range name {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '_' || char == '-') {
			return false
		}
	}

	return true
}
