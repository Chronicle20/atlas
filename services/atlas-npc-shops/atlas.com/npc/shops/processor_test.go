package shops_test

import (
	"atlas-npc/commodities"
	"atlas-npc/data/consumable"
	"atlas-npc/shops"
	"atlas-npc/test"
	"context"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"testing"
)

// mockConsumableCache is a mock implementation of the ConsumableCacheInterface
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

// originalCache stores the original cache instance
var originalCache shops.ConsumableCacheInterface

func TestShopsProcessor(t *testing.T) {
	// Create processor, database, and cleanup function
	processor, db, cleanup := test.CreateShopsProcessor(t)
	defer cleanup()

	// Mock the processor's RechargeableConsumablesDecorator method to do nothing
	if p, ok := processor.(*shops.ProcessorImpl); ok {
		p.RechargeableConsumablesDecoratorFn = func(m shops.Model) shops.Model {
			return m
		}
	}

	// Run tests
	t.Run("TestGetByNpcId", func(t *testing.T) {
		testGetByNpcId(t, processor, db)
	})

	t.Run("TestAddCommodity", func(t *testing.T) {
		testAddCommodity(t, processor, db)
	})

	t.Run("TestUpdateCommodity", func(t *testing.T) {
		testUpdateCommodity(t, processor, db)
	})

	t.Run("TestRemoveCommodity", func(t *testing.T) {
		testRemoveCommodity(t, processor, db)
	})

	t.Run("TestCreateShop", func(t *testing.T) {
		testCreateShop(t, processor, db)
	})

	t.Run("TestUpdateShop", func(t *testing.T) {
		testUpdateShop(t, processor, db)
	})

	t.Run("TestDeleteAllShops", func(t *testing.T) {
		testDeleteAllShops(t, processor, db)
	})
}

func testGetByNpcId(t *testing.T, processor shops.Processor, db *gorm.DB) {
	// Test data
	npcId := uint32(2001)
	templateId := uint32(3001)
	mesoPrice := uint32(5000)
	tokenPrice := uint32(2500)

	// Create test commodity for the shop
	// Default values for new fields
	discountRate := byte(0)
	tokenTemplateId := uint32(0)
	period := uint32(0)
	levelLimited := uint32(0)
	_, err := processor.AddCommodity(npcId, templateId, mesoPrice, discountRate, tokenTemplateId, tokenPrice, period, levelLimited)
	if err != nil {
		t.Fatalf("Failed to create test commodity: %v", err)
	}

	// Get shop by NPC ID
	shop, err := processor.GetByNpcId(processor.CommodityDecorator)(npcId)
	if err != nil {
		t.Fatalf("Failed to get shop by NPC ID: %v", err)
	}

	// Verify shop
	if shop.NpcId() != npcId {
		t.Errorf("Expected NPC ID %d, got %d", npcId, shop.NpcId())
	}

	// Verify shop commodities
	commodities := shop.Commodities()
	if len(commodities) == 0 {
		t.Fatalf("Expected at least one commodity, got none")
	}

	found := false
	for _, c := range commodities {
		if c.TemplateId() == templateId {
			found = true
			if c.MesoPrice() != mesoPrice {
				t.Errorf("Expected meso price %d, got %d", mesoPrice, c.MesoPrice())
			}
			if c.TokenPrice() != tokenPrice {
				t.Errorf("Expected token price %d, got %d", tokenPrice, c.TokenPrice())
			}
		}
	}

	if !found {
		t.Errorf("Could not find commodity with template ID %d", templateId)
	}
}

