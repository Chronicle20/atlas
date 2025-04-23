package asset_test

import (
	"atlas-inventory/asset"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/google/uuid"
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
	ieam := asset.NewBuilder[any](1, uuid.New(), 1040010, 380, asset.ReferenceTypeEquipable).
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
	if ird.Strength() != ord.Strength() {
		t.Errorf("Strength mismatch: %d != %d", ird.Strength(), ord.Strength())
	}
	if ird.Dexterity() != ord.Dexterity() {
		t.Errorf("Dexterity mismatch: %d != %d", ird.Dexterity(), ord.Dexterity())
	}
	if ird.Intelligence() != ord.Intelligence() {
		t.Errorf("Intelligence mismatch: %d != %d", ird.Intelligence(), ord.Intelligence())
	}
	if ird.Luck() != ord.Luck() {
		t.Errorf("Luck mismatch: %d != %d", ird.Luck(), ord.Luck())
	}
	if ird.HP() != ord.HP() {
		t.Errorf("HP mismatch: %d != %d", ird.HP(), ord.HP())
	}
	if ird.MP() != ord.MP() {
		t.Errorf("MP mismatch: %d != %d", ird.MP(), ord.MP())
	}
	if ird.WeaponAttack() != ord.WeaponAttack() {
		t.Errorf("WeaponAttack mismatch: %d != %d", ird.WeaponAttack(), ord.WeaponAttack())
	}
	if ird.MagicAttack() != ord.MagicAttack() {
		t.Errorf("MagicAttack mismatch: %d != %d", ird.MagicAttack(), ord.MagicAttack())
	}
	if ird.WeaponDefense() != ord.WeaponDefense() {
		t.Errorf("WeaponDefense mismatch: %d != %d", ird.WeaponDefense(), ord.WeaponDefense())
	}
	if ird.MagicDefense() != ord.MagicDefense() {
		t.Errorf("MagicDefense mismatch: %d != %d", ird.MagicDefense(), ord.MagicDefense())
	}
	if ird.Accuracy() != ord.Accuracy() {
		t.Errorf("Accuracy mismatch: %d != %d", ird.Accuracy(), ord.Accuracy())
	}
	if ird.Avoidability() != ord.Avoidability() {
		t.Errorf("Avoidability mismatch: %d != %d", ird.Avoidability(), ord.Avoidability())
	}
	if ird.Hands() != ord.Hands() {
		t.Errorf("Hands mismatch: %d != %d", ird.Hands(), ord.Hands())
	}
	if ird.Speed() != ord.Speed() {
		t.Errorf("Speed mismatch: %d != %d", ird.Speed(), ord.Speed())
	}
	if ird.Jump() != ord.Jump() {
		t.Errorf("Jump mismatch: %d != %d", ird.Jump(), ord.Jump())
	}
	if ird.Slots() != ord.Slots() {
		t.Errorf("Slots mismatch: %d != %d", ird.Slots(), ord.Slots())
	}
	if ird.OwnerId() != ord.OwnerId() {
		t.Errorf("OwnerId mismatch: %d != %d", ird.OwnerId(), ord.OwnerId())
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
	if ird.LevelType() != ord.LevelType() {
		t.Errorf("LevelType mismatch: %d != %d", ird.LevelType(), ord.LevelType())
	}
	if ird.Level() != ord.Level() {
		t.Errorf("Level mismatch: %d != %d", ird.Level(), ord.Level())
	}
	if ird.Experience() != ord.Experience() {
		t.Errorf("Experience mismatch: %d != %d", ird.Experience(), ord.Experience())
	}
	if ird.HammersApplied() != ord.HammersApplied() {
		t.Errorf("HammersApplied mismatch: %d != %d", ird.HammersApplied(), ord.HammersApplied())
	}
	if !ird.Expiration().Equal(ord.Expiration()) {
		t.Errorf("Expiration mismatch: %v != %v", ird.Expiration(), ord.Expiration())
	}
}
