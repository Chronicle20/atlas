package dragon

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Model is the channel-side view of a dragon fetched from atlas-dragons.
// X/Y are int32 — SPAWN_DRAGON encodes 4-byte coordinates.
type Model struct {
	ownerCharacterId uint32
	x                int32
	y                int32
	stance           byte
	jobId            uint16
}

func (m Model) OwnerCharacterId() uint32 { return m.ownerCharacterId }
func (m Model) X() int32                 { return m.x }
func (m Model) Y() int32                 { return m.y }
func (m Model) Stance() byte             { return m.stance }
func (m Model) JobId() uint16            { return m.jobId }

type RestModel struct {
	Id               string     `json:"-"`
	OwnerCharacterId uint32     `json:"ownerCharacterId"`
	X                int32      `json:"x"`
	Y                int32      `json:"y"`
	Stance           byte       `json:"stance"`
	JobId            uint16     `json:"jobId"`
	WorldId          world.Id   `json:"worldId"`
	ChannelId        channel.Id `json:"channelId"`
	MapId            _map.Id    `json:"mapId"`
	Instance         uuid.UUID  `json:"instance"`
}

func (r RestModel) GetName() string { return "dragons" }

func (r RestModel) GetID() string { return r.Id }

func (r *RestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

func Extract(r RestModel) (Model, error) {
	return Model{
		ownerCharacterId: r.OwnerCharacterId,
		x:                r.X,
		y:                r.Y,
		stance:           r.Stance,
		jobId:            r.JobId,
	}, nil
}
