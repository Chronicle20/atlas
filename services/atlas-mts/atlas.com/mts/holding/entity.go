package holding

import (
	"atlas-mts/serial"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Migration creates the holdings table. It is a brand-new table (no legacy
// primary-key rewrite), so AutoMigrate alone produces the correct surrogate-key
// shape and the composite indexes declared on the entity tags. It also migrates
// the shared per-(tenant, world) ITC-serial counter table, since CreateHolding
// draws a serial from it on every insert — co-migrating keeps the dependency
// satisfied for every caller (prod boot and the per-package test harness alike).
func Migration(db *gorm.DB) error {
	if err := serial.Migration(db); err != nil {
		return err
	}
	return db.AutoMigrate(&entity{})
}

// entity is the GORM row for a take-home holding.
//
// The primary key is a surrogate UUID (Id); business identity is never the key,
// and a (tenant_id, id) unique index keeps the row tenant-scoped. The item
// snapshot is stored as explicit name-keyed columns — one column per stat, no
// JSON blob — so a binary COPY/restore is column-order safe.
//
// DeletedAt is a GORM soft-delete column: take-home soft-deletes by id so a
// repeated take-home is idempotent (the second delete affects zero rows).
//
// Composite indexes back the design's hot queries:
//   - (tenant_id, world_id, owner_id) — list a character's holdings in a world
//   - (tenant_id, world_id, serial) UNIQUE — serial->row resolution for the
//     take-home ITC_OPERATION arm; the serial is the client's nITCSN, drawn from
//     the SAME per-(tenant, world) counter as listings, so a serial maps to
//     exactly one holding OR listing within a world.
type entity struct {
	Id       uuid.UUID `gorm:"column:id;type:uuid;primaryKey;uniqueIndex:idx_holdings_tenant_id,priority:2"`
	TenantId uuid.UUID `gorm:"column:tenant_id;type:uuid;not null;uniqueIndex:idx_holdings_tenant_id,priority:1;index:idx_holdings_world_owner,priority:1;uniqueIndex:idx_holdings_world_serial,priority:1"`
	WorldId  byte      `gorm:"column:world_id;not null;index:idx_holdings_world_owner,priority:2;uniqueIndex:idx_holdings_world_serial,priority:2"`
	Serial   uint32    `gorm:"column:serial;not null;uniqueIndex:idx_holdings_world_serial,priority:3"`
	OwnerId  uint32    `gorm:"column:owner_id;not null;index:idx_holdings_world_owner,priority:3"`

	Origin string `gorm:"column:origin;not null"`

	TemplateId uint32 `gorm:"column:template_id;not null"`
	Quantity   uint32 `gorm:"column:quantity;not null"`

	Strength      uint16 `gorm:"column:strength;not null"`
	Dexterity     uint16 `gorm:"column:dexterity;not null"`
	Intelligence  uint16 `gorm:"column:intelligence;not null"`
	Luck          uint16 `gorm:"column:luck;not null"`
	HP            uint16 `gorm:"column:hp;not null"`
	MP            uint16 `gorm:"column:mp;not null"`
	WeaponAttack  uint16 `gorm:"column:weapon_attack;not null"`
	MagicAttack   uint16 `gorm:"column:magic_attack;not null"`
	WeaponDefense uint16 `gorm:"column:weapon_defense;not null"`
	MagicDefense  uint16 `gorm:"column:magic_defense;not null"`
	Accuracy      uint16 `gorm:"column:accuracy;not null"`
	Avoidability  uint16 `gorm:"column:avoidability;not null"`
	Hands         uint16 `gorm:"column:hands;not null"`
	Speed         uint16 `gorm:"column:speed;not null"`
	Jump          uint16 `gorm:"column:jump;not null"`
	Slots         uint16 `gorm:"column:slots;not null"`
	Level         byte   `gorm:"column:level;not null"`
	ItemLevel     byte   `gorm:"column:item_level;not null"`
	ItemExp       uint32 `gorm:"column:item_exp;not null"`
	RingId        uint32 `gorm:"column:ring_id;not null"`
	ViciousCount  uint32 `gorm:"column:vicious_count;not null"`
	Flags         uint16 `gorm:"column:flags;not null"`
	Owner         string `gorm:"column:owner;not null;default:''"`

	CreatedAt time.Time      `gorm:"column:created_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (e entity) TableName() string {
	return "holdings"
}
