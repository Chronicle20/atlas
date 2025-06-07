package shops_test

import (
	"atlas-npc/commodities"
	"atlas-npc/data/consumable"
	"atlas-npc/shops"
	"context"
	"encoding/json"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockConsumableCache is a mock implementation of the shops.ConsumableCacheInterface
type mockConsumableCache struct {
	consumables map[uuid.UUID][]consumable.Model
}

// GetConsumables returns the rechargeable consumables for a tenant
func (c *mockConsumableCache) GetConsumables(l logrus.FieldLogger, ctx context.Context, tenantId uuid.UUID) []consumable.Model {
	if consumables, ok := c.consumables[tenantId]; ok {
		return consumables
	}
	return []consumable.Model{}
}

// SetConsumables sets the rechargeable consumables for a tenant
func (c *mockConsumableCache) SetConsumables(tenantId uuid.UUID, consumables []consumable.Model) {
	c.consumables[tenantId] = consumables
}

// testLogger creates a logger for testing
func testLogger() logrus.FieldLogger {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	return logger
}

// GetServer creates a ServerInformation object for testing
func GetServer() jsonapi.ServerInformation {
	return &testServerInformation{}
}

// testServerInformation implements the jsonapi.ServerInformation interface
type testServerInformation struct{}

func (s *testServerInformation) GetBaseURL() string {
	return "http://localhost:8080"
}

func (s *testServerInformation) GetPrefix() string {
	return "api"
}

// TestShopRestModel tests the shop REST model
func TestShopRestModel(t *testing.T) {
	// Save the original cache instance
	originalCache := shops.GetConsumableCache()

	// Create a mock cache
	mockCache := &mockConsumableCache{
		consumables: make(map[uuid.UUID][]consumable.Model),
	}

	// Replace the singleton instance with the mock
	shops.SetConsumableCacheForTesting(mockCache)

	// Restore the original cache instance after the test
	defer func() {
		shops.SetConsumableCacheForTesting(originalCache)
	}()
	// Create a shop model with commodities
	npcId := uint32(9000001)

	// Create commodity 1
	commodityId1 := uuid.New()
	commodity1 := (&commodities.ModelBuilder{}).
		SetId(commodityId1).
		SetNpcId(npcId).
		SetTemplateId(2000).
		SetMesoPrice(1000).
		SetDiscountRate(0).
		SetTokenTemplateId(0).
		SetTokenPrice(0).
		SetPeriod(0).
		SetLevelLimit(0).
		SetUnitPrice(1.0).
		SetSlotMax(100).
		Build()

	// Create commodity 2
	commodityId2 := uuid.New()
	commodity2 := (&commodities.ModelBuilder{}).
		SetId(commodityId2).
		SetNpcId(npcId).
		SetTemplateId(2001).
		SetMesoPrice(1500).
		SetDiscountRate(0).
		SetTokenTemplateId(0).
		SetTokenPrice(0).
		SetPeriod(0).
		SetLevelLimit(0).
		SetUnitPrice(1.0).
		SetSlotMax(100).
		Build()

	// Create shop model with recharger flag set to true
	shopModel := shops.NewBuilder(npcId).
		SetCommodities([]commodities.Model{commodity1, commodity2}).
		SetRecharger(true).
		Build()

	// Transform the model to a RestModel
	restModel, err := shops.Transform(shopModel)
	if err != nil {
		t.Fatalf("Failed to transform shop model to REST model: %v", err)
	}

	// Marshal a server response
	rr := httptest.NewRecorder()
	server.MarshalResponse[shops.RestModel](testLogger())(rr)(GetServer())(make(map[string][]string))(restModel)

	if rr.Code != http.StatusOK {
		t.Fatalf("Failed to write rest model: %v", err)
	}

	// Unmarshal the server response
	body := rr.Body.Bytes()

	// Print the response body for debugging
	t.Logf("Response body: %s", string(body))

	// Parse the JSON response manually to extract the commodity data
	var jsonResponse map[string]interface{}
	err = json.Unmarshal(body, &jsonResponse)
	if err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Extract the included data (commodities)
	included, ok := jsonResponse["included"].([]interface{})
	if !ok {
		t.Fatalf("Failed to extract included data from response")
	}

	// Create a map of commodity IDs to commodity data
	commodityMap := make(map[string]commodities.RestModel)
	for _, item := range included {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		// Check if this is a commodity
		itemType, ok := itemMap["type"].(string)
		if !ok || itemType != "commodities" {
			continue
		}

		// Extract the commodity ID
		id, ok := itemMap["id"].(string)
		if !ok {
			continue
		}

		// Extract the attributes
		attrs, ok := itemMap["attributes"].(map[string]interface{})
		if !ok {
			continue
		}

		// Create a commodity RestModel
		commodity := commodities.RestModel{}
		commodity.SetID(id)

		// Set the attributes
		if templateId, ok := attrs["templateId"].(float64); ok {
			commodity.TemplateId = uint32(templateId)
		}
		if mesoPrice, ok := attrs["mesoPrice"].(float64); ok {
			commodity.MesoPrice = uint32(mesoPrice)
		}
		if discountRate, ok := attrs["discountRate"].(float64); ok {
			commodity.DiscountRate = byte(discountRate)
		}
		if tokenTemplateId, ok := attrs["tokenTemplateId"].(float64); ok {
			commodity.TokenTemplateId = uint32(tokenTemplateId)
		}
		if tokenPrice, ok := attrs["tokenPrice"].(float64); ok {
			commodity.TokenPrice = uint32(tokenPrice)
		}
		if period, ok := attrs["period"].(float64); ok {
			commodity.Period = uint32(period)
		}
		if levelLimit, ok := attrs["levelLimit"].(float64); ok {
			commodity.LevelLimit = uint32(levelLimit)
		}
		if unitPrice, ok := attrs["unitPrice"].(float64); ok {
			commodity.UnitPrice = unitPrice
		}
		if slotMax, ok := attrs["slotMax"].(float64); ok {
			commodity.SlotMax = uint32(slotMax)
		}

		// Add the commodity to the map
		commodityMap[id] = commodity
	}

	// Unmarshal the shop data
	unmarshaledRestModel := shops.RestModel{}
	err = jsonapi.Unmarshal(body, &unmarshaledRestModel)
	if err != nil {
		t.Fatalf("Failed to unmarshal rest model: %v", err)
	}

	// Print the unmarshaled model for debugging
	t.Logf("Unmarshaled model: %+v", unmarshaledRestModel)
	t.Logf("Unmarshaled commodities: %+v", unmarshaledRestModel.Commodities)

	// Manually set the commodity data in the unmarshaled model
	commodities := make([]commodities.RestModel, 0)
	for _, c := range unmarshaledRestModel.Commodities {
		if commodity, ok := commodityMap[c.GetID()]; ok {
			commodities = append(commodities, commodity)
		} else {
			commodities = append(commodities, c)
		}
	}
	unmarshaledRestModel.Commodities = commodities

	t.Logf("Updated commodities: %+v", unmarshaledRestModel.Commodities)

	// Extract the model from the RestModel
	extractedModel, err := shops.Extract(unmarshaledRestModel)
	if err != nil {
		t.Fatalf("Failed to extract model from REST model: %v", err)
	}

	// Verify the contents match the initial model
	if extractedModel.NpcId() != shopModel.NpcId() {
		t.Errorf("Expected NPC ID %d, got %d", shopModel.NpcId(), extractedModel.NpcId())
	}

	// Verify the recharger flag
	if extractedModel.Recharger() != shopModel.Recharger() {
		t.Errorf("Expected Recharger %v, got %v", shopModel.Recharger(), extractedModel.Recharger())
	}

	if len(extractedModel.Commodities()) != len(shopModel.Commodities()) {
		t.Errorf("Expected %d commodities, got %d", len(shopModel.Commodities()), len(extractedModel.Commodities()))
	}

	// Verify each commodity
	for i, c := range extractedModel.Commodities() {
		originalCommodity := shopModel.Commodities()[i]

		if c.TemplateId() != originalCommodity.TemplateId() {
			t.Errorf("Commodity %d: Expected template ID %d, got %d", i, originalCommodity.TemplateId(), c.TemplateId())
		}

		if c.MesoPrice() != originalCommodity.MesoPrice() {
			t.Errorf("Commodity %d: Expected meso price %d, got %d", i, originalCommodity.MesoPrice(), c.MesoPrice())
		}

		if c.UnitPrice() != originalCommodity.UnitPrice() {
			t.Errorf("Commodity %d: Expected unit price %f, got %f", i, originalCommodity.UnitPrice(), c.UnitPrice())
		}

		if c.SlotMax() != originalCommodity.SlotMax() {
			t.Errorf("Commodity %d: Expected slot max %d, got %d", i, originalCommodity.SlotMax(), c.SlotMax())
		}
	}
}