func testAddCommodity(t *testing.T, processor shops.Processor, db *gorm.DB) {
	// Test data
	npcId := uint32(2002)
	templateId := uint32(3002)
	mesoPrice := uint32(6000)
	tokenPrice := uint32(3000)

	// Add commodity to shop
	// Default values for new fields
	discountRate := byte(0)
	tokenTemplateId := uint32(0)
	period := uint32(0)
	levelLimited := uint32(0)
	commodity, err := processor.AddCommodity(npcId, templateId, mesoPrice, discountRate, tokenTemplateId, tokenPrice, period, levelLimited)
	if err != nil {
		t.Fatalf("Failed to add commodity to shop: %v", err)
	}

	// Verify commodity was added
	if commodity.TemplateId() != templateId {
		t.Errorf("Expected template ID %d, got %d", templateId, commodity.TemplateId())
	}
	if commodity.MesoPrice() != mesoPrice {
		t.Errorf("Expected meso price %d, got %d", mesoPrice, commodity.MesoPrice())
	}
	if commodity.TokenPrice() != tokenPrice {
		t.Errorf("Expected token price %d, got %d", tokenPrice, commodity.TokenPrice())
	}

	// Verify commodity exists in database
	var entity commodities.Entity
	result := db.Where("npc_id = ? AND template_id = ?", npcId, templateId).First(&entity)
	if result.Error != nil {
		t.Fatalf("Failed to find commodity in database: %v", result.Error)
	}
	if entity.TemplateId != templateId {
		t.Errorf("Expected template ID %d, got %d", templateId, entity.TemplateId)
	}
}

func testUpdateCommodity(t *testing.T, processor shops.Processor, db *gorm.DB) {
	// Test data
	npcId := uint32(2003)
	templateId := uint32(3003)
	mesoPrice := uint32(7000)
	tokenPrice := uint32(3500)

	// Updated values
	updatedTemplateId := uint32(3004)
	updatedMesoPrice := uint32(7500)
	updatedTokenPrice := uint32(3750)

	// Add commodity to shop
	// Default values for new fields
	discountRate := byte(0)
	tokenTemplateId := uint32(0)
	period := uint32(0)
	levelLimited := uint32(0)
	commodity, err := processor.AddCommodity(npcId, templateId, mesoPrice, discountRate, tokenTemplateId, tokenPrice, period, levelLimited)
	if err != nil {
		t.Fatalf("Failed to add test commodity: %v", err)
	}

	// Update commodity
	// Default values for new fields in update
	updatedDiscountRate := byte(0)
	updatedTokenTemplateId := uint32(0)
	updatedPeriod := uint32(0)
	updatedLevelLimited := uint32(0)
	updatedCommodity, err := processor.UpdateCommodity(commodity.Id(), updatedTemplateId, updatedMesoPrice, updatedDiscountRate, updatedTokenTemplateId, updatedTokenPrice, updatedPeriod, updatedLevelLimited)
	if err != nil {
		t.Fatalf("Failed to update commodity: %v", err)
	}

	// Verify updated commodity
	if updatedCommodity.TemplateId() != updatedTemplateId {
		t.Errorf("Expected template ID %d, got %d", updatedTemplateId, updatedCommodity.TemplateId())
	}
	if updatedCommodity.MesoPrice() != updatedMesoPrice {
		t.Errorf("Expected meso price %d, got %d", updatedMesoPrice, updatedCommodity.MesoPrice())
	}
	if updatedCommodity.TokenPrice() != updatedTokenPrice {
		t.Errorf("Expected token price %d, got %d", updatedTokenPrice, updatedCommodity.TokenPrice())
	}

	// Verify commodity was updated in database
	var entity commodities.Entity
	result := db.Where("id = ?", commodity.Id()).First(&entity)
	if result.Error != nil {
		t.Fatalf("Failed to find commodity in database: %v", result.Error)
	}
	if entity.TemplateId != updatedTemplateId {
		t.Errorf("Expected template ID %d, got %d", updatedTemplateId, entity.TemplateId)
	}
	if entity.MesoPrice != updatedMesoPrice {
		t.Errorf("Expected meso price %d, got %d", updatedMesoPrice, entity.MesoPrice)
	}
	if entity.TokenPrice != updatedTokenPrice {
		t.Errorf("Expected token price %d, got %d", updatedTokenPrice, entity.TokenPrice)
	}
}

