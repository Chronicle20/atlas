package escrow

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// legacyItemEntity is the pre-snapshot row shape, kept only in this test so the
// migration can be driven against a table that actually has the stale columns.
type legacyItemEntity struct {
	Id           string `gorm:"column:id;primaryKey"`
	RingId       uint32 `gorm:"column:ring_id;not null"`
	ItemLevel    byte   `gorm:"column:item_level;not null"`
	ItemExp      uint32 `gorm:"column:item_exp;not null"`
	ViciousCount uint32 `gorm:"column:vicious_count;not null"`
}

func (legacyItemEntity) TableName() string { return itemTable }

// TestMigrationDropsStaleColumns pins that a database already migrated to the
// pre-snapshot shape loses the columns nothing writes any more. AutoMigrate adds
// but never drops, and each stale column was created NOT NULL with no default —
// so leaving one behind makes every subsequent INSERT fail.
func TestMigrationDropsStaleColumns(t *testing.T) {
	db := testDb(t)
	if err := db.AutoMigrate(&legacyItemEntity{}); err != nil {
		t.Fatalf("legacy AutoMigrate: %v", err)
	}
	for _, c := range staleItemColumns {
		if !db.Migrator().HasColumn(&ItemEntity{}, c) {
			t.Fatalf("precondition: legacy table is missing column %q", c)
		}
	}
	if err := Migration(db); err != nil {
		t.Fatalf("Migration: %v", err)
	}
	for _, c := range staleItemColumns {
		if db.Migrator().HasColumn(&ItemEntity{}, c) {
			t.Errorf("stale column %q survived the migration", c)
		}
	}
}

// legacyMesoEntity is the pre-stake-table row shape: one in-flight stake held in
// three columns on the meso row itself. Kept only in this test so the migration
// can be driven against a table that actually has that slot.
type legacyMesoEntity struct {
	Id             uuid.UUID `gorm:"column:id;type:uuid;primaryKey"`
	TenantId       uuid.UUID `gorm:"column:tenant_id;type:uuid;not null"`
	TenantRegion   string    `gorm:"column:tenant_region;type:varchar(32);not null;default:''"`
	TenantMajor    uint16    `gorm:"column:tenant_major;not null;default:0"`
	TenantMinor    uint16    `gorm:"column:tenant_minor;not null;default:0"`
	RoomId         uuid.UUID `gorm:"column:room_id;type:uuid;not null"`
	OwnerId        uint32    `gorm:"column:owner_id;not null"`
	Amount         int64     `gorm:"column:amount;not null"`
	PendingStakeId uuid.UUID `gorm:"column:pending_stake_id;type:uuid"`
	PendingAmount  uint32    `gorm:"column:pending_amount;not null;default:0"`
	PendingDelta   int32     `gorm:"column:pending_delta;not null;default:0"`
	CreatedAt      time.Time `gorm:"column:created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at"`
}

func (legacyMesoEntity) TableName() string { return mesoTable }

// TestMigrationLiftsAnArmedStakeOutOfTheOldSlot pins that the three pending_*
// columns are not simply dropped.
//
// A row carrying a stake at migration time describes meso that has ALREADY left
// a player's pocket and has no other record anywhere. Dropping the slot without
// lifting it strands that debit: its terminal status finds no stake to resolve,
// and the boot sweep — which walks stakes — never sees it either.
func TestMigrationLiftsAnArmedStakeOutOfTheOldSlot(t *testing.T) {
	db := testDb(t)
	te := testTenant(t)

	if err := db.AutoMigrate(&legacyMesoEntity{}); err != nil {
		t.Fatalf("legacy AutoMigrate: %v", err)
	}
	roomId := uuid.New()
	stakeId := uuid.New()
	armed := legacyMesoEntity{
		Id: uuid.New(), TenantId: te.Id(), TenantRegion: te.Region(),
		TenantMajor: te.MajorVersion(), TenantMinor: te.MinorVersion(),
		RoomId: roomId, OwnerId: 100, Amount: 5_000,
		PendingStakeId: stakeId, PendingAmount: 900, PendingDelta: -4_100,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := db.Create(&armed).Error; err != nil {
		t.Fatalf("seed a legacy row with an armed stake: %v", err)
	}
	// A second row with an EMPTY slot, to prove the backfill lifts only genuine
	// stakes. The old shape used uuid.Nil, not NULL, as its "none" sentinel.
	idle := legacyMesoEntity{
		Id: uuid.New(), TenantId: te.Id(), TenantRegion: te.Region(),
		TenantMajor: te.MajorVersion(), TenantMinor: te.MinorVersion(),
		RoomId: roomId, OwnerId: 200, Amount: 1_000,
		PendingStakeId: uuid.Nil,
		CreatedAt:      time.Now(), UpdatedAt: time.Now(),
	}
	if err := db.Create(&idle).Error; err != nil {
		t.Fatalf("seed a legacy row with no stake: %v", err)
	}

	if err := Migration(db); err != nil {
		t.Fatalf("Migration: %v", err)
	}

	for _, c := range staleMesoColumns {
		if db.Migrator().HasColumn(&MesoEntity{}, c) {
			t.Errorf("stale column %q survived the migration", c)
		}
	}

	lifted, found, err := MesoStakeById(db)(stakeId)
	if err != nil {
		t.Fatalf("MesoStakeById: %v", err)
	}
	if !found {
		t.Fatal("the armed stake was dropped with the columns; the player's debit is stranded with nothing to resolve against")
	}
	if lifted.OwnerId() != 100 || lifted.RoomId() != roomId {
		t.Errorf("lifted stake owner/room: got %d/%s, want 100/%s", lifted.OwnerId(), lifted.RoomId(), roomId)
	}
	if lifted.Amount() != 900 || lifted.Delta() != -4_100 {
		t.Errorf("lifted stake: got amount %d delta %d, want 900/-4100", lifted.Amount(), lifted.Delta())
	}

	// The idle row contributes nothing, and the committed totals are untouched.
	all, err := AllMesoStakes(db)
	if err != nil {
		t.Fatalf("AllMesoStakes: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("stakes after backfill: got %d, want only the one genuinely armed", len(all))
	}
	if got, _, err := MesoByOwner(db, te.Id())(roomId, 100); err != nil || got != 5_000 {
		t.Errorf("committed total for the armed row: got %d (err %v), want 5000 untouched", got, err)
	}

	// Migration runs on every boot; a second pass must not duplicate the lift.
	if err := Migration(db); err != nil {
		t.Fatalf("second Migration: %v", err)
	}
	if all, err := AllMesoStakes(db); err != nil || len(all) != 1 {
		t.Errorf("stakes after a second migration: got %d (err %v), want still 1", len(all), err)
	}
}
