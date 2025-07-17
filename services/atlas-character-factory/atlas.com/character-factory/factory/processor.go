package factory

import (
	"atlas-character-factory/configuration"
	"atlas-character-factory/configuration/tenant/characters/template"
	job2 "atlas-character-factory/job"
	"atlas-character-factory/saga"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// FollowUpSagaTemplate stores the template information needed to create a follow-up saga
type FollowUpSagaTemplate struct {
	TenantId uuid.UUID
	Input    RestModel
	Template template.RestModel
}

// FollowUpSagaTemplateStore provides thread-safe storage for follow-up saga templates
type FollowUpSagaTemplateStore struct {
	templates map[string]FollowUpSagaTemplate
	mutex     sync.RWMutex
}

// Singleton instance
var (
	templateStoreInstance *FollowUpSagaTemplateStore
	templateStoreOnce     sync.Once
)

// GetFollowUpSagaTemplateStore returns the singleton instance of the template store
func GetFollowUpSagaTemplateStore() *FollowUpSagaTemplateStore {
	templateStoreOnce.Do(func() {
		templateStoreInstance = &FollowUpSagaTemplateStore{
			templates: make(map[string]FollowUpSagaTemplate),
		}
	})
	return templateStoreInstance
}

// Store stores the template information for later use when character created event is received
func (s *FollowUpSagaTemplateStore) Store(tenantId uuid.UUID, characterName string, input RestModel, template template.RestModel) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Store with tenant-aware key to avoid conflicts
	key := fmt.Sprintf("%s:%s", tenantId.String(), characterName)
	s.templates[key] = FollowUpSagaTemplate{
		TenantId: tenantId,
		Input:    input,
		Template: template,
	}

	return nil
}

// Get retrieves the stored template information
func (s *FollowUpSagaTemplateStore) Get(tenantId uuid.UUID, characterName string) (FollowUpSagaTemplate, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	key := fmt.Sprintf("%s:%s", tenantId.String(), characterName)
	template, exists := s.templates[key]
	return template, exists
}

// Remove removes the stored template information after use
func (s *FollowUpSagaTemplateStore) Remove(tenantId uuid.UUID, characterName string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	key := fmt.Sprintf("%s:%s", tenantId.String(), characterName)
	delete(s.templates, key)
}

// Clear removes all stored templates (useful for testing)
func (s *FollowUpSagaTemplateStore) Clear() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.templates = make(map[string]FollowUpSagaTemplate)
}

// Size returns the number of stored templates
func (s *FollowUpSagaTemplateStore) Size() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return len(s.templates)
}

// storeFollowUpSagaTemplate stores the template information for later use when character created event is received
func storeFollowUpSagaTemplate(ctx context.Context, characterName string, input RestModel, template template.RestModel) error {
	t := tenant.MustFromContext(ctx)
	store := GetFollowUpSagaTemplateStore()
	return store.Store(t.Id(), characterName, input, template)
}

// GetFollowUpSagaTemplate retrieves the stored template information
func GetFollowUpSagaTemplate(tenantId uuid.UUID, characterName string) (FollowUpSagaTemplate, bool) {
	store := GetFollowUpSagaTemplateStore()
	return store.Get(tenantId, characterName)
}

// RemoveFollowUpSagaTemplate removes the stored template information after use
func RemoveFollowUpSagaTemplate(tenantId uuid.UUID, characterName string) {
	store := GetFollowUpSagaTemplateStore()
	store.Remove(tenantId, characterName)
}

