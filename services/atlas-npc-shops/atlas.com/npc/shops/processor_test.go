package shops_test

import (
	"atlas-npc/commodities"
	"atlas-npc/shops"
	"atlas-npc/test"
	"gorm.io/gorm"
	"testing"
)

func TestShopsProcessor(t *testing.T) {
	// Create processor, database, and cleanup function
	processor, db, cleanup := test.CreateShopsProcessor(t)
	defer cleanup()

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
