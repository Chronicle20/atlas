package factory

import (
	"atlas-character-factory/configuration/tenant/characters/template"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestFollowUpSagaTemplateStoreSingleton(t *testing.T) {
	t.Run("singleton_returns_same_instance", func(t *testing.T) {
		store1 := GetFollowUpSagaTemplateStore()
		store2 := GetFollowUpSagaTemplateStore()
		
		if store1 != store2 {
			t.Errorf("GetFollowUpSagaTemplateStore() should return the same instance, got different instances")
		}
	})
	
	t.Run("singleton_concurrent_access", func(t *testing.T) {
		var wg sync.WaitGroup
		instances := make([]*FollowUpSagaTemplateStore, 100)
		
		// Create 100 goroutines that get the singleton instance
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				instances[index] = GetFollowUpSagaTemplateStore()
			}(i)
		}
		
		wg.Wait()
		
		// All instances should be the same
		firstInstance := instances[0]
		for i := 1; i < 100; i++ {
			if instances[i] != firstInstance {
				t.Errorf("All instances should be the same, but instance %d is different", i)
			}
		}
	})
}

func TestFollowUpSagaTemplateStoreThreadSafety(t *testing.T) {
	store := GetFollowUpSagaTemplateStore()
	store.Clear() // Start with clean state
	
	t.Run("concurrent_read_write", func(t *testing.T) {
		var wg sync.WaitGroup
		tenantId := uuid.New()
		
		// Create sample data
		input := RestModel{
			AccountId:   1001,
			Name:        "TestCharacter",
			WorldId:     0,
			Gender:      0,
			JobIndex:    100,
			SubJobIndex: 0,
		}
		
		templateData := template.RestModel{
			JobIndex:    100,
			SubJobIndex: 0,
			Items:       []uint32{1000, 2000, 3000},
			Skills:      []uint32{100, 200},
		}
		
		// Start 50 goroutines writing different character names
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				characterName := fmt.Sprintf("Character_%d", index)
				err := store.Store(tenantId, characterName, input, templateData)
				if err != nil {
					t.Errorf("Error storing template: %v", err)
				}
			}(i)
		}
		
		// Start 50 goroutines reading different character names
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				characterName := fmt.Sprintf("Character_%d", index)
				// Small delay to let some writes happen first
				time.Sleep(time.Millisecond)
				_, exists := store.Get(tenantId, characterName)
				// We don't assert on exists because of timing, just ensure no panic
				_ = exists
			}(i)
		}
		
		wg.Wait()
		
		// Verify all 50 templates were stored
		if store.Size() != 50 {
			t.Errorf("Expected 50 templates stored, got %d", store.Size())
		}
	})
	
	t.Run("concurrent_remove_operations", func(t *testing.T) {
		store.Clear()
		var wg sync.WaitGroup
		tenantId := uuid.New()
		
		// Store 100 templates
		for i := 0; i < 100; i++ {
			characterName := fmt.Sprintf("Character_%d", i)
			input := RestModel{Name: characterName}
			templateData := template.RestModel{Items: []uint32{uint32(i)}}
			store.Store(tenantId, characterName, input, templateData)
		}
		
		// Verify all stored
		if store.Size() != 100 {
			t.Errorf("Expected 100 templates stored, got %d", store.Size())
		}
		
		// Remove all templates concurrently
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				characterName := fmt.Sprintf("Character_%d", index)
				store.Remove(tenantId, characterName)
			}(i)
		}
		
		wg.Wait()
		
		// Verify all removed
		if store.Size() != 0 {
			t.Errorf("Expected 0 templates after removal, got %d", store.Size())
		}
	})
}

func TestFollowUpSagaTemplateStoreTenantIsolation(t *testing.T) {
	store := GetFollowUpSagaTemplateStore()
	store.Clear()
	
	tenant1 := uuid.New()
	tenant2 := uuid.New()
	characterName := "TestCharacter"
	
	input1 := RestModel{AccountId: 1001, Name: characterName}
	input2 := RestModel{AccountId: 2002, Name: characterName}
	
	template1 := template.RestModel{Items: []uint32{1000}}
	template2 := template.RestModel{Items: []uint32{2000}}
	
	// Store templates for both tenants with same character name
	store.Store(tenant1, characterName, input1, template1)
	store.Store(tenant2, characterName, input2, template2)
	
	// Verify both stored
	if store.Size() != 2 {
		t.Errorf("Expected 2 templates stored, got %d", store.Size())
	}
	
	// Verify tenant isolation
	result1, exists1 := store.Get(tenant1, characterName)
	result2, exists2 := store.Get(tenant2, characterName)
	
	if !exists1 || !exists2 {
		t.Errorf("Both templates should exist")
	}
	
	if result1.Input.AccountId != 1001 {
		t.Errorf("Tenant 1 should have AccountId 1001, got %d", result1.Input.AccountId)
	}
	
	if result2.Input.AccountId != 2002 {
		t.Errorf("Tenant 2 should have AccountId 2002, got %d", result2.Input.AccountId)
	}
	
	// Remove tenant 1's template
	store.Remove(tenant1, characterName)
	
	// Verify tenant 1's template is gone but tenant 2's remains
	_, exists1After := store.Get(tenant1, characterName)
	_, exists2After := store.Get(tenant2, characterName)
	
	if exists1After {
		t.Errorf("Tenant 1's template should be removed")
	}
	
	if !exists2After {
		t.Errorf("Tenant 2's template should still exist")
	}
}

func TestFollowUpSagaTemplateStoreHelperFunctions(t *testing.T) {
	store := GetFollowUpSagaTemplateStore()
	store.Clear()
	
	// Test that helper functions use the singleton
	tenantId := uuid.New()
	characterName := "TestCharacter"
	
	input := RestModel{AccountId: 1001, Name: characterName}
	templateData := template.RestModel{Items: []uint32{1000}}
	
	// Store via helper function should work
	// Note: We can't easily test storeFollowUpSagaTemplate without a context,
	// but we can test the Get and Remove helper functions
	
	// Store directly
	store.Store(tenantId, characterName, input, templateData)
	
	// Get via helper function
	result, exists := GetFollowUpSagaTemplate(tenantId, characterName)
	if !exists {
		t.Errorf("Template should exist")
	}
	
	if result.Input.AccountId != 1001 {
		t.Errorf("Expected AccountId 1001, got %d", result.Input.AccountId)
	}
	
	// Remove via helper function
	RemoveFollowUpSagaTemplate(tenantId, characterName)
	
	// Verify removed
	_, existsAfter := GetFollowUpSagaTemplate(tenantId, characterName)
	if existsAfter {
		t.Errorf("Template should be removed")
	}
}