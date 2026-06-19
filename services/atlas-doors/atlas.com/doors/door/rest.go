package door

import (
	"fmt"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/point"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

// RestModel is the JSON:API resource model for a mystic door.
type RestModel struct {
	Id               string       `json:"-"`
	AreaDoorId       uint32       `json:"areaDoorId"`
	TownDoorId       uint32       `json:"townDoorId"`
	PairId           uint32       `json:"pairId"`
	OwnerCharacterId character.Id `json:"ownerCharacterId"`
	PartyId          uint32       `json:"partyId"`
	WorldId          world.Id     `json:"worldId"`
	ChannelId        channel.Id   `json:"channelId"`
	MapId            _map.Id      `json:"mapId"`
	Instance         uuid.UUID    `json:"instance"`
	TownMapId        _map.Id      `json:"townMapId"`
	Slot             byte         `json:"slot"`
	TownPortalId     uint32       `json:"townPortalId"`
	AreaX            point.X      `json:"areaX"`
	AreaY            point.Y      `json:"areaY"`
	TownX            point.X      `json:"townX"`
	TownY            point.Y      `json:"townY"`
	SkillId          skill.Id     `json:"skillId"`
	SkillLevel       byte         `json:"skillLevel"`
	ExpiresAt        time.Time    `json:"expiresAt"`
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func (r RestModel) GetName() string {
	return "doors"
}

// Transform maps a domain Model to a RestModel.
func Transform(m Model) (RestModel, error) {
	f := m.Field()
	return RestModel{
		Id:               fmt.Sprintf("%d", m.AreaDoorId()),
		AreaDoorId:       m.AreaDoorId(),
		TownDoorId:       m.TownDoorId(),
		PairId:           m.PairId(),
		OwnerCharacterId: m.OwnerCharacterId(),
		PartyId:          m.PartyId(),
		WorldId:          f.WorldId(),
		ChannelId:        f.ChannelId(),
		MapId:            f.MapId(),
		Instance:         f.Instance(),
		TownMapId:        m.TownMapId(),
		Slot:             m.Slot(),
		TownPortalId:     m.TownPortalId(),
		AreaX:            m.AreaX(),
		AreaY:            m.AreaY(),
		TownX:            m.TownX(),
		TownY:            m.TownY(),
		SkillId:          m.SkillId(),
		SkillLevel:       m.SkillLevel(),
		ExpiresAt:        m.ExpiresAt(),
	}, nil
}
