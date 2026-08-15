package occurrence

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Entity is the GORM persistence record for an event occurrence.
type Entity struct {
	ID                uuid.UUID `gorm:"primaryKey;column:id;type:uuid"`
	TenantID          uuid.UUID `gorm:"column:tenant_id;type:uuid;not null"`
	EventDefinitionID uuid.UUID `gorm:"column:event_definition_id;type:uuid;not null"`
	Type              string    `gorm:"column:type;not null"`
	State             string    `gorm:"column:state;not null"`
	Stage             string    `gorm:"column:stage"`
	Context           string    `gorm:"column:context;type:jsonb;not null;default:'{}'"`
	// worldId, channelId and voyageId are PROMOTED out of context to scalar
	// columns so the gameplay queries of FR-API7 are index-served rather than
	// jsonb scans (FR-API8, FR-O7).
	WorldID          *uint8     `gorm:"column:world_id"`
	ChannelID        *uint8     `gorm:"column:channel_id"`
	VoyageID         *uuid.UUID `gorm:"column:voyage_id;type:uuid"`
	ConcurrencyKey   string     `gorm:"column:concurrency_key;not null;default:''"`
	StartedAt        time.Time  `gorm:"column:started_at;not null"`
	NextTransitionAt *time.Time `gorm:"column:next_transition_at"`
	CompletedAt      *time.Time `gorm:"column:completed_at"`
	CompletionReason string     `gorm:"column:completion_reason"`
}

func (Entity) TableName() string { return "event_occurrence" }

// MapEntity is the FR-API8 child table. Visual distinguishes the deck (gets the
// enemy-ship visual) from the cabin (counts toward "aboard", gets nothing) at
// QUERY time, so FR-B13/FR-B14 is a predicate rather than a branch in
// atlas-channel.
type MapEntity struct {
	OccurrenceID uuid.UUID `gorm:"primaryKey;column:occurrence_id;type:uuid"`
	MapID        uint32    `gorm:"primaryKey;column:map_id"`
	Visual       bool      `gorm:"column:visual;not null;default:false"`
}

func (MapEntity) TableName() string { return "event_occurrence_map" }

// MonsterEntity is the durable form of "are any left?" (FR-B18). A SET, not a
// counter: a counter cannot be made idempotent under Kafka redelivery, and
// cannot tolerate KILLED arriving before CREATED (design §9.5, §14 A4).
type MonsterEntity struct {
	OccurrenceID uuid.UUID `gorm:"primaryKey;column:occurrence_id;type:uuid"`
	UniqueID     uint32    `gorm:"primaryKey;column:unique_id"`
	MonsterID    uint32    `gorm:"column:monster_id;not null"`
	// No `default:true` here deliberately: GORM's zero-value-omission for
	// defaulted columns would substitute the DB default whenever an explicit
	// Alive:false is inserted (ObserveMonsterGone), silently resurrecting a
	// KILLED-before-CREATED row. Every write path sets Alive explicitly.
	Alive      bool      `gorm:"column:alive;not null"`
	ObservedAt time.Time `gorm:"column:observed_at;not null"`
}

func (MonsterEntity) TableName() string { return "event_occurrence_monster" }

// MigrateTable creates the occurrence tables and the partial unique index that
// backs ErrConcurrencyKeyTaken: at most one row per (tenant, concurrency key)
// may exist, but an empty key (the "unbounded" opt-out) is excluded from the
// constraint entirely. Postgres and SQLite share this partial-index syntax.
func MigrateTable(db *gorm.DB) error {
	if err := db.AutoMigrate(&Entity{}, &MapEntity{}, &MonsterEntity{}); err != nil {
		return err
	}
	return db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS ux_event_occurrence_concurrency_key ` +
		`ON event_occurrence (tenant_id, concurrency_key) WHERE concurrency_key <> ''`).Error
}

// Make converts a persistence Entity into a domain Model.
func Make(e Entity) (Model, error) {
	b := NewBuilder(e.EventDefinitionID, e.Type).
		SetId(e.ID).
		SetState(e.State).
		SetStage(e.Stage).
		SetContext(json.RawMessage(e.Context)).
		SetConcurrencyKey(e.ConcurrencyKey).
		SetStartedAt(e.StartedAt).
		SetNextTransitionAt(e.NextTransitionAt).
		SetCompletedAt(e.CompletedAt).
		SetCompletionReason(e.CompletionReason)

	if e.WorldID != nil {
		b.SetWorldId(world.Id(*e.WorldID))
	}
	if e.ChannelID != nil {
		b.SetChannelId(channel.Id(*e.ChannelID))
	}
	if e.VoyageID != nil {
		b.SetVoyageId(*e.VoyageID)
	}

	return b.Build()
}

// ToEntity converts a domain Model into a persistence Entity, stamping the
// tenant id. VoyageID is left nil for uuid.Nil (the "no voyage scope" sentinel,
// per registry.Seed's contract) rather than persisting the nil UUID.
func ToEntity(m Model, tenantId uuid.UUID) (Entity, error) {
	ctx := m.Context()
	if ctx == nil {
		ctx = json.RawMessage("{}")
	}

	worldId := uint8(m.WorldId())
	channelId := uint8(m.ChannelId())

	e := Entity{
		ID:                m.Id(),
		TenantID:          tenantId,
		EventDefinitionID: m.DefinitionId(),
		Type:              m.Type(),
		State:             m.State(),
		Stage:             m.Stage(),
		Context:           string(ctx),
		WorldID:           &worldId,
		ChannelID:         &channelId,
		ConcurrencyKey:    m.ConcurrencyKey(),
		StartedAt:         m.StartedAt(),
		NextTransitionAt:  m.NextTransitionAt(),
		CompletedAt:       m.CompletedAt(),
		CompletionReason:  m.CompletionReason(),
	}

	if m.VoyageId() != uuid.Nil {
		voyageId := m.VoyageId()
		e.VoyageID = &voyageId
	}

	return e, nil
}
