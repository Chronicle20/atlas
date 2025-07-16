package factory

import (
	"atlas-character-factory/configuration"
	"atlas-character-factory/configuration/tenant/characters/template"
	"atlas-character-factory/saga"
	"context"
	"errors"
	"fmt"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"time"
)

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

			// Generate transaction ID for saga
			transactionId := uuid.New()
			l.Debugf("Beginning saga-based character creation for account [%d] in world [%d] with transaction [%s].", input.AccountId, input.WorldId, transactionId.String())

			// Build the character creation saga
			characterSaga := buildCharacterCreationSaga(transactionId, input, template)

			// Emit the saga to the orchestrator
			sagaProcessor := saga.NewProcessor(l, ctx)
			err = sagaProcessor.Create(characterSaga)
			if err != nil {
				l.WithError(err).Errorf("Unable to emit character creation saga for character [%s].", input.Name)
				return "", err
			}

			l.Debugf("Character creation saga [%s] emitted successfully for character [%s].", transactionId.String(), input.Name)
			return transactionId.String(), nil
		}

	}
}

// buildCharacterCreationSaga constructs a character creation saga with all necessary steps
// This function replaces the previous async orchestration logic with a saga-based approach.
// The saga orchestrator will execute these steps sequentially, ensuring atomicity and fault tolerance.
// Steps are constructed based on the validated template configuration for the character's job and gender.
func buildCharacterCreationSaga(transactionId uuid.UUID, input RestModel, template template.RestModel) saga.Saga {
	builder := saga.NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(saga.CharacterCreation).
		SetInitiatedBy(fmt.Sprintf("account_%d", input.AccountId))

	// Step 1: Create character
	createCharacterPayload := saga.CharacterCreatePayload{
		AccountId: input.AccountId,
		Name:      input.Name,
		WorldId:   input.WorldId,
		ChannelId: 0, // Default channel
		JobId:     input.JobIndex,
		Face:      input.Face,
		Hair:      input.Hair,
		HairColor: input.HairColor,
		Skin:      uint32(input.SkinColor),
		Top:       input.Top,
		Bottom:    input.Bottom,
		Shoes:     input.Shoes,
		Weapon:    input.Weapon,
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
				TemplateId:  eq.templateId,
				Source:      1, // Source is always 1 for creation
				Destination: 0, // Destination is always 0 for creation
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
			Level:       1,       // Default level
			MasterLevel: 0,       // Default master level
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
