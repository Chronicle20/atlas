package commodities_test

import (
	"atlas-npc/commodities"
	"atlas-npc/test"
	"gorm.io/gorm"
	"testing"
)

func TestCommoditiesProcessor(t *testing.T) {
	// Create processor, database, and cleanup function
	processor, db, cleanup := test.CreateCommoditiesProcessor(t)
	defer cleanup()

	// Run tests
	t.Run("TestCreateCommodity", func(t *testing.T) {
		testCreateCommodity(t, processor, db)
	})

	t.Run("TestGetByNpcId", func(t *testing.T) {
		testGetByNpcId(t, processor, db)
	})

	t.Run("TestUpdateCommodity", func(t *testing.T) {
		testUpdateCommodity(t, processor, db)
	})

	t.Run("TestDeleteCommodity", func(t *testing.T) {
		testDeleteCommodity(t, processor, db)
	})
}

func testCreateCommodity(t *testing.T, processor commodities.Processor, db *gorm.DB) {
	// Test data
	npcId := uint32(1001)
	templateId := uint32(2001)
	mesoPrice := uint32(1000)
	tokenPrice := uint32(500)

	// Create commodity
	// Default values for new fields
	discountRate := byte(0)
	tokenItemId := uint32(0)
	period := uint32(0)
	levelLimited := uint32(0)
	commodity, err := processor.CreateCommodity(npcId, templateId, mesoPrice, discountRate, tokenItemId, tokenPrice, period, levelLimited)
	if err != nil {
		t.Fatalf("Failed to create commodity: %v", err)
	}

	// Verify commodity was created
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
	result := db.Where("npc_id = ?", npcId).First(&entity)
	if result.Error != nil {
		t.Fatalf("Failed to find commodity in database: %v", result.Error)
	}
	if entity.TemplateId != templateId {
		t.Errorf("Expected template ID %d, got %d", templateId, entity.TemplateId)
	}
}

func testGetByNpcId(t *testing.T, processor commodities.Processor, db *gorm.DB) {
	// Test data
	npcId := uint32(1002)
	templateId := uint32(2002)
	mesoPrice := uint32(2000)
	tokenPrice := uint32(1000)

	// Create test commodity
	// Default values for new fields
	discountRate := byte(0)
	tokenItemId := uint32(0)
	period := uint32(0)
	levelLimited := uint32(0)
	_, err := processor.CreateCommodity(npcId, templateId, mesoPrice, discountRate, tokenItemId, tokenPrice, period, levelLimited)
	if err != nil {
		t.Fatalf("Failed to create test commodity: %v", err)
	}

	// Get commodities by NPC ID
	commodities, err := processor.GetByNpcId(npcId)
	if err != nil {
		t.Fatalf("Failed to get commodities by NPC ID: %v", err)
	}

	// Verify commodities
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

func testUpdateCommodity(t *testing.T, processor commodities.Processor, db *gorm.DB) {
	// Test data
	npcId := uint32(1003)
	templateId := uint32(2003)
	mesoPrice := uint32(3000)
	tokenPrice := uint32(1500)

	// Updated values
	updatedTemplateId := uint32(2004)
	updatedMesoPrice := uint32(3500)
	updatedTokenPrice := uint32(1750)

	// Create test commodity
	// Default values for new fields
	discountRate := byte(0)
	tokenItemId := uint32(0)
	period := uint32(0)
	levelLimited := uint32(0)
	commodity, err := processor.CreateCommodity(npcId, templateId, mesoPrice, discountRate, tokenItemId, tokenPrice, period, levelLimited)
	if err != nil {
		t.Fatalf("Failed to create test commodity: %v", err)
	}

	// Update commodity
	// Default values for new fields in update
	updatedDiscountRate := byte(0)
	updatedTokenItemId := uint32(0)
	updatedPeriod := uint32(0)
	updatedLevelLimited := uint32(0)
	updatedCommodity, err := processor.UpdateCommodity(commodity.Id(), updatedTemplateId, updatedMesoPrice, updatedDiscountRate, updatedTokenItemId, updatedTokenPrice, updatedPeriod, updatedLevelLimited)
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

func testDeleteCommodity(t *testing.T, processor commodities.Processor, db *gorm.DB) {
	// Test data
	npcId := uint32(1004)
	templateId := uint32(2005)
	mesoPrice := uint32(4000)
	tokenPrice := uint32(2000)

	// Create test commodity
	// Default values for new fields
	discountRate := byte(0)
	tokenItemId := uint32(0)
	period := uint32(0)
	levelLimited := uint32(0)
	commodity, err := processor.CreateCommodity(npcId, templateId, mesoPrice, discountRate, tokenItemId, tokenPrice, period, levelLimited)
	if err != nil {
		t.Fatalf("Failed to create test commodity: %v", err)
	}

	// Delete commodity
	err = processor.DeleteCommodity(commodity.Id())
	if err != nil {
		t.Fatalf("Failed to delete commodity: %v", err)
	}

	// Verify commodity was deleted from database
	var entity commodities.Entity
	result := db.Where("id = ?", commodity.Id()).First(&entity)
	if result.Error == nil || result.Error.Error() != "record not found" {
		t.Errorf("Expected commodity to be deleted, but it still exists")
	}
}