// +build cgo

package family

import (
	"atlas-family/database"
	"atlas-family/logger"
	"os"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func setupTestDB() (*gorm.DB, func()) {
	// Create in-memory SQLite database for testing
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		panic("failed to connect to test database")
	}

	// Run migrations
	if err := Migration(db); err != nil {
		panic("failed to run migrations: " + err.Error())
	}

	// Return cleanup function
	cleanup := func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}

	return db, cleanup
}

func TestFamilyIntegration_CreateAndRetrieve(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()

	// Create a family member
	characterId := uint32(12345)
	tenantId := uuid.New()
	level := uint16(50)
	world := byte(1)

	member, err := NewBuilder(characterId, tenantId, level, world).
		SetRep(1000).
		SetDailyRep(100).
		Build()

	if err != nil {
		t.Fatalf("Failed to build family member: %v", err)
	}

	// Save to database
	entity := ToEntity(member)
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to save family member: %v", err)
	}

	// Retrieve from database
	var retrievedEntity Entity
	if err := db.Where("character_id = ?", characterId).First(&retrievedEntity).Error; err != nil {
		t.Fatalf("Failed to retrieve family member: %v", err)
	}

	// Convert back to model
	retrievedMember, err := Make(retrievedEntity)
	if err != nil {
		t.Fatalf("Failed to convert entity to model: %v", err)
	}

	// Verify fields match
	if retrievedMember.CharacterId() != characterId {
		t.Errorf("Expected CharacterId %d, got %d", characterId, retrievedMember.CharacterId())
	}

	if retrievedMember.TenantId() != tenantId {
		t.Errorf("Expected TenantId %v, got %v", tenantId, retrievedMember.TenantId())
	}

	if retrievedMember.Level() != level {
		t.Errorf("Expected Level %d, got %d", level, retrievedMember.Level())
	}

	if retrievedMember.World() != world {
		t.Errorf("Expected World %d, got %d", world, retrievedMember.World())
	}

	if retrievedMember.Rep() != 1000 {
		t.Errorf("Expected Rep %d, got %d", 1000, retrievedMember.Rep())
	}

	if retrievedMember.DailyRep() != 100 {
		t.Errorf("Expected DailyRep %d, got %d", 100, retrievedMember.DailyRep())
	}
}

func TestFamilyIntegration_FamilyRelationships(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()

	// Create senior member
	seniorId := uint32(12345)
	seniorTenantId := uuid.New()
	senior, err := NewBuilder(seniorId, seniorTenantId, uint16(50), 1).Build()
	if err != nil {
		t.Fatalf("Failed to build senior member: %v", err)
	}

	// Create junior member
	juniorId := uint32(67890)
	juniorTenantId := uuid.New()
	junior, err := NewBuilder(juniorId, juniorTenantId, uint16(45), 1).
		SetSeniorId(seniorId).
		Build()
	if err != nil {
		t.Fatalf("Failed to build junior member: %v", err)
	}

	// Update senior with junior
	seniorWithJunior, err := senior.Builder().
		AddJunior(juniorId).
		Build()
	if err != nil {
		t.Fatalf("Failed to add junior to senior: %v", err)
	}

	// Save both to database
	seniorEntity := ToEntity(seniorWithJunior)
	if err := db.Create(&seniorEntity).Error; err != nil {
		t.Fatalf("Failed to save senior member: %v", err)
	}

	juniorEntity := ToEntity(junior)
	if err := db.Create(&juniorEntity).Error; err != nil {
		t.Fatalf("Failed to save junior member: %v", err)
	}

	// Retrieve and verify relationships
	var retrievedSenior Entity
	if err := db.Where("character_id = ?", seniorId).First(&retrievedSenior).Error; err != nil {
		t.Fatalf("Failed to retrieve senior member: %v", err)
	}

	var retrievedJunior Entity
	if err := db.Where("character_id = ?", juniorId).First(&retrievedJunior).Error; err != nil {
		t.Fatalf("Failed to retrieve junior member: %v", err)
	}

	// Convert to models
	seniorModel, err := Make(retrievedSenior)
	if err != nil {
		t.Fatalf("Failed to convert senior entity to model: %v", err)
	}

	juniorModel, err := Make(retrievedJunior)
	if err != nil {
		t.Fatalf("Failed to convert junior entity to model: %v", err)
	}

	// Verify relationships
	if !seniorModel.HasJuniors() {
		t.Error("Senior should have juniors")
	}

	if len(seniorModel.JuniorIds()) != 1 {
		t.Errorf("Expected 1 junior, got %d", len(seniorModel.JuniorIds()))
	}

	if seniorModel.JuniorIds()[0] != juniorId {
		t.Errorf("Expected junior ID %d, got %d", juniorId, seniorModel.JuniorIds()[0])
	}

	if !juniorModel.HasSenior() {
		t.Error("Junior should have a senior")
	}

	if juniorModel.SeniorId() == nil || *juniorModel.SeniorId() != seniorId {
		actualSeniorId := uint32(0)
		if juniorModel.SeniorId() != nil {
			actualSeniorId = *juniorModel.SeniorId()
		}
		t.Errorf("Expected senior ID %d, got %d", seniorId, actualSeniorId)
	}
}