func testRemoveCommodity(t *testing.T, processor shops.Processor, db *gorm.DB) {
	// Test data
	npcId := uint32(2004)
	templateId := uint32(3005)
	mesoPrice := uint32(8000)
	tokenPrice := uint32(4000)

	// Add commodity to shop
	// Default values for new fields
	discountRate := byte(0)
	tokenTemplateId := uint32(0)
	period := uint32(0)
	levelLimited := uint32(0)
	commodity, err := processor.AddCommodity(npcId, templateId, mesoPrice, discountRate, tokenTemplateId, tokenPrice, period, levelLimited)
	if err != nil {
		t.Fatalf("Failed to add test commodity: %v", err)
	}

	// Remove commodity
	err = processor.RemoveCommodity(commodity.Id())
	if err != nil {
		t.Fatalf("Failed to remove commodity: %v", err)
	}

	// Verify commodity was removed from database
	var entity commodities.Entity
	result := db.Where("id = ?", commodity.Id()).First(&entity)
	if result.Error == nil || result.Error.Error() != "record not found" {
		t.Errorf("Expected commodity to be removed, but it still exists")
	}
}

func testCreateShop(t *testing.T, processor shops.Processor, db *gorm.DB) {
	// Test data
	npcId := uint32(2005)
	templateId := uint32(3006)
	mesoPrice := uint32(9000)
	tokenPrice := uint32(4500)
	recharger := true // Test with recharger set to true

	// Create a commodity for the shop
	discountRate := byte(0)
	tokenTemplateId := uint32(0)
	period := uint32(0)
	levelLimited := uint32(0)
	commodity, err := processor.AddCommodity(npcId, templateId, mesoPrice, discountRate, tokenTemplateId, tokenPrice, period, levelLimited)
	if err != nil {
		t.Fatalf("Failed to create test commodity: %v", err)
	}

	// Create shop with the commodity and recharger value
	shop, err := processor.CreateShop(npcId, recharger, []commodities.Model{commodity})
	if err != nil {
		t.Fatalf("Failed to create shop: %v", err)
	}

	// Verify shop was created with correct values
	if shop.NpcId() != npcId {
		t.Errorf("Expected NPC ID %d, got %d", npcId, shop.NpcId())
	}
	if shop.Recharger() != recharger {
		t.Errorf("Expected Recharger %v, got %v", recharger, shop.Recharger())
	}
	if len(shop.Commodities()) != 1 {
		t.Errorf("Expected 1 commodity, got %d", len(shop.Commodities()))
	}

	// Verify shop entity exists in database
	var count int64
	result := db.Model(&shops.Entity{}).Where("npc_id = ?", npcId).Count(&count)
	if result.Error != nil {
		t.Fatalf("Failed to count shop entities in database: %v", result.Error)
	}
	if count != 1 {
		t.Errorf("Expected 1 shop entity in database, got %d", count)
	}

	// Test with recharger set to false
	npcId = uint32(2006)
	recharger = false

	// Create another commodity for the second shop
	commodity, err = processor.AddCommodity(npcId, templateId, mesoPrice, discountRate, tokenTemplateId, tokenPrice, period, levelLimited)
	if err != nil {
		t.Fatalf("Failed to create test commodity: %v", err)
	}

	// Create shop with recharger set to false
	shop, err = processor.CreateShop(npcId, recharger, []commodities.Model{commodity})
	if err != nil {
		t.Fatalf("Failed to create shop: %v", err)
	}

	// Verify shop was created with correct values
	if shop.Recharger() != recharger {
		t.Errorf("Expected Recharger %v, got %v", recharger, shop.Recharger())
	}

	// Verify shop entity exists in database
	count = 0
	result = db.Model(&shops.Entity{}).Where("npc_id = ?", npcId).Count(&count)
	if result.Error != nil {
		t.Fatalf("Failed to count shop entities in database: %v", result.Error)
	}
	if count != 1 {
		t.Errorf("Expected 1 shop entity in database, got %d", count)
	}
}