func Create(l logrus.FieldLogger) func(ctx context.Context) func(input RestModel) (string, error) {
	return func(ctx context.Context) func(input RestModel) (string, error) {
		return func(input RestModel) (string, error) {
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
				l.WithError(err).Errorf("Unable to find template validation configuration")
				return "", err
			}

			var found = false
			var template template.RestModel
			for _, ref := range tc.Characters.Templates {
				if ref.JobIndex == input.JobIndex && ref.SubJobIndex == input.SubJobIndex && ref.Gender == input.Gender {
					found = true
					template = ref
				}
			}
			if !found {
				l.WithError(err).Errorf("Unable to find template validation configuration")
				return "", err
			}

			if !validFace(template.Faces, input.Face) {
				l.Errorf("Chosen face [%d] is not valid for job [%d].", input.Face, input.JobIndex)
				return "", errors.New("chosen face is not valid for job")
			}

			if !validHair(template.Hairs, input.Hair) {
				l.Errorf("Chosen hair [%d] is not valid for job [%d].", input.Hair, input.JobIndex)
				return "", errors.New("chosen hair is not valid for job")
			}

			if !validHairColor(template.HairColors, input.HairColor) {
				l.Errorf("Chosen hair color [%d] is not valid for job [%d].", input.HairColor, input.JobIndex)
				return "", errors.New("chosen hair color is not valid for job")
			}

			if !validSkinColor(template.SkinColors, uint32(input.SkinColor)) {
				l.Errorf("Chosen skin color [%d] is not valid for job [%d]", input.SkinColor, input.JobIndex)
				return "", errors.New("chosen skin color is not valid for job")
			}

			if !validTop(template.Tops, input.Top) {
				l.Errorf("Chosen top [%d] is not valid for job [%d]", input.Top, input.JobIndex)
				return "", errors.New("chosen top is not valid for job")
			}

			if !validBottom(template.Bottoms, input.Bottom) {
				l.Errorf("Chosen bottom [%d] is not valid for job [%d]", input.Bottom, input.JobIndex)
				return "", errors.New("chosen bottom is not valid for job")
			}

			if !validShoes(template.Shoes, input.Shoes) {
				l.Errorf("Chosen shoes [%d] is not valid for job [%d]", input.Shoes, input.JobIndex)
				return "", errors.New("chosen shoes is not valid for job")
			}

			if !validWeapon(template.Weapons, input.Weapon) {
				l.Errorf("Chosen weapon [%d] is not valid for job [%d]", input.Weapon, input.JobIndex)
				return "", errors.New("chosen weapon is not valid for job")
			}

			// Generate transaction ID for character creation saga
			characterCreationId := uuid.New()
			l.Debugf("Beginning character creation saga for account [%d] in world [%d] with transaction [%s].", input.AccountId, input.WorldId, characterCreationId.String())

			// Build the character creation only saga
			characterOnlySaga := buildCharacterCreationOnlySaga(characterCreationId, input)

			// Store the template information for follow-up saga creation
			// This will be used when the character created event is received
			err = storeFollowUpSagaTemplate(ctx, input.Name, input, template)
			if err != nil {
				l.WithError(err).Errorf("Unable to store follow-up saga template for character [%s].", input.Name)
				return "", err
			}

			// Emit the character creation saga
			sagaProcessor := saga.NewProcessor(l, ctx)
			err = sagaProcessor.Create(characterOnlySaga)
			if err != nil {
				l.WithError(err).Errorf("Unable to emit character creation saga for character [%s].", input.Name)
				return "", err
			}

			l.Debugf("Character creation saga [%s] emitted successfully for character [%s].", characterCreationId.String(), input.Name)
			return characterCreationId.String(), nil
		}

	}
}

// buildCharacterCreationOnlySaga constructs a saga that only creates the character.
// This saga will complete when the character is created and will emit a character created event.
func buildCharacterCreationOnlySaga(transactionId uuid.UUID, input RestModel) saga.Saga {
	builder := saga.NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(saga.CharacterCreationOnly).
		SetInitiatedBy(fmt.Sprintf("account_%d", input.AccountId))

	// Step 1: Create character
	createCharacterPayload := saga.CharacterCreatePayload{
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
		Skin:         uint32(input.SkinColor),
		Top:          input.Top,
		Bottom:       input.Bottom,
		Shoes:        input.Shoes,
		Weapon:       input.Weapon,
		MapId:        input.MapId,
	}

	builder.AddStep("create_character", saga.Pending, saga.CreateCharacter, createCharacterPayload)

	return builder.Build()
}

