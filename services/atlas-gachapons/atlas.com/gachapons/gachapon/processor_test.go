package gachapon_test

import (
	"atlas-gachapons/gachapon"
	"atlas-gachapons/test"
	"testing"
)

func TestGachaponProcessorCRUD(t *testing.T) {
	processor, db, cleanup := test.CreateGachaponProcessor(t)
	defer cleanup()

	tenantId := test.TestTenantId

	t.Run("Create and GetById", func(t *testing.T) {
		// Create a gachapon
		model, err := gachapon.NewBuilder(tenantId, "crud-test-1").
			SetName("CRUD Test Gachapon").
			SetNpcIds([]uint32{9100100, 9100101}).
			SetCommonWeight(70).
			SetUncommonWeight(25).
			SetRareWeight(5).
			Build()
		if err != nil {
			t.Fatalf("Failed to build gachapon: %v", err)
		}

		err = processor.Create(model)
		if err != nil {
			t.Fatalf("Failed to create gachapon: %v", err)
		}

		// Get by ID
		retrieved, err := processor.GetById("crud-test-1")
		if err != nil {
			t.Fatalf("Failed to get gachapon by ID: %v", err)
		}

		if retrieved.Id() != "crud-test-1" {
			t.Errorf("Expected ID 'crud-test-1', got '%s'", retrieved.Id())
		}
		if retrieved.Name() != "CRUD Test Gachapon" {
			t.Errorf("Expected name 'CRUD Test Gachapon', got '%s'", retrieved.Name())
		}
		if len(retrieved.NpcIds()) != 2 {
			t.Errorf("Expected 2 NPC IDs, got %d", len(retrieved.NpcIds()))
		}
		if retrieved.CommonWeight() != 70 {
			t.Errorf("Expected common weight 70, got %d", retrieved.CommonWeight())
		}
	})

	t.Run("GetAll", func(t *testing.T) {
		// Create another gachapon
		model, err := gachapon.NewBuilder(tenantId, "crud-test-2").
			SetName("Second Gachapon").
			SetNpcIds([]uint32{9100102}).
			SetCommonWeight(50).
			SetUncommonWeight(40).
			SetRareWeight(10).
			Build()
		if err != nil {
			t.Fatalf("Failed to build gachapon: %v", err)
		}

		err = processor.Create(model)
		if err != nil {
			t.Fatalf("Failed to create second gachapon: %v", err)
		}

		// Get all
		all, err := processor.GetAll()()
		if err != nil {
			t.Fatalf("Failed to get all gachapons: %v", err)
		}

		if len(all) < 2 {
			t.Errorf("Expected at least 2 gachapons, got %d", len(all))
		}
	})

	t.Run("Update", func(t *testing.T) {
		// Update the first gachapon
		err := processor.Update("crud-test-1", "Updated Name", 60, 30, 10)
		if err != nil {
			t.Fatalf("Failed to update gachapon: %v", err)
		}

		// Verify update
		updated, err := processor.GetById("crud-test-1")
		if err != nil {
			t.Fatalf("Failed to get updated gachapon: %v", err)
		}

		if updated.Name() != "Updated Name" {
			t.Errorf("Expected name 'Updated Name', got '%s'", updated.Name())
		}
		if updated.CommonWeight() != 60 {
			t.Errorf("Expected common weight 60, got %d", updated.CommonWeight())
		}
		if updated.UncommonWeight() != 30 {
			t.Errorf("Expected uncommon weight 30, got %d", updated.UncommonWeight())
		}
		if updated.RareWeight() != 10 {
			t.Errorf("Expected rare weight 10, got %d", updated.RareWeight())
		}
	})

	t.Run("Delete", func(t *testing.T) {
		// Delete the second gachapon
		err := processor.Delete("crud-test-2")
		if err != nil {
			t.Fatalf("Failed to delete gachapon: %v", err)
		}

		// Verify deletion
		_, err = processor.GetById("crud-test-2")
		if err == nil {
			t.Error("Expected error when getting deleted gachapon, got nil")
		}
	})

	t.Run("GetById NotFound", func(t *testing.T) {
		_, err := processor.GetById("non-existent")
		if err == nil {
			t.Error("Expected error when getting non-existent gachapon, got nil")
		}
	})

	t.Run("BulkCreate", func(t *testing.T) {
		// Create multiple gachapons
		models := make([]gachapon.Model, 3)
		for i := 0; i < 3; i++ {
			m, err := gachapon.NewBuilder(tenantId, "bulk-test-"+string(rune('A'+i))).
				SetName("Bulk Gachapon " + string(rune('A'+i))).
				SetNpcIds([]uint32{uint32(9100200 + i)}).
				SetCommonWeight(70).
				SetUncommonWeight(25).
				SetRareWeight(5).
				Build()
			if err != nil {
				t.Fatalf("Failed to build bulk gachapon %d: %v", i, err)
			}
			models[i] = m
		}

		err := gachapon.BulkCreateGachapon(db, models)
		if err != nil {
			t.Fatalf("Failed to bulk create gachapons: %v", err)
		}

		// Verify all were created
		for i := 0; i < 3; i++ {
			_, err := processor.GetById("bulk-test-" + string(rune('A'+i)))
			if err != nil {
				t.Errorf("Failed to get bulk-created gachapon %d: %v", i, err)
			}
		}
	})
}
