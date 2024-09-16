package model

import (
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"testing"
)

func TestShiftGeneration(t *testing.T) {
	t1, _ := tenant.Create(uuid.New(), "JMS", 185, 1)
	t2, _ := tenant.Create(uuid.New(), "GMS", 83, 1)

	validateTemplateTemporaryStats(t)(t1, 110)
	//validateTemplateTemporaryStats(t)(tenant.Model{Id: uuid.New(), Region: "GMS", MajorVersion: 87, MinorVersion: 1}, 86)
	validateTemplateTemporaryStats(t)(t2, 82)
}

func validateTemplateTemporaryStats(t *testing.T) func(tenant tenant.Model, shiftBase uint) {
	return func(tenant tenant.Model, shiftBase uint) {
		validateCharacterTemporaryStatTypeByName(t)(tenant, CharacterTemporaryStatTypeNameEnergyCharge, shiftBase)
		validateCharacterTemporaryStatTypeByName(t)(tenant, CharacterTemporaryStatTypeNameDashSpeed, shiftBase+1)
		validateCharacterTemporaryStatTypeByName(t)(tenant, CharacterTemporaryStatTypeNameDashJump, shiftBase+2)
		validateCharacterTemporaryStatTypeByName(t)(tenant, CharacterTemporaryStatTypeNameMonsterRiding, shiftBase+3)
		validateCharacterTemporaryStatTypeByName(t)(tenant, CharacterTemporaryStatTypeNameSpeedInfusion, shiftBase+4)
		validateCharacterTemporaryStatTypeByName(t)(tenant, CharacterTemporaryStatTypeNameHomingBeacon, shiftBase+5)
		validateCharacterTemporaryStatTypeByName(t)(tenant, CharacterTemporaryStatTypeNameUndead, shiftBase+6)
	}
}

func validateCharacterTemporaryStatTypeByName(t *testing.T) func(tenant tenant.Model, name CharacterTemporaryStatTypeName, shift uint) {
	return func(tenant tenant.Model, name CharacterTemporaryStatTypeName, shift uint) {
		var ctst CharacterTemporaryStatType
		var err error
		ctst, err = CharacterTemporaryStatTypeByName(tenant)(name)
		if err != nil || ctst.shift != shift {
			t.Fatalf("Failed to get correct shift for [%s]. Got [%d], Expected [%d]", name, ctst.shift, shift)
		}
	}
}