// BuildCharacterCreationFollowUpSaga constructs a saga that awards items, equipment, and skills for a created character.
// This saga will be created after the character creation event is received and will use the actual character ID.
func BuildCharacterCreationFollowUpSaga(transactionId uuid.UUID, characterId uint32, input RestModel, template template.RestModel) saga.Saga {
	builder := saga.NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(saga.CharacterCreationFollowUp).
		SetInitiatedBy(fmt.Sprintf("account_%d", input.AccountId))

	// Step 1: Award assets for template items
	for i, templateId := range template.Items {
		stepId := fmt.Sprintf("award_item_%d", i)
		awardAssetPayload := saga.AwardItemActionPayload{
			CharacterId: characterId, // Use the actual character ID
			Item: saga.ItemPayload{
				TemplateId: templateId,
				Quantity:   1,
			},
		}
		builder.AddStep(stepId, saga.Pending, saga.AwardAsset, awardAssetPayload)
	}

	// Step 2: Create and equip assets for equipment (Top, Bottom, Shoes, Weapon)
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
		if eq.templateId != 0 { // Only add step if equipment is provided
			stepId := fmt.Sprintf("equip_%s", eq.name)
			createAndEquipPayload := saga.CreateAndEquipAssetPayload{
				CharacterId: characterId, // Use the actual character ID
				Item: saga.ItemPayload{
					TemplateId: eq.templateId,
					Quantity:   1,
				},
			}
			builder.AddStep(stepId, saga.Pending, saga.CreateAndEquipAsset, createAndEquipPayload)
		}
	}

	// Step 3: Create skills for starter skills
	for i, skillId := range template.Skills {
		stepId := fmt.Sprintf("create_skill_%d", i)
		createSkillPayload := saga.CreateSkillPayload{
			CharacterId: characterId, // Use the actual character ID
			SkillId:     skillId,
			Level:       1,           // Default level
			MasterLevel: 0,           // Default master level
			Expiration:  time.Time{}, // No expiration
		}
		builder.AddStep(stepId, saga.Pending, saga.CreateSkill, createSkillPayload)
	}

	return builder.Build()
}

// buildCharacterCreationSaga constructs a legacy combined character creation saga for backward compatibility.
// This function is deprecated and maintained only for test compatibility.
// New code should use buildCharacterCreationOnlySaga and buildCharacterCreationFollowUpSaga separately.
func buildCharacterCreationSaga(transactionId uuid.UUID, input RestModel, template template.RestModel) saga.Saga {
	builder := saga.NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(saga.CharacterCreation).
		SetInitiatedBy(fmt.Sprintf("account_%d", input.AccountId))

	// Step 1: Create character
	createCharacterPayload := saga.CharacterCreatePayload{
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
		Skin:         uint32(input.SkinColor),
		Top:          input.Top,
		Bottom:       input.Bottom,
		Shoes:        input.Shoes,
		Weapon:       input.Weapon,
		MapId:        input.MapId,
	}

	builder.AddStep("create", saga.Pending, saga.CreateCharacter, createCharacterPayload)

	// Step 2: Award assets for template items
	for i, templateId := range template.Items {
		stepId := fmt.Sprintf("award_item_%d", i)
		awardAssetPayload := saga.AwardItemActionPayload{
			CharacterId: 0, // Will be set by orchestrator after character creation
			Item: saga.ItemPayload{
				TemplateId: templateId,
				Quantity:   1,
			},
		}
		builder.AddStep(stepId, saga.Pending, saga.AwardAsset, awardAssetPayload)
	}

	// Step 3: Create and equip assets for equipment (Top, Bottom, Shoes, Weapon)
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
		if eq.templateId != 0 { // Only add step if equipment is provided
			stepId := fmt.Sprintf("equip_%s", eq.name)
			createAndEquipPayload := saga.CreateAndEquipAssetPayload{
				CharacterId: 0, // Will be set by orchestrator after character creation
				Item: saga.ItemPayload{
					TemplateId: eq.templateId,
					Quantity:   1,
				},
			}
			builder.AddStep(stepId, saga.Pending, saga.CreateAndEquipAsset, createAndEquipPayload)
		}
	}

	// Step 4: Create skills for starter skills
	for i, skillId := range template.Skills {
		stepId := fmt.Sprintf("create_skill_%d", i)
		createSkillPayload := saga.CreateSkillPayload{
			CharacterId: 0, // Will be set by orchestrator after character creation
			SkillId:     skillId,
			Level:       1,           // Default level
			MasterLevel: 0,           // Default master level
			Expiration:  time.Time{}, // No expiration
		}
		builder.AddStep(stepId, saga.Pending, saga.CreateSkill, createSkillPayload)
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

func validJob(jobIndex uint32, subJobIndex uint32) bool {
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
