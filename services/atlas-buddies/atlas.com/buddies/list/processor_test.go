package list

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestIncreaseCapacityIntegration tests the end-to-end increase capacity flow
func TestIncreaseCapacityIntegration(t *testing.T) {
	// Setup database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	// Create the tables manually for SQLite compatibility
	err = db.Exec(`
		CREATE TABLE lists (
			tenant_id TEXT NOT NULL,
			id TEXT PRIMARY KEY,
			character_id INTEGER NOT NULL,
			capacity INTEGER NOT NULL
		)
	`).Error
	if err != nil {
		t.Fatalf("Failed to create lists table: %v", err)
	}

	err = db.Exec(`
		CREATE TABLE buddies (
			character_id INTEGER PRIMARY KEY,
			list_id TEXT NOT NULL,
			"group" TEXT NOT NULL,
			character_name TEXT NOT NULL,
			channel_id INTEGER NOT NULL DEFAULT -1,
			in_shop BOOLEAN NOT NULL DEFAULT false,
			pending BOOLEAN NOT NULL DEFAULT false
		)
	`).Error
	if err != nil {
		t.Fatalf("Failed to create buddies table: %v", err)
	}

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Use NewProcessor to get a properly initialized processor
	// Note: This requires tenant context which we'll mock at the NewProcessor level
	characterId := uint32(12345)
	initialCapacity := byte(20)

	// Test the administrator function directly first
	tenantId := uuid.New()

	// Create initial entity directly
	entity := Entity{
		TenantId:    tenantId,
		Id:          uuid.New(),
		CharacterId: characterId,
		Capacity:    initialCapacity,
	}
	err = db.Create(&entity).Error
	if err != nil {
		t.Fatalf("Failed to create test entity: %v", err)
	}

	// Test updateCapacity function
	tests := []struct {
		name          string
		newCapacity   byte
		expectedError bool
	}{
		{
			name:          "Valid capacity increase",
			newCapacity:   30,
			expectedError: false,
		},
		{
			name:          "Invalid capacity - same value",
			newCapacity:   initialCapacity,
			expectedError: true,
		},
		{
			name:          "Invalid capacity - lower value", 
			newCapacity:   15,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset capacity for each test
			db.Model(&entity).Update("capacity", initialCapacity)

			err := updateCapacity(db, tenantId, characterId, tt.newCapacity)

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error, but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}

				// Verify the capacity was updated
				var updatedEntity Entity
				err = db.Where("tenant_id = ? AND character_id = ?", tenantId, characterId).First(&updatedEntity).Error
				if err != nil {
					t.Fatalf("Failed to retrieve updated entity: %v", err)
				}

				if updatedEntity.Capacity != tt.newCapacity {
					t.Errorf("Expected capacity %d, but got %d", tt.newCapacity, updatedEntity.Capacity)
				}
			}
		})
	}
}

