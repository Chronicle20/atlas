package asset_test

import (
	"atlas-inventory/asset"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"net/http"
	"net/http/httptest"
	"testing"
)

type Server struct {
	baseUrl string
	prefix  string
}

func (s Server) GetBaseURL() string {
	return s.baseUrl
}

func (s Server) GetPrefix() string {
	return s.prefix
}

func GetServer() Server {
	return Server{
		baseUrl: "",
		prefix:  "/api/",
	}
}

func testLogger() logrus.FieldLogger {
	l, _ := test.NewNullLogger()
	return l
}

func TestMarshalUnmarshalSunny(t *testing.T) {
	ieam := asset.NewBuilder[any](1, 1040010, 380, asset.ReferenceTypeEquipable).
		SetReferenceData(asset.NewEquipableReferenceDataBuilder().
			SetWeaponDefense(3).
			SetSlots(7).
			Build()).
		Build()
	ierm, err := model.Map(asset.Transform)(model.FixedProvider(ieam))()
	if err != nil {
		t.Fatalf("Failed to transform model.")
	}

	rr := httptest.NewRecorder()
	server.MarshalResponse[asset.BaseRestModel](testLogger())(rr)(GetServer())(make(map[string][]string))(ierm)

	if rr.Code != http.StatusOK {
		t.Fatalf("Failed to write rest model: %v", err)
	}

	body := rr.Body.Bytes()

	oerm := asset.BaseRestModel{}
	err = jsonapi.Unmarshal(body, &oerm)
	if err != nil {
		t.Fatalf("Failed to unmarshal rest model.")
	}

	oeam, err := model.Map(asset.Extract)(model.FixedProvider(oerm))()
	if err != nil {
		t.Fatalf("Failed to extract model.")
	}

	if ieam.Id() != oeam.Id() {
		t.Fatalf("Ids do not match")
	}
	if ieam.TemplateId() != oeam.TemplateId() {
		t.Fatalf("Template Ids do not match")
	}
	if ieam.ReferenceId() != oeam.ReferenceId() {
		t.Fatalf("Reference Ids do not match")
	}
	if ieam.ReferenceType() != oeam.ReferenceType() {
		t.Fatalf("Reference Types do not match")
	}
	var ird asset.EquipableReferenceData
	var ok bool
	if ird, ok = ieam.ReferenceData().(asset.EquipableReferenceData); !ok {
		t.Fatalf("Cannot cast ReferenceData")
	}
	var ord asset.EquipableReferenceData
	if ord, ok = oeam.ReferenceData().(asset.EquipableReferenceData); !ok {
		t.Fatalf("Cannot cast ReferenceData")
	}
	if ird.GetStrength() != ord.GetStrength() {
		t.Errorf("Strength mismatch: %d != %d", ird.GetStrength(), ord.GetStrength())
	}
	if ird.GetDexterity() != ord.GetDexterity() {
		t.Errorf("Dexterity mismatch: %d != %d", ird.GetDexterity(), ord.GetDexterity())
	}
	if ird.GetIntelligence() != ord.GetIntelligence() {
		t.Errorf("Intelligence mismatch: %d != %d", ird.GetIntelligence(), ord.GetIntelligence())
	}
	if ird.GetLuck() != ord.GetLuck() {
		t.Errorf("Luck mismatch: %d != %d", ird.GetLuck(), ord.GetLuck())
	}
	if ird.GetHP() != ord.GetHP() {
		t.Errorf("HP mismatch: %d != %d", ird.GetHP(), ord.GetHP())
	}
	if ird.GetMP() != ord.GetMP() {
		t.Errorf("MP mismatch: %d != %d", ird.GetMP(), ord.GetMP())
	}
	if ird.GetWeaponAttack() != ord.GetWeaponAttack() {
		t.Errorf("WeaponAttack mismatch: %d != %d", ird.GetWeaponAttack(), ord.GetWeaponAttack())
	}
	if ird.GetMagicAttack() != ord.GetMagicAttack() {
		t.Errorf("MagicAttack mismatch: %d != %d", ird.GetMagicAttack(), ord.GetMagicAttack())
	}
	if ird.GetWeaponDefense() != ord.GetWeaponDefense() {
		t.Errorf("WeaponDefense mismatch: %d != %d", ird.GetWeaponDefense(), ord.GetWeaponDefense())
	}
	if ird.GetMagicDefense() != ord.GetMagicDefense() {
		t.Errorf("MagicDefense mismatch: %d != %d", ird.GetMagicDefense(), ord.GetMagicDefense())
	}
	if ird.GetAccuracy() != ord.GetAccuracy() {
		t.Errorf("Accuracy mismatch: %d != %d", ird.GetAccuracy(), ord.GetAccuracy())
	}
	if ird.GetAvoidability() != ord.GetAvoidability() {
		t.Errorf("Avoidability mismatch: %d != %d", ird.GetAvoidability(), ord.GetAvoidability())
	}
	if ird.GetHands() != ord.GetHands() {
		t.Errorf("Hands mismatch: %d != %d", ird.GetHands(), ord.GetHands())
	}
	if ird.GetSpeed() != ord.GetSpeed() {
		t.Errorf("Speed mismatch: %d != %d", ird.GetSpeed(), ord.GetSpeed())
	}
	if ird.GetJump() != ord.GetJump() {
		t.Errorf("Jump mismatch: %d != %d", ird.GetJump(), ord.GetJump())
	}
	if ird.GetSlots() != ord.GetSlots() {
		t.Errorf("Slots mismatch: %d != %d", ird.GetSlots(), ord.GetSlots())
	}
	if ird.GetOwnerId() != ord.GetOwnerId() {
		t.Errorf("OwnerId mismatch: %d != %d", ird.GetOwnerId(), ord.GetOwnerId())
	}
	if ird.IsLocked() != ord.IsLocked() {
		t.Errorf("Locked mismatch: %v != %v", ird.IsLocked(), ord.IsLocked())
	}
	if ird.HasSpikes() != ord.HasSpikes() {
		t.Errorf("Spikes mismatch: %v != %v", ird.HasSpikes(), ord.HasSpikes())
	}
	if ird.IsKarmaUsed() != ord.IsKarmaUsed() {
		t.Errorf("KarmaUsed mismatch: %v != %v", ird.IsKarmaUsed(), ord.IsKarmaUsed())
	}
	if ird.IsCold() != ord.IsCold() {
		t.Errorf("Cold mismatch: %v != %v", ird.IsCold(), ord.IsCold())
	}
	if ird.CanBeTraded() != ord.CanBeTraded() {
		t.Errorf("CanBeTraded mismatch: %v != %v", ird.CanBeTraded(), ord.CanBeTraded())
	}
	if ird.GetLevelType() != ord.GetLevelType() {
		t.Errorf("LevelType mismatch: %d != %d", ird.GetLevelType(), ord.GetLevelType())
	}
	if ird.GetLevel() != ord.GetLevel() {
		t.Errorf("Level mismatch: %d != %d", ird.GetLevel(), ord.GetLevel())
	}
	if ird.GetExperience() != ord.GetExperience() {
		t.Errorf("Experience mismatch: %d != %d", ird.GetExperience(), ord.GetExperience())
	}
	if ird.GetHammersApplied() != ord.GetHammersApplied() {
		t.Errorf("HammersApplied mismatch: %d != %d", ird.GetHammersApplied(), ord.GetHammersApplied())
	}
	if !ird.GetExpiration().Equal(ord.GetExpiration()) {
		t.Errorf("Expiration mismatch: %v != %v", ird.GetExpiration(), ord.GetExpiration())
	}
}