func testUpdateShop(t *testing.T, processor shops.Processor, db *gorm.DB) {
	// Test data
	npcId := uint32(2007)
	templateId := uint32(3007)
	mesoPrice := uint32(10000)
	tokenPrice := uint32(5000)
	initialRecharger := true
	updatedRecharger := false

	// Create a commodity for the shop
	discountRate := byte(0)
	tokenTemplateId := uint32(0)
	period := uint32(0)
	levelLimited := uint32(0)
	commodity, err := processor.AddCommodity(npcId, templateId, mesoPrice, discountRate, tokenTemplateId, tokenPrice, period, levelLimited)
	if err != nil {
		t.Fatalf("Failed to create test commodity: %v", err)
	}

	// Create shop with initial recharger value
	_, err = processor.CreateShop(npcId, initialRecharger, []commodities.Model{commodity})
	if err != nil {
		t.Fatalf("Failed to create shop: %v", err)
	}

	// Update shop with new recharger value
	updatedShop, err := processor.UpdateShop(npcId, updatedRecharger, []commodities.Model{commodity})
	if err != nil {
		t.Fatalf("Failed to update shop: %v", err)
	}

	// Verify shop was updated with correct values
	if updatedShop.NpcId() != npcId {
		t.Errorf("Expected NPC ID %d, got %d", npcId, updatedShop.NpcId())
	}
	if updatedShop.Recharger() != updatedRecharger {
		t.Errorf("Expected Recharger %v, got %v", updatedRecharger, updatedShop.Recharger())
	}
	if len(updatedShop.Commodities()) != 1 {
		t.Errorf("Expected 1 commodity, got %d", len(updatedShop.Commodities()))
	}

	// Verify shop entity exists in database
	var count int64
	result := db.Model(&shops.Entity{}).Where("npc_id = ?", npcId).Count(&count)
	if result.Error != nil {
		t.Fatalf("Failed to count shop entities in database: %v", result.Error)
	}
	if count != 1 {
		t.Errorf("Expected 1 shop entity in database, got %d", count)
	}

	// Verify the recharger value was updated in the database by retrieving the shop again
	retrievedShop, err := processor.GetByNpcId()(npcId)
	if err != nil {
		t.Fatalf("Failed to retrieve shop from database: %v", err)
	}
	if retrievedShop.Recharger() != updatedRecharger {
		t.Errorf("Expected Recharger %v in database, got %v", updatedRecharger, retrievedShop.Recharger())
	}

	// Test updating a non-existent shop (should create a new one)
	npcId = uint32(2008)
	initialRecharger = false
	updatedRecharger = true

	// Create a commodity for the new shop
	commodity, err = processor.AddCommodity(npcId, templateId, mesoPrice, discountRate, tokenTemplateId, tokenPrice, period, levelLimited)
	if err != nil {
		t.Fatalf("Failed to create test commodity: %v", err)
	}

	// Update non-existent shop (should create a new one)
	newShop, err := processor.UpdateShop(npcId, updatedRecharger, []commodities.Model{commodity})
	if err != nil {
		t.Fatalf("Failed to update/create shop: %v", err)
	}

	// Verify shop was created with correct values
	if newShop.NpcId() != npcId {
		t.Errorf("Expected NPC ID %d, got %d", npcId, newShop.NpcId())
	}
	if newShop.Recharger() != updatedRecharger {
		t.Errorf("Expected Recharger %v, got %v", updatedRecharger, newShop.Recharger())
	}

	// Verify shop entity exists in database
	count = 0
	result = db.Model(&shops.Entity{}).Where("npc_id = ?", npcId).Count(&count)
	if result.Error != nil {
		t.Fatalf("Failed to count shop entities in database: %v", result.Error)
	}
	if count != 1 {
		t.Errorf("Expected 1 shop entity in database, got %d", count)
	}

	// Verify the recharger value was set correctly in the database by retrieving the shop again
	retrievedNewShop, err := processor.GetByNpcId()(npcId)
	if err != nil {
		t.Fatalf("Failed to retrieve shop from database: %v", err)
	}
	if retrievedNewShop.Recharger() != updatedRecharger {
		t.Errorf("Expected Recharger %v in database, got %v", updatedRecharger, retrievedNewShop.Recharger())
	}
}