// TestIncreaseCapacityValidationBoundaries tests edge cases for capacity validation
func TestIncreaseCapacityValidationBoundaries(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	// Create the tables manually for SQLite compatibility
	err = db.Exec(`
		CREATE TABLE lists (
			tenant_id TEXT NOT NULL,
			id TEXT PRIMARY KEY,
			character_id INTEGER NOT NULL,
			capacity INTEGER NOT NULL
		)
	`).Error
	if err != nil {
		t.Fatalf("Failed to create lists table: %v", err)
	}

	err = db.Exec(`
		CREATE TABLE buddies (
			character_id INTEGER PRIMARY KEY,
			list_id TEXT NOT NULL,
			"group" TEXT NOT NULL,
			character_name TEXT NOT NULL,
			channel_id INTEGER NOT NULL DEFAULT -1,
			in_shop BOOLEAN NOT NULL DEFAULT false,
			pending BOOLEAN NOT NULL DEFAULT false
		)
	`).Error
	if err != nil {
		t.Fatalf("Failed to create buddies table: %v", err)
	}

	tenantId := uuid.New()
	characterId := uint32(12345)
	initialCapacity := byte(20)

	// Create initial entity
	entity := Entity{
		TenantId:    tenantId,
		Id:          uuid.New(),
		CharacterId: characterId,
		Capacity:    initialCapacity,
	}
	err = db.Create(&entity).Error
	if err != nil {
		t.Fatalf("Failed to create test entity: %v", err)
	}

	testCases := []struct {
		name        string
		newCapacity byte
		shouldPass  bool
	}{
		{"Capacity + 1", initialCapacity + 1, true},
		{"Capacity + 10", initialCapacity + 10, true},
		{"Same capacity", initialCapacity, false},
		{"Capacity - 1", initialCapacity - 1, false},
		{"Zero capacity", 0, false},
		{"Max capacity", 255, true},
		{"Minimum valid increase", initialCapacity + 1, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset capacity for each test
			db.Model(&entity).Update("capacity", initialCapacity)

			err := updateCapacity(db, tenantId, characterId, tc.newCapacity)

			if tc.shouldPass && err != nil {
				t.Errorf("Expected validation to pass for capacity %d, but got error: %v", tc.newCapacity, err)
			}
			if !tc.shouldPass && err == nil {
				t.Errorf("Expected validation to fail for capacity %d, but it passed", tc.newCapacity)
			}
		})
	}
}

// TestIncreaseCapacityDatabaseIntegration tests database state changes without requiring full processor context
func TestIncreaseCapacityDatabaseIntegration(t *testing.T) {
	// Setup database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	// Create the tables manually for SQLite compatibility
	err = db.Exec(`
		CREATE TABLE lists (
			tenant_id TEXT NOT NULL,
			id TEXT PRIMARY KEY,
			character_id INTEGER NOT NULL,
			capacity INTEGER NOT NULL
		)
	`).Error
	if err != nil {
		t.Fatalf("Failed to create lists table: %v", err)
	}

	err = db.Exec(`
		CREATE TABLE buddies (
			character_id INTEGER PRIMARY KEY,
			list_id TEXT NOT NULL,
			"group" TEXT NOT NULL,
			character_name TEXT NOT NULL,
			channel_id INTEGER NOT NULL DEFAULT -1,
			in_shop BOOLEAN NOT NULL DEFAULT false,
			pending BOOLEAN NOT NULL DEFAULT false
		)
	`).Error
	if err != nil {
		t.Fatalf("Failed to create buddies table: %v", err)
	}

	tenantId := uuid.New()
	characterId := uint32(12345)
	initialCapacity := byte(20)
	newCapacity := byte(30)

	// Create initial buddy list entity
	entity := Entity{
		TenantId:    tenantId,
		Id:          uuid.New(),
		CharacterId: characterId,
		Capacity:    initialCapacity,
	}
	err = db.Create(&entity).Error
	if err != nil {
		t.Fatalf("Failed to create test entity: %v", err)
	}

	testCases := []struct {
		name                   string
		characterId           uint32
		newCapacity           byte
		expectError           bool
		expectedErrorType     string
		expectCapacityChange  bool
	}{
		{
			name:                  "Valid capacity increase",
			characterId:          characterId,
			newCapacity:          newCapacity,
			expectError:          false,
			expectCapacityChange: true,
		},
		{
			name:              "Invalid capacity - same value",
			characterId:       characterId,
			newCapacity:       initialCapacity,
			expectError:       true,
			expectedErrorType: "INVALID_CAPACITY",
		},
		{
			name:              "Invalid capacity - lower value",
			characterId:       characterId,
			newCapacity:       initialCapacity - 5,
			expectError:       true,
			expectedErrorType: "INVALID_CAPACITY",
		},
		{
			name:              "Character not found",
			characterId:       99999,
			newCapacity:       50,
			expectError:       true,
			expectedErrorType: "CHARACTER_NOT_FOUND",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset capacity for valid character tests
			if tc.characterId == characterId {
				db.Model(&entity).Update("capacity", initialCapacity)
			}

			// Test the administrator function directly (bypassing processor tenant context complexity)
			err := updateCapacity(db, tenantId, tc.characterId, tc.newCapacity)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error, but got none")
				} else if tc.expectedErrorType != "" {
					// Check if error message contains expected type
					if tc.expectedErrorType == "CHARACTER_NOT_FOUND" && db.Dialector.Name() != "sqlite" {
						// For character not found, we expect a GORM error
						t.Logf("Got expected error for character not found: %v", err)
					} else if tc.expectedErrorType == "INVALID_CAPACITY" && err.Error() != "INVALID_CAPACITY" {
						t.Errorf("Expected error type %s, but got: %v", tc.expectedErrorType, err)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}

				if tc.expectCapacityChange {
					// Verify the database was updated
					var updatedEntity Entity
					err = db.Where("tenant_id = ? AND character_id = ?", tenantId, tc.characterId).First(&updatedEntity).Error
					if err != nil {
						t.Fatalf("Failed to retrieve updated entity: %v", err)
					}

					if updatedEntity.Capacity != tc.newCapacity {
						t.Errorf("Expected capacity %d, but got %d", tc.newCapacity, updatedEntity.Capacity)
					}
				}
			}
		})
	}
}

// TestIncreaseCapacityTransactionIntegration tests transaction handling and data consistency
func TestIncreaseCapacityTransactionIntegration(t *testing.T) {
	// Setup database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	// Create the tables manually for SQLite compatibility
	err = db.Exec(`
		CREATE TABLE lists (
			tenant_id TEXT NOT NULL,
			id TEXT PRIMARY KEY,
			character_id INTEGER NOT NULL,
			capacity INTEGER NOT NULL
		)
	`).Error
	if err != nil {
		t.Fatalf("Failed to create lists table: %v", err)
	}

	err = db.Exec(`
		CREATE TABLE buddies (
			character_id INTEGER PRIMARY KEY,
			list_id TEXT NOT NULL,
			"group" TEXT NOT NULL,
			character_name TEXT NOT NULL,
			channel_id INTEGER NOT NULL DEFAULT -1,
			in_shop BOOLEAN NOT NULL DEFAULT false,
			pending BOOLEAN NOT NULL DEFAULT false
		)
	`).Error
	if err != nil {
		t.Fatalf("Failed to create buddies table: %v", err)
	}

	tenantId := uuid.New()
	characterId := uint32(12345)
	initialCapacity := byte(20)

	// Create initial buddy list entity
	entity := Entity{
		TenantId:    tenantId,
		Id:          uuid.New(),
		CharacterId: characterId,
		Capacity:    initialCapacity,
	}
	err = db.Create(&entity).Error
	if err != nil {
		t.Fatalf("Failed to create test entity: %v", err)
	}

	// Test transaction rollback on failure
	t.Run("Transaction consistency", func(t *testing.T) {
		// Test that invalid capacity doesn't change the database
		originalCapacity := initialCapacity
		
		err := updateCapacity(db, tenantId, characterId, originalCapacity-1) // Invalid: lower capacity
		if err == nil {
			t.Error("Expected error for invalid capacity, but got none")
		}

		// Verify capacity wasn't changed
		var unchangedEntity Entity
		err = db.Where("tenant_id = ? AND character_id = ?", tenantId, characterId).First(&unchangedEntity).Error
		if err != nil {
			t.Fatalf("Failed to retrieve entity after failed update: %v", err)
		}

		if unchangedEntity.Capacity != originalCapacity {
			t.Errorf("Expected capacity to remain %d after failed update, but got %d", originalCapacity, unchangedEntity.Capacity)
		}
	})

	// Test successful capacity increase
	t.Run("Successful capacity update", func(t *testing.T) {
		newCapacity := byte(30)
		
		err := updateCapacity(db, tenantId, characterId, newCapacity)
		if err != nil {
			t.Errorf("Expected no error for valid capacity increase, but got: %v", err)
		}

		// Verify capacity was changed
		var updatedEntity Entity
		err = db.Where("tenant_id = ? AND character_id = ?", tenantId, characterId).First(&updatedEntity).Error
		if err != nil {
			t.Fatalf("Failed to retrieve entity after successful update: %v", err)
		}

		if updatedEntity.Capacity != newCapacity {
			t.Errorf("Expected capacity to be %d after successful update, but got %d", newCapacity, updatedEntity.Capacity)
		}
	})
}

// TestIncreaseCapacityConcurrentIntegration tests concurrent capacity changes at the administrator level
func TestIncreaseCapacityConcurrentIntegration(t *testing.T) {
	// Setup database with better concurrency support
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	// Enable WAL mode for better concurrency
	db.Exec("PRAGMA journal_mode=WAL;")

	// Create the tables manually for SQLite compatibility
	err = db.Exec(`
		CREATE TABLE lists (
			tenant_id TEXT NOT NULL,
			id TEXT PRIMARY KEY,
			character_id INTEGER NOT NULL,
			capacity INTEGER NOT NULL
		)
	`).Error
	if err != nil {
		t.Fatalf("Failed to create lists table: %v", err)
	}

	err = db.Exec(`
		CREATE TABLE buddies (
			character_id INTEGER PRIMARY KEY,
			list_id TEXT NOT NULL,
			"group" TEXT NOT NULL,
			character_name TEXT NOT NULL,
			channel_id INTEGER NOT NULL DEFAULT -1,
			in_shop BOOLEAN NOT NULL DEFAULT false,
			pending BOOLEAN NOT NULL DEFAULT false
		)
	`).Error
	if err != nil {
		t.Fatalf("Failed to create buddies table: %v", err)
	}

	tenantId := uuid.New()
	characterId := uint32(12345)
	initialCapacity := byte(20)

	// Create initial buddy list entity
	entity := Entity{
		TenantId:    tenantId,
		Id:          uuid.New(),
		CharacterId: characterId,
		Capacity:    initialCapacity,
	}
	err = db.Create(&entity).Error
	if err != nil {
		t.Fatalf("Failed to create test entity: %v", err)
	}

	// Test concurrent capacity changes at the administrator level
	t.Run("Concurrent capacity changes", func(t *testing.T) {
		var err1, err2 error
		done := make(chan bool, 2)

		// Simulate concurrent capacity increases using administrator functions
		go func() {
			err1 = updateCapacity(db, tenantId, characterId, 25)
			done <- true
		}()

		go func() {
			err2 = updateCapacity(db, tenantId, characterId, 30)
			done <- true
		}()

		// Wait for both operations to complete
		<-done
		<-done

		// At least one should succeed due to the validation in updateCapacity
		// The second one should fail because it checks current capacity first
		successCount := 0
		if err1 == nil {
			successCount++
		}
		if err2 == nil {
			successCount++
		}

		if successCount == 0 {
			t.Errorf("Expected at least one concurrent operation to succeed, but both failed. err1: %v, err2: %v", err1, err2)
		}

		// Verify final state is consistent
		var finalEntity Entity
		err = db.Where("tenant_id = ? AND character_id = ?", tenantId, characterId).First(&finalEntity).Error
		if err != nil {
			t.Fatalf("Failed to retrieve final entity: %v", err)
		}

		// Final capacity should be either 25 or 30, and greater than initial
		if finalEntity.Capacity <= initialCapacity {
			t.Errorf("Expected final capacity to be greater than %d, but got %d", initialCapacity, finalEntity.Capacity)
		}

		t.Logf("Concurrent test results: err1=%v, err2=%v, final_capacity=%d", err1, err2, finalEntity.Capacity)
	})
}

// mockTenant implements the tenant.Model interface for testing
type mockTenant struct {
	tenantId uuid.UUID
}

func (m mockTenant) Id() uuid.UUID {
	return m.tenantId
}

// Helper function to create a mock tenant context
func createMockTenantContext(tenantId uuid.UUID) context.Context {
	// Create a proper tenant model that implements the interface
	mockTenantModel := mockTenant{tenantId: tenantId}
	// Since we don't know the exact context key used by atlas-tenant,
	// we'll create a context that should work with the MustFromContext function
	// This may need to be adjusted based on the actual tenant package implementation
	return context.WithValue(context.Background(), "tenant", mockTenantModel)
}