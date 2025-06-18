package inventory

import (
	"atlas-cashshop/cashshop/inventory/asset"
	"atlas-cashshop/cashshop/inventory/compartment"
	"atlas-cashshop/cashshop/item"
	"atlas-cashshop/logger"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// testLogger creates a logger for testing
func testLogger() *logrus.Logger {
	return logger.CreateLogger("test")
}

// GetServer returns a server information instance for testing
func GetServer() Server {
	return Server{
		baseUrl: "",
		prefix:  "/api/",
	}
}

// Server implements jsonapi.ServerInformation
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

// TestInventoryTransformSerializeDeserializeExtract tests the transform, serialize, deserialize, and extract functions
func TestInventoryTransformSerializeDeserializeExtract(t *testing.T) {
	// Create a cash inventory with compartments, assets, and items
	accountId := uint32(12345)

	// Create items
	item1 := item.NewBuilder().
		SetId(1).
		SetCashId(1001).
		SetTemplateId(5000).
		SetQuantity(1).
		SetFlag(0).
		SetPurchasedBy(accountId).
		SetExpiration(time.Now().Add(30 * 24 * time.Hour)).
		Build()

	item2 := item.NewBuilder().
		SetId(2).
		SetCashId(1002).
		SetTemplateId(5001).
		SetQuantity(5).
		SetFlag(0).
		SetPurchasedBy(accountId).
		SetExpiration(time.Now().Add(30 * 24 * time.Hour)).
		Build()

	item3 := item.NewBuilder().
		SetId(3).
		SetCashId(1003).
		SetTemplateId(5002).
		SetQuantity(1).
		SetFlag(0).
		SetPurchasedBy(accountId).
		SetExpiration(time.Now().Add(30 * 24 * time.Hour)).
		Build()

	// Create assets
	explorerCompartmentId := uuid.New()
	cygnusCompartmentId := uuid.New()
	legendCompartmentId := uuid.New()

	asset1 := asset.NewBuilder(uuid.New(), explorerCompartmentId, item1).Build()
	asset2 := asset.NewBuilder(uuid.New(), cygnusCompartmentId, item2).Build()
	asset3 := asset.NewBuilder(uuid.New(), legendCompartmentId, item3).Build()

	// Create compartments
	explorerCompartment := compartment.NewBuilder(explorerCompartmentId, accountId, compartment.TypeExplorer, 100).
		AddAsset(asset1).
		Build()

	cygnusCompartment := compartment.NewBuilder(cygnusCompartmentId, accountId, compartment.TypeCygnus, 100).
		AddAsset(asset2).
		Build()

	legendCompartment := compartment.NewBuilder(legendCompartmentId, accountId, compartment.TypeLegend, 100).
		AddAsset(asset3).
		Build()

	// Create inventory
	inventory := NewBuilder(accountId).
		SetExplorer(explorerCompartment).
		SetCygnus(cygnusCompartment).
		SetLegend(legendCompartment).
		Build()

	// Transform the inventory to a REST model
	ierm, err := model.Map(Transform)(model.FixedProvider(inventory))()
	if err != nil {
		t.Fatalf("Failed to transform model: %v", err)
	}

	// Marshal the REST model to JSON
	rr := httptest.NewRecorder()
	server.MarshalResponse[RestModel](testLogger())(rr)(GetServer())(make(map[string][]string))(ierm)

	if rr.Code != http.StatusOK {
		t.Fatalf("Failed to write rest model: %v", err)
	}

	body := rr.Body.Bytes()

	// Unmarshal the JSON back to a REST model
	oerm := RestModel{}
	err = jsonapi.Unmarshal(body, &oerm)
	if err != nil {
		t.Fatalf("Failed to unmarshal rest model: %v", err)
	}

	// Extract the REST model back to a domain model
	oeam, err := model.Map(Extract)(model.FixedProvider(oerm))()
	if err != nil {
		t.Fatalf("Failed to extract model: %v", err)
	}

	// Verify the contents of the two domain objects (input / output)
	if inventory.AccountId() != oeam.AccountId() {
		t.Errorf("AccountId mismatch: expected %d, got %d", inventory.AccountId(), oeam.AccountId())
	}

	// Verify compartments
	if len(inventory.Compartments()) != len(oeam.Compartments()) {
		t.Errorf("Compartment count mismatch: expected %d, got %d", len(inventory.Compartments()), len(oeam.Compartments()))
	}

	// Verify explorer compartment
	explorerIn := inventory.Explorer()
	explorerOut := oeam.Explorer()

	if explorerIn.Type() != explorerOut.Type() {
		t.Errorf("Explorer compartment type mismatch: expected %d, got %d", explorerIn.Type(), explorerOut.Type())
	}

	if explorerIn.AccountId() != explorerOut.AccountId() {
		t.Errorf("Explorer compartment accountId mismatch: expected %d, got %d", explorerIn.AccountId(), explorerOut.AccountId())
	}

	if explorerIn.Capacity() != explorerOut.Capacity() {
		t.Errorf("Explorer compartment capacity mismatch: expected %d, got %d", explorerIn.Capacity(), explorerOut.Capacity())
	}

	if len(explorerIn.Assets()) != len(explorerOut.Assets()) {
		t.Errorf("Explorer compartment asset count mismatch: expected %d, got %d", len(explorerIn.Assets()), len(explorerOut.Assets()))
	}

	// Verify cygnus compartment
	cygnusIn := inventory.Cygnus()
	cygnusOut := oeam.Cygnus()

	if cygnusIn.Type() != cygnusOut.Type() {
		t.Errorf("Cygnus compartment type mismatch: expected %d, got %d", cygnusIn.Type(), cygnusOut.Type())
	}

	if cygnusIn.AccountId() != cygnusOut.AccountId() {
		t.Errorf("Cygnus compartment accountId mismatch: expected %d, got %d", cygnusIn.AccountId(), cygnusOut.AccountId())
	}

	if cygnusIn.Capacity() != cygnusOut.Capacity() {
		t.Errorf("Cygnus compartment capacity mismatch: expected %d, got %d", cygnusIn.Capacity(), cygnusOut.Capacity())
	}

	if len(cygnusIn.Assets()) != len(cygnusOut.Assets()) {
		t.Errorf("Cygnus compartment asset count mismatch: expected %d, got %d", len(cygnusIn.Assets()), len(cygnusOut.Assets()))
	}

	// Verify legend compartment
	legendIn := inventory.Legend()
	legendOut := oeam.Legend()

	if legendIn.Type() != legendOut.Type() {
		t.Errorf("Legend compartment type mismatch: expected %d, got %d", legendIn.Type(), legendOut.Type())
	}

	if legendIn.AccountId() != legendOut.AccountId() {
		t.Errorf("Legend compartment accountId mismatch: expected %d, got %d", legendIn.AccountId(), legendOut.AccountId())
	}

	if legendIn.Capacity() != legendOut.Capacity() {
		t.Errorf("Legend compartment capacity mismatch: expected %d, got %d", legendIn.Capacity(), legendOut.Capacity())
	}

	if len(legendIn.Assets()) != len(legendOut.Assets()) {
		t.Errorf("Legend compartment asset count mismatch: expected %d, got %d", len(legendIn.Assets()), len(legendOut.Assets()))
	}

	// Verify assets and items in explorer compartment
	explorerAssetIn := explorerIn.Assets()[0]
	explorerAssetOut := explorerOut.Assets()[0]

	if explorerAssetIn.TemplateId() != explorerAssetOut.TemplateId() {
		t.Errorf("Explorer asset templateId mismatch: expected %d, got %d", explorerAssetIn.TemplateId(), explorerAssetOut.TemplateId())
	}

	if explorerAssetIn.Quantity() != explorerAssetOut.Quantity() {
		t.Errorf("Explorer asset quantity mismatch: expected %d, got %d", explorerAssetIn.Quantity(), explorerAssetOut.Quantity())
	}

	// Verify assets and items in cygnus compartment
	cygnusAssetIn := cygnusIn.Assets()[0]
	cygnusAssetOut := cygnusOut.Assets()[0]

	if cygnusAssetIn.TemplateId() != cygnusAssetOut.TemplateId() {
		t.Errorf("Cygnus asset templateId mismatch: expected %d, got %d", cygnusAssetIn.TemplateId(), cygnusAssetOut.TemplateId())
	}

	if cygnusAssetIn.Quantity() != cygnusAssetOut.Quantity() {
		t.Errorf("Cygnus asset quantity mismatch: expected %d, got %d", cygnusAssetIn.Quantity(), cygnusAssetOut.Quantity())
	}

	// Verify assets and items in legend compartment
	legendAssetIn := legendIn.Assets()[0]
	legendAssetOut := legendOut.Assets()[0]

	if legendAssetIn.TemplateId() != legendAssetOut.TemplateId() {
		t.Errorf("Legend asset templateId mismatch: expected %d, got %d", legendAssetIn.TemplateId(), legendAssetOut.TemplateId())
	}

	if legendAssetIn.Quantity() != legendAssetOut.Quantity() {
		t.Errorf("Legend asset quantity mismatch: expected %d, got %d", legendAssetIn.Quantity(), legendAssetOut.Quantity())
	}
}