func testDeleteAllShops(t *testing.T, processor shops.Processor, db *gorm.DB) {
	// Test data for first shop
	npcId1 := uint32(3001)
	templateId1 := uint32(4001)
	mesoPrice1 := uint32(5000)
	tokenPrice1 := uint32(2500)
	recharger1 := true

	// Test data for second shop
	npcId2 := uint32(3002)
	templateId2 := uint32(4002)
	mesoPrice2 := uint32(6000)
	tokenPrice2 := uint32(3000)
	recharger2 := false

	// Default values for new fields
	discountRate := byte(0)
	tokenTemplateId := uint32(0)
	period := uint32(0)
	levelLimited := uint32(0)

	// Create commodities for the shops
	commodity1, err := processor.AddCommodity(npcId1, templateId1, mesoPrice1, discountRate, tokenTemplateId, tokenPrice1, period, levelLimited)
	if err != nil {
		t.Fatalf("Failed to create test commodity 1: %v", err)
	}

	commodity2, err := processor.AddCommodity(npcId2, templateId2, mesoPrice2, discountRate, tokenTemplateId, tokenPrice2, period, levelLimited)
	if err != nil {
		t.Fatalf("Failed to create test commodity 2: %v", err)
	}

	// Create shops with the commodities
	_, err = processor.CreateShop(npcId1, recharger1, []commodities.Model{commodity1})
	if err != nil {
		t.Fatalf("Failed to create shop 1: %v", err)
	}

	_, err = processor.CreateShop(npcId2, recharger2, []commodities.Model{commodity2})
	if err != nil {
		t.Fatalf("Failed to create shop 2: %v", err)
	}

	// Verify shops and commodities exist in database
	var shopCount int64
	result := db.Model(&shops.Entity{}).Count(&shopCount)
	if result.Error != nil {
		t.Fatalf("Failed to count shop entities in database: %v", result.Error)
	}
	if shopCount < 2 {
		t.Errorf("Expected at least 2 shop entities in database, got %d", shopCount)
	}

	var commodityCount int64
	result = db.Model(&commodities.Entity{}).Count(&commodityCount)
	if result.Error != nil {
		t.Fatalf("Failed to count commodity entities in database: %v", result.Error)
	}
	if commodityCount < 2 {
		t.Errorf("Expected at least 2 commodity entities in database, got %d", commodityCount)
	}

	// Call DeleteAllShops
	err = processor.DeleteAllShops()
	if err != nil {
		t.Fatalf("Failed to delete all shops: %v", err)
	}

	// Verify all shops are deleted from database
	shopCount = 0
	result = db.Model(&shops.Entity{}).Count(&shopCount)
	if result.Error != nil {
		t.Fatalf("Failed to count shop entities in database after deletion: %v", result.Error)
	}
	if shopCount != 0 {
		t.Errorf("Expected 0 shop entities in database after deletion, got %d", shopCount)
	}

	// Verify all commodities are deleted from database
	commodityCount = 0
	result = db.Model(&commodities.Entity{}).Count(&commodityCount)
	if result.Error != nil {
		t.Fatalf("Failed to count commodity entities in database after deletion: %v", result.Error)
	}
	if commodityCount != 0 {
		t.Errorf("Expected 0 commodity entities in database after deletion, got %d", commodityCount)
	}
}
