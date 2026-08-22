// Package playernpc is the persistence layer for a deployed Player NPC:
// identity, placement, frozen appearance/equipment, frozen ranks, and
// resolved position. It follows the repo's immutable-model convention
// (services/atlas-notes/atlas.com/notes/note) -- unexported fields,
// accessors, a Builder, and Make/ToEntity between the domain model and
// the GORM entity. No *_testhelpers.go: test setup goes through the
// Builder.
package playernpc

import (
	"time"

	"github.com/google/uuid"
)

// Model is the immutable domain representation of a deployed Player NPC.
// rx0/rx1 are computed once (Builder.SetX, design §3.1) and carried
// verbatim thereafter -- they are never recomputed on read.
type Model struct {
	id             uuid.UUID
	characterId    uint32
	name           string
	worldId        byte
	mapId          uint32
	scriptId       uint32
	objectId       uint32
	gender         byte
	skin           byte
	face           uint32
	hair           uint32
	jobId          uint16
	x              int16
	cy             int16
	fh             uint16
	rx0            int16
	rx1            int16
	dir            byte
	step           byte
	worldRank      uint32
	overallRank    uint32
	worldJobRank   uint32
	overallJobRank uint32
	createdAt      time.Time
	updatedAt      time.Time
	equipment      []EquipmentModel
}

func (m Model) Id() uuid.UUID       { return m.id }
func (m Model) CharacterId() uint32 { return m.characterId }
func (m Model) Name() string        { return m.name }
func (m Model) WorldId() byte       { return m.worldId }
func (m Model) MapId() uint32       { return m.mapId }
func (m Model) ScriptId() uint32    { return m.scriptId }
func (m Model) ObjectId() uint32    { return m.objectId }
func (m Model) Gender() byte        { return m.gender }
func (m Model) Skin() byte          { return m.skin }
func (m Model) Face() uint32        { return m.face }
func (m Model) Hair() uint32        { return m.hair }
func (m Model) JobId() uint16       { return m.jobId }
func (m Model) X() int16            { return m.x }
func (m Model) Cy() int16           { return m.cy }
func (m Model) Fh() uint16          { return m.fh }
func (m Model) RX0() int16          { return m.rx0 }
func (m Model) RX1() int16          { return m.rx1 }
func (m Model) Dir() byte           { return m.dir }

// Step is the grid/podium positioner step this NPC was last placed at
// (design 5.1/5.2). It is persisted, not recomputed on read, per Task 15's
// resolution of design 3.1's open question -- see processor.go's package
// doc for the rationale.
func (m Model) Step() byte             { return m.step }
func (m Model) WorldRank() uint32      { return m.worldRank }
func (m Model) OverallRank() uint32    { return m.overallRank }
func (m Model) WorldJobRank() uint32   { return m.worldJobRank }
func (m Model) OverallJobRank() uint32 { return m.overallJobRank }
func (m Model) CreatedAt() time.Time   { return m.createdAt }
func (m Model) UpdatedAt() time.Time   { return m.updatedAt }
func (m Model) Equipment() []EquipmentModel {
	return m.equipment
}

// EquipmentModel is one frozen equipment slot on a deployed Player NPC.
type EquipmentModel struct {
	id     uuid.UUID
	slot   int16
	itemId uint32
}

func (e EquipmentModel) Id() uuid.UUID  { return e.id }
func (e EquipmentModel) Slot() int16    { return e.slot }
func (e EquipmentModel) ItemId() uint32 { return e.itemId }
