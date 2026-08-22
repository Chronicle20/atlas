package playernpc

import (
	"sort"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Entity is a deployed Player NPC, PRD §6 `player_npcs`. Equipment is
// declared as an association purely so AutoMigrate creates the
// `player_npc_equipment.player_npc_id` foreign key with ON DELETE CASCADE
// (PRD §6 migration note) -- the administrator never populates or relies on
// GORM's association save/preload for it, inserting/reading child rows
// itself instead.
//
// Step is an addition beyond design §3.1's literal column list (Task 15's
// resolution of that section's open question: the positioner step was
// "computed but not persisted"). Deriving it instead -- replaying
// NextGridPosition from step 0 on every deploy -- would mean reconstructing
// a map's entire placement history through repeated ground-snap network
// calls every time, and that reconstruction's correctness silently depends
// on the snap endpoint staying byte-for-byte deterministic across however
// many historical calls a busy map has accumulated. A persisted column
// costs one more not-null byte per row, no worse than rx0/rx1 (already
// stored rather than recomputed, by the same section's own reasoning), and
// is exact rather than replayed. All Player NPCs on one map share the same
// Step after any reorganize (design §5.4 rewrites every row in the map in
// one transaction), so reading any one row's Step (or 0 when the map has
// none yet) gives the map's current step; see administrator.go's
// currentStepForMap.
type Entity struct {
	Id             uuid.UUID         `gorm:"primaryKey;type:uuid"`
	TenantId       uuid.UUID         `gorm:"not null;type:uuid;uniqueIndex:idx_pn_tw_script,priority:1;uniqueIndex:idx_pn_tw_map_name,priority:1;uniqueIndex:idx_pn_tw_map_object,priority:1;index:idx_pn_tw_map,priority:1;index:idx_pn_tw_character,priority:1"`
	CharacterId    uint32            `gorm:"not null;index:idx_pn_tw_character,priority:3"`
	Name           string            `gorm:"not null;uniqueIndex:idx_pn_tw_map_name,priority:4"`
	WorldId        byte              `gorm:"not null;uniqueIndex:idx_pn_tw_script,priority:2;uniqueIndex:idx_pn_tw_map_name,priority:2;uniqueIndex:idx_pn_tw_map_object,priority:2;index:idx_pn_tw_map,priority:2;index:idx_pn_tw_character,priority:2"`
	MapId          uint32            `gorm:"not null;uniqueIndex:idx_pn_tw_map_name,priority:3;uniqueIndex:idx_pn_tw_map_object,priority:3;index:idx_pn_tw_map,priority:3"`
	ScriptId       uint32            `gorm:"not null;uniqueIndex:idx_pn_tw_script,priority:3"`
	ObjectId       uint32            `gorm:"not null;uniqueIndex:idx_pn_tw_map_object,priority:4"`
	Gender         byte              `gorm:"not null"`
	Skin           byte              `gorm:"not null"`
	Face           uint32            `gorm:"not null"`
	Hair           uint32            `gorm:"not null"`
	JobId          uint16            `gorm:"not null"`
	X              int16             `gorm:"not null"`
	Cy             int16             `gorm:"not null"`
	Fh             uint16            `gorm:"not null"`
	RX0            int16             `gorm:"not null"`
	RX1            int16             `gorm:"not null"`
	Dir            byte              `gorm:"not null;default:1"`
	Step           byte              `gorm:"not null;default:0"`
	WorldRank      uint32            `gorm:"not null;default:0"`
	OverallRank    uint32            `gorm:"not null;default:0"`
	WorldJobRank   uint32            `gorm:"not null;default:0"`
	OverallJobRank uint32            `gorm:"not null;default:0"`
	Equipment      []EquipmentEntity `gorm:"foreignKey:PlayerNpcId;constraint:OnDelete:CASCADE"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// TableName specifies the database table name for Entity.
func (Entity) TableName() string {
	return "player_npcs"
}

// BeforeCreate assigns a new id when the caller did not supply one.
func (e *Entity) BeforeCreate(_ *gorm.DB) error {
	if e.Id == uuid.Nil {
		e.Id = uuid.New()
	}
	return nil
}

// EquipmentEntity is one frozen equipment slot, PRD §6
// `player_npc_equipment`.
type EquipmentEntity struct {
	Id          uuid.UUID `gorm:"primaryKey;type:uuid"`
	TenantId    uuid.UUID `gorm:"not null;type:uuid;uniqueIndex:idx_pne_tenant_npc_slot,priority:1"`
	PlayerNpcId uuid.UUID `gorm:"not null;type:uuid;uniqueIndex:idx_pne_tenant_npc_slot,priority:2"`
	Slot        int16     `gorm:"not null;uniqueIndex:idx_pne_tenant_npc_slot,priority:3"`
	ItemId      uint32    `gorm:"not null"`
}

// TableName specifies the database table name for EquipmentEntity.
func (EquipmentEntity) TableName() string {
	return "player_npc_equipment"
}

// BeforeCreate assigns a new id when the caller did not supply one.
func (e *EquipmentEntity) BeforeCreate(_ *gorm.DB) error {
	if e.Id == uuid.Nil {
		e.Id = uuid.New()
	}
	return nil
}

// Make converts an Entity plus its equipment rows into a Model. Equipment
// is sorted by slot ascending -- callers must not rely on the order rows
// were fetched in. rx0/rx1 are restored from the stored columns rather than
// recomputed (design §3.1).
func Make(e Entity, equipmentEntities []EquipmentEntity) (Model, error) {
	sorted := make([]EquipmentEntity, len(equipmentEntities))
	copy(sorted, equipmentEntities)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Slot < sorted[j].Slot })

	equipment := make([]EquipmentModel, 0, len(sorted))
	for _, ee := range sorted {
		em, err := MakeEquipment(ee)
		if err != nil {
			return Model{}, err
		}
		equipment = append(equipment, em)
	}

	return NewBuilder().
		SetId(e.Id).
		SetCharacterId(e.CharacterId).
		SetName(e.Name).
		SetWorldId(e.WorldId).
		SetMapId(e.MapId).
		SetScriptId(e.ScriptId).
		SetObjectId(e.ObjectId).
		SetGender(e.Gender).
		SetSkin(e.Skin).
		SetFace(e.Face).
		SetHair(e.Hair).
		setJobIdCategory(e.JobId).
		SetX(e.X).
		SetCy(e.Cy).
		SetFh(e.Fh).
		SetRX0(e.RX0).
		SetRX1(e.RX1).
		SetDir(e.Dir).
		SetStep(e.Step).
		SetWorldRank(e.WorldRank).
		SetOverallRank(e.OverallRank).
		SetWorldJobRank(e.WorldJobRank).
		SetOverallJobRank(e.OverallJobRank).
		SetCreatedAt(e.CreatedAt).
		SetUpdatedAt(e.UpdatedAt).
		SetEquipment(equipment).
		Build()
}

// MakeEquipment converts an EquipmentEntity into an EquipmentModel.
func MakeEquipment(e EquipmentEntity) (EquipmentModel, error) {
	return NewEquipmentBuilder().
		SetId(e.Id).
		SetSlot(e.Slot).
		SetItemId(e.ItemId).
		Build()
}

// ToEntity converts a Model's root fields into an Entity. Equipment is
// converted separately by MakeEquipmentEntities, since the parent id is
// only known once the root row has been created.
func (m Model) ToEntity(tenantId uuid.UUID) Entity {
	return Entity{
		Id:             m.Id(),
		TenantId:       tenantId,
		CharacterId:    m.CharacterId(),
		Name:           m.Name(),
		WorldId:        m.WorldId(),
		MapId:          m.MapId(),
		ScriptId:       m.ScriptId(),
		ObjectId:       m.ObjectId(),
		Gender:         m.Gender(),
		Skin:           m.Skin(),
		Face:           m.Face(),
		Hair:           m.Hair(),
		JobId:          m.JobId(),
		X:              m.X(),
		Cy:             m.Cy(),
		Fh:             m.Fh(),
		RX0:            m.RX0(),
		RX1:            m.RX1(),
		Dir:            m.Dir(),
		Step:           m.Step(),
		WorldRank:      m.WorldRank(),
		OverallRank:    m.OverallRank(),
		WorldJobRank:   m.WorldJobRank(),
		OverallJobRank: m.OverallJobRank(),
		CreatedAt:      m.CreatedAt(),
		UpdatedAt:      m.UpdatedAt(),
	}
}

// MakeEquipmentEntities converts a Model's equipment collection into
// EquipmentEntity rows for playerNpcId.
func MakeEquipmentEntities(tenantId uuid.UUID, playerNpcId uuid.UUID, m Model) []EquipmentEntity {
	entities := make([]EquipmentEntity, 0, len(m.Equipment()))
	for _, em := range m.Equipment() {
		entities = append(entities, EquipmentEntity{
			Id:          em.Id(),
			TenantId:    tenantId,
			PlayerNpcId: playerNpcId,
			Slot:        em.Slot(),
			ItemId:      em.ItemId(),
		})
	}
	return entities
}

// Migration sets up the player_npcs and player_npc_equipment tables.
func Migration(db *gorm.DB) error {
	if err := db.AutoMigrate(&Entity{}); err != nil {
		return err
	}
	return db.AutoMigrate(&EquipmentEntity{})
}