func TestFamilyIntegration_ReputationOperations(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()

	// Create member with some reputation
	characterId := uint32(12345)
	tenantId := uuid.New()
	
	member, err := NewBuilder(characterId, tenantId, uint16(50), 1).
		SetRep(1000).
		SetDailyRep(100).
		Build()
	if err != nil {
		t.Fatalf("Failed to build family member: %v", err)
	}

	// Save to database
	entity := ToEntity(member)
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to save family member: %v", err)
	}

	// Test reputation addition
	t.Run("Add reputation", func(t *testing.T) {
		updatedMember, err := member.Builder().
			AddRep(500).
			AddDailyRep(50).
			Build()
		if err != nil {
			t.Fatalf("Failed to add reputation: %v", err)
		}

		if updatedMember.Rep() != 1500 {
			t.Errorf("Expected Rep %d, got %d", 1500, updatedMember.Rep())
		}

		if updatedMember.DailyRep() != 150 {
			t.Errorf("Expected DailyRep %d, got %d", 150, updatedMember.DailyRep())
		}
	})

	// Test reputation deduction
	t.Run("Subtract reputation", func(t *testing.T) {
		updatedMember, err := member.Builder().
			SubtractRep(300).
			Build()
		if err != nil {
			t.Fatalf("Failed to subtract reputation: %v", err)
		}

		if updatedMember.Rep() != 700 {
			t.Errorf("Expected Rep %d, got %d", 700, updatedMember.Rep())
		}
	})

	// Test daily rep reset
	t.Run("Reset daily reputation", func(t *testing.T) {
		updatedMember, err := member.Builder().
			ResetDailyRep().
			Build()
		if err != nil {
			t.Fatalf("Failed to reset daily reputation: %v", err)
		}

		if updatedMember.DailyRep() != 0 {
			t.Errorf("Expected DailyRep %d, got %d", 0, updatedMember.DailyRep())
		}
	})
}

func TestFamilyIntegration_ValidationRules(t *testing.T) {
	characterId := uint32(12345)
	tenantId := uuid.New()

	// Test level difference validation
	t.Run("Level difference validation", func(t *testing.T) {
		tests := []struct {
			name           string
			seniorLevel    uint16
			juniorLevel    uint16
			expectedResult bool
		}{
			{"Valid difference", 50, 45, true},
			{"Same level", 50, 50, true},
			{"Max difference", 50, 30, true},
			{"Exceeds max difference", 50, 29, false},
			{"Junior higher level", 45, 50, true},
			{"Large difference", 100, 50, false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := ValidateLevelDifference(tt.seniorLevel, tt.juniorLevel)
				if result != tt.expectedResult {
					t.Errorf("ValidateLevelDifference(%d, %d) = %v, want %v",
						tt.seniorLevel, tt.juniorLevel, result, tt.expectedResult)
				}
			})
		}
	})

	// Test location validation
	t.Run("Location validation", func(t *testing.T) {
		tests := []struct {
			name           string
			seniorWorld    byte
			seniorMap      uint32
			juniorWorld    byte
			juniorMap      uint32
			expectedResult bool
		}{
			{"Same location", 1, 100, 1, 100, true},
			{"Different world", 1, 100, 2, 100, false},
			{"Different map", 1, 100, 1, 200, false},
			{"Both different", 1, 100, 2, 200, false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := ValidateLocation(tt.seniorWorld, tt.seniorMap, tt.juniorWorld, tt.juniorMap)
				if result != tt.expectedResult {
					t.Errorf("ValidateLocation(%d, %d, %d, %d) = %v, want %v",
						tt.seniorWorld, tt.seniorMap, tt.juniorWorld, tt.juniorMap, result, tt.expectedResult)
				}
			})
		}
	})

	// Test daily rep cap validation
	t.Run("Daily rep cap validation", func(t *testing.T) {
		tests := []struct {
			name           string
			currentDailyRep uint32
			additionalRep  uint32
			expectedResult bool
		}{
			{"Under cap", 1000, 500, true},
			{"At cap", 5000, 0, true},
			{"Would exceed cap", 4500, 600, false},
			{"Already over cap", 5500, 100, false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := ValidateDailyRepCap(tt.currentDailyRep, tt.additionalRep)
				if result != tt.expectedResult {
					t.Errorf("ValidateDailyRepCap(%d, %d) = %v, want %v",
						tt.currentDailyRep, tt.additionalRep, result, tt.expectedResult)
				}
			})
		}
	})

	// Test builder validation
	t.Run("Builder validation", func(t *testing.T) {
		tests := []struct {
			name        string
			setupBuilder func() *Builder
			expectError bool
		}{
			{
				name: "Valid builder",
				setupBuilder: func() *Builder {
					return NewBuilder(characterId, tenantId, uint16(50), 1)
				},
				expectError: false,
			},
			{
				name: "Too many juniors",
				setupBuilder: func() *Builder {
					return NewBuilder(characterId, tenantId, uint16(50), 1).
						AddJunior(11111).
						AddJunior(22222).
						AddJunior(33333)
				},
				expectError: true,
			},
			{
				name: "Daily rep over cap",
				setupBuilder: func() *Builder {
					return NewBuilder(characterId, tenantId, uint16(50), 1).
						SetDailyRep(6000)
				},
				expectError: true,
			},
			{
				name: "Self reference as senior",
				setupBuilder: func() *Builder {
					return NewBuilder(characterId, tenantId, uint16(50), 1).
						SetSeniorId(characterId)
				},
				expectError: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				builder := tt.setupBuilder()
				_, err := builder.Build()

				if tt.expectError && err == nil {
					t.Error("Expected error but got none")
				}

				if !tt.expectError && err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			})
		}
	})
}

func TestFamilyIntegration_ProviderOperations(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()

	// Create test members
	seniorId := uint32(12345)
	juniorId := uint32(67890)
	tenantId := uuid.New()

	// Create and save senior
	senior, err := NewBuilder(seniorId, tenantId, uint16(50), 1).
		SetRep(1000).
		Build()
	if err != nil {
		t.Fatalf("Failed to build senior: %v", err)
	}

	seniorEntity := ToEntity(senior)
	if err := db.Create(&seniorEntity).Error; err != nil {
		t.Fatalf("Failed to save senior: %v", err)
	}

	// Create and save junior
	junior, err := NewBuilder(juniorId, tenantId, uint16(45), 1).
		SetSeniorId(seniorId).
		SetRep(500).
		Build()
	if err != nil {
		t.Fatalf("Failed to build junior: %v", err)
	}

	juniorEntity := ToEntity(junior)
	if err := db.Create(&juniorEntity).Error; err != nil {
		t.Fatalf("Failed to save junior: %v", err)
	}

	// Test GetByCharacterIdProvider
	t.Run("GetByCharacterIdProvider", func(t *testing.T) {
		result, err := GetByCharacterIdProvider(seniorId)(db)()
		if err != nil {
			t.Fatalf("GetByCharacterIdProvider failed: %v", err)
		}

		if result.CharacterId() != seniorId {
			t.Errorf("Expected CharacterId %d, got %d", seniorId, result.CharacterId())
		}
	})

	// Test GetByTenantIdProvider
	t.Run("GetByTenantIdProvider", func(t *testing.T) {
		result, err := GetByTenantIdProvider(tenantId)(db)()
		if err != nil {
			t.Fatalf("GetByTenantIdProvider failed: %v", err)
		}

		if len(result) != 2 {
			t.Errorf("Expected 2 members, got %d", len(result))
		}
	})

	// Test GetBySeniorIdProvider
	t.Run("GetBySeniorIdProvider", func(t *testing.T) {
		result, err := GetBySeniorIdProvider(seniorId)(db)()
		if err != nil {
			t.Fatalf("GetBySeniorIdProvider failed: %v", err)
		}

		if len(result) != 1 {
			t.Errorf("Expected 1 junior, got %d", len(result))
		}

		if result[0].CharacterId() != juniorId {
			t.Errorf("Expected junior ID %d, got %d", juniorId, result[0].CharacterId())
		}
	})
}

// Integration test to verify the complete build process works
func TestFamilyIntegration_BuildProcess(t *testing.T) {
	// Test that the project builds successfully
	if os.Getenv("SKIP_BUILD_TEST") != "true" {
		t.Skip("Skipping build test - set SKIP_BUILD_TEST=true to run")
	}

	t.Run("Build family service", func(t *testing.T) {
		// This would typically be run as part of CI/CD
		// For now, we'll just verify imports work
		l := logger.CreateLogger("test")
		if l == nil {
			t.Error("Failed to create logger")
		}

		// Verify database connection works
		db := database.Connect(l)
		if db == nil {
			t.Error("Failed to connect to database")
		}
	})
}