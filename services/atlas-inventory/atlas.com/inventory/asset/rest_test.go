package asset_test

import (
	"atlas-inventory/asset"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
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
	ieam := asset.NewBuilder(uuid.New(), 1040010).
		SetId(1).
		SetSlot(3).
		SetWeaponDefense(3).
		SetSlots(7).
		SetStrength(10).
		SetDexterity(5).
		Build()
	ierm, err := model.Map(asset.Transform)(model.FixedProvider(ieam))()
	if err != nil {
		t.Fatalf("Failed to transform model.")
	}

	rr := httptest.NewRecorder()
	server.MarshalResponse[asset.RestModel](testLogger())(rr)(GetServer())(make(map[string][]string))(ierm)

	if rr.Code != http.StatusOK {
		t.Fatalf("Failed to write rest model: %v", err)
	}

	body := rr.Body.Bytes()

	oerm := asset.RestModel{}
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
	if ieam.Strength() != oeam.Strength() {
		t.Errorf("Strength mismatch: %d != %d", ieam.Strength(), oeam.Strength())
	}
	if ieam.Dexterity() != oeam.Dexterity() {
		t.Errorf("Dexterity mismatch: %d != %d", ieam.Dexterity(), oeam.Dexterity())
	}
	if ieam.WeaponDefense() != oeam.WeaponDefense() {
		t.Errorf("WeaponDefense mismatch: %d != %d", ieam.WeaponDefense(), oeam.WeaponDefense())
	}
	if ieam.Slots() != oeam.Slots() {
		t.Errorf("Slots mismatch: %d != %d", ieam.Slots(), oeam.Slots())
	}
}
