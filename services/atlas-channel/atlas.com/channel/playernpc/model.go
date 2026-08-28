// Package playernpc is atlas-channel's read client for atlas-player-npcs'
// PRD §5 REST surface: the deployed Player NPCs currently placed in a map,
// consumed to replay existing state to a character entering the map. It
// mirrors the shape of the neighbouring kite/ client (model.go, builder.go,
// rest.go, requests.go, processor.go) and follows the repo's immutable-model
// convention -- unexported fields, accessors, a Builder. No *_testhelpers.go:
// test setup goes through the Builder.
package playernpc

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Model is one deployed Player NPC as atlas-channel sees it -- PRD §5's
// attribute list verbatim, including overallJobRank (see rest.go's doc
// comment on RestModel for the PRD §5/§6 discrepancy this resolves).
type Model struct {
	id             uuid.UUID
	characterId    uint32
	name           string
	worldId        world.Id
	mapId          _map.Id
	scriptId       uint32
	objectId       uint32
	gender         byte
	skin           byte
	face           uint32
	hair           uint32
	jobId          job.Id
	x              int16
	cy             int16
	fh             uint16
	rx0            int16
	rx1            int16
	dir            byte
	worldRank      uint32
	overallRank    uint32
	worldJobRank   uint32
	overallJobRank uint32
	equipment      []EquipmentModel
	deployedAt     time.Time
}

func (m Model) Id() uuid.UUID          { return m.id }
func (m Model) CharacterId() uint32    { return m.characterId }
func (m Model) Name() string           { return m.name }
func (m Model) WorldId() world.Id      { return m.worldId }
func (m Model) MapId() _map.Id         { return m.mapId }
func (m Model) ScriptId() uint32       { return m.scriptId }
func (m Model) ObjectId() uint32       { return m.objectId }
func (m Model) Gender() byte           { return m.gender }
func (m Model) Skin() byte             { return m.skin }
func (m Model) Face() uint32           { return m.face }
func (m Model) Hair() uint32           { return m.hair }
func (m Model) JobId() job.Id          { return m.jobId }
func (m Model) X() int16               { return m.x }
func (m Model) Cy() int16              { return m.cy }
func (m Model) Fh() uint16             { return m.fh }
func (m Model) RX0() int16             { return m.rx0 }
func (m Model) RX1() int16             { return m.rx1 }
func (m Model) Dir() byte              { return m.dir }
func (m Model) WorldRank() uint32      { return m.worldRank }
func (m Model) OverallRank() uint32    { return m.overallRank }
func (m Model) WorldJobRank() uint32   { return m.worldJobRank }
func (m Model) OverallJobRank() uint32 { return m.overallJobRank }
func (m Model) Equipment() []EquipmentModel {
	return m.equipment
}
func (m Model) DeployedAt() time.Time { return m.deployedAt }

// AtPosition returns a copy of m relocated to the given coordinates,
// leaving every other attribute (including dir, which no reposition
// carries) untouched. REPOSITIONED delivers new coordinates for an
// object atlas-player-npcs may not have re-published yet, so folding
// them into the model keeps every packet the handler emits for that
// object -- respawn and controller grant alike -- on one snapshot.
func (m Model) AtPosition(x int16, cy int16, fh uint16, rx0 int16, rx1 int16) Model {
	m.x = x
	m.cy = cy
	m.fh = fh
	m.rx0 = rx0
	m.rx1 = rx1
	return m
}

// EquipmentModel is one frozen equipment slot on a deployed Player NPC.
type EquipmentModel struct {
	slot   int16
	itemId uint32
}

func (e EquipmentModel) Slot() int16    { return e.slot }
func (e EquipmentModel) ItemId() uint32 { return e.itemId }
