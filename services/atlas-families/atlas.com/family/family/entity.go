package family

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Entity represents the GORM-compatible database representation of a family member
type Entity struct {
	ID          uint32    `gorm:"primaryKey;autoIncrement" json:"id"`
	CharacterId uint32    `gorm:"uniqueIndex;not null" json:"characterId"`
	TenantId    uuid.UUID `gorm:"type:uuid;not null;index" json:"tenantId"`
	SeniorId    *uint32   `gorm:"index" json:"seniorId"`
	JuniorIds   []uint32  `gorm:"type:integer[]" json:"juniorIds"`
	Rep         uint32    `gorm:"default:0" json:"rep"`
	DailyRep    uint32    `gorm:"default:0" json:"dailyRep"`
	Level       uint16    `gorm:"not null" json:"level"`
	World       byte      `gorm:"not null" json:"world"`
	Channel     byte      `gorm:"not null" json:"channel"`
	MapId       uint32    `gorm:"not null" json:"mapId"`
	CreatedAt   time.Time `gorm:"not null" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"not null" json:"updatedAt"`
}

// TableName specifies the table name for the Entity
func (Entity) TableName() string {
	return "family_members"
}

// Migration creates the family_members table with proper indexes and constraints
func Migration(db *gorm.DB) error {
	err := db.AutoMigrate(&Entity{})
	if err != nil {
		return err
	}

	// Add indexes for optimized queries
	err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_family_members_tenant_character 
		ON family_members(tenant_id, character_id);
		
		CREATE INDEX IF NOT EXISTS idx_family_members_senior_id 
		ON family_members(senior_id) WHERE senior_id IS NOT NULL;
		
		CREATE INDEX IF NOT EXISTS idx_family_members_world_map 
		ON family_members(world, map_id);
		
		CREATE INDEX IF NOT EXISTS idx_family_members_updated_at 
		ON family_members(updated_at);
	`).Error
	if err != nil {
		return err
	}

	// Add constraints
	err = db.Exec(`
		ALTER TABLE family_members 
		ADD CONSTRAINT check_junior_count 
		CHECK (array_length(junior_ids, 1) IS NULL OR array_length(junior_ids, 1) <= 2);
		
		ALTER TABLE family_members 
		ADD CONSTRAINT check_rep_non_negative 
		CHECK (rep >= 0);
		
		ALTER TABLE family_members 
		ADD CONSTRAINT check_daily_rep_non_negative 
		CHECK (daily_rep >= 0);
		
		ALTER TABLE family_members 
		ADD CONSTRAINT check_daily_rep_limit 
		CHECK (daily_rep <= 5000);
		
		ALTER TABLE family_members 
		ADD CONSTRAINT check_level_positive 
		CHECK (level > 0);
		
		ALTER TABLE family_members 
		ADD CONSTRAINT check_no_self_senior 
		CHECK (senior_id != character_id);
	`).Error
	if err != nil {
		return err
	}

	return nil
}

// Make transforms an Entity into an immutable FamilyMember model
func Make(entity Entity) (FamilyMember, error) {
	// Validate required fields
	if err := ValidateCharacterId(entity.CharacterId); err != nil {
		return FamilyMember{}, err
	}

	if err := ValidateTenantId(entity.TenantId); err != nil {
		return FamilyMember{}, err
	}

	if err := ValidateLevel(entity.Level); err != nil {
		return FamilyMember{}, err
	}

	if err := ValidateJuniorIds(entity.CharacterId, entity.JuniorIds); err != nil {
		return FamilyMember{}, err
	}

	if err := ValidateSeniorId(entity.CharacterId, entity.SeniorId); err != nil {
		return FamilyMember{}, err
	}

	// Copy junior IDs to avoid shared references
	juniorIds := make([]uint32, len(entity.JuniorIds))
	copy(juniorIds, entity.JuniorIds)

	return FamilyMember{
		id:          entity.ID,
		characterId: entity.CharacterId,
		tenantId:    entity.TenantId,
		seniorId:    entity.SeniorId,
		juniorIds:   juniorIds,
		rep:         entity.Rep,
		dailyRep:    entity.DailyRep,
		level:       entity.Level,
		world:       entity.World,
		channel:     entity.Channel,
		mapId:       entity.MapId,
		createdAt:   entity.CreatedAt,
		updatedAt:   entity.UpdatedAt,
	}, nil
}

// ToEntity converts a FamilyMember model back to an Entity for database operations
func ToEntity(fm FamilyMember) Entity {
	// Copy junior IDs to avoid shared references
	juniorIds := make([]uint32, len(fm.juniorIds))
	copy(juniorIds, fm.juniorIds)

	return Entity{
		ID:          fm.id,
		CharacterId: fm.characterId,
		TenantId:    fm.tenantId,
		SeniorId:    fm.seniorId,
		JuniorIds:   juniorIds,
		Rep:         fm.rep,
		DailyRep:    fm.dailyRep,
		Level:       fm.level,
		World:       fm.world,
		Channel:     fm.channel,
		MapId:       fm.mapId,
		CreatedAt:   fm.createdAt,
		UpdatedAt:   fm.updatedAt,
	}
}