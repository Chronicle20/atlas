package list

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		return nil, err
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
		return nil, err
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
		return nil, err
	}

	return db, nil
}

func TestUpdateCapacity(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
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

	tests := []struct {
		name          string
		characterId   uint32
		newCapacity   byte
		expectedError string
	}{
		{
			name:        "Valid capacity increase",
			characterId: characterId,
			newCapacity: 30,
		},
		{
			name:          "Invalid capacity - same value",
			characterId:   characterId,
			newCapacity:   initialCapacity,
			expectedError: "INVALID_CAPACITY",
		},
		{
			name:          "Invalid capacity - lower value",
			characterId:   characterId,
			newCapacity:   15,
			expectedError: "INVALID_CAPACITY",
		},
		{
			name:          "Character not found",
			characterId:   99999,
			newCapacity:   30,
			expectedError: "record not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := updateCapacity(db, tenantId, tt.characterId, tt.newCapacity)

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error containing '%s', but got no error", tt.expectedError)
					return
				}
				if err.Error() != tt.expectedError && !errors.Is(err, gorm.ErrRecordNotFound) {
					t.Errorf("Expected error '%s', but got '%s'", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}

				// Verify the capacity was updated
				var updatedEntity Entity
				err = db.Where("tenant_id = ? AND character_id = ?", tenantId, tt.characterId).First(&updatedEntity).Error
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