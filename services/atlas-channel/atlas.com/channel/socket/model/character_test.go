package model

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

func TestShiftGeneration(t *testing.T) {
	t1, _ := tenant.Create(uuid.New(), "JMS", 185, 1)
	t2, _ := tenant.Create(uuid.New(), "GMS", 83, 1)

	validateTemplateTemporaryStats(t)(t1, 110)
	validateTemplateTemporaryStats(t)(t2, 82)
}

func validateTemplateTemporaryStats(t *testing.T) func(tenant tenant.Model, shiftBase uint) {
	return func(tenant tenant.Model, shiftBase uint) {
		validateCharacterTemporaryStatTypeByName(t)(tenant, character.TemporaryStatTypeEnergyCharge, shiftBase)
		validateCharacterTemporaryStatTypeByName(t)(tenant, character.TemporaryStatTypeDashSpeed, shiftBase+1)
		validateCharacterTemporaryStatTypeByName(t)(tenant, character.TemporaryStatTypeDashJump, shiftBase+2)
		validateCharacterTemporaryStatTypeByName(t)(tenant, character.TemporaryStatTypeMonsterRiding, shiftBase+3)
		validateCharacterTemporaryStatTypeByName(t)(tenant, character.TemporaryStatTypeSpeedInfusion, shiftBase+4)
		validateCharacterTemporaryStatTypeByName(t)(tenant, character.TemporaryStatTypeHomingBeacon, shiftBase+5)
		validateCharacterTemporaryStatTypeByName(t)(tenant, character.TemporaryStatTypeUndead, shiftBase+6)
	}
}

func validateCharacterTemporaryStatTypeByName(t *testing.T) func(tenant tenant.Model, name character.TemporaryStatType, shift uint) {
	return func(tenant tenant.Model, name character.TemporaryStatType, shift uint) {
		var ctst packetmodel.CharacterTemporaryStatType
		var err error
		ctst, err = packetmodel.CharacterTemporaryStatTypeByName(tenant)(name)
		if err != nil || ctst.Shift() != shift {
			t.Fatalf("Failed to get correct shift for [%s]. Got [%d], Expected [%d]", name, ctst.Shift(), shift)
		}
	}
}
