package door

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type RestModel struct {
	Id               string     `json:"-"`
	AreaDoorId       uint32     `json:"areaDoorId"`
	TownDoorId       uint32     `json:"townDoorId"`
	PairId           uint32     `json:"pairId"`
	OwnerCharacterId uint32     `json:"ownerCharacterId"`
	PartyId          uint32     `json:"partyId"`
	WorldId          world.Id   `json:"worldId"`
	ChannelId        channel.Id `json:"channelId"`
	MapId            _map.Id    `json:"mapId"`
	Instance         uuid.UUID  `json:"instance"`
	TownMapId        _map.Id    `json:"townMapId"`
	Slot             byte       `json:"slot"`
	TownPortalId     uint32     `json:"townPortalId"`
	AreaX            int16      `json:"areaX"`
	AreaY            int16      `json:"areaY"`
	TownX            int16      `json:"townX"`
	TownY            int16      `json:"townY"`
	SkillId          uint32     `json:"skillId"`
	SkillLevel       byte       `json:"skillLevel"`
	ExpiresAt        time.Time  `json:"expiresAt"`
}

func (r RestModel) GetName() string { return "doors" }

func (r RestModel) GetID() string { return r.Id }

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// SetToOneReferenceID satisfies the api2go UnmarshalToOneRelations interface.
func (r *RestModel) SetToOneReferenceID(_ string, _ string) error { return nil }

// SetToManyReferenceIDs satisfies the api2go UnmarshalToManyRelations interface.
func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:               rm.Id,
		areaDoorId:       rm.AreaDoorId,
		townDoorId:       rm.TownDoorId,
		pairId:           rm.PairId,
		ownerCharacterId: rm.OwnerCharacterId,
		partyId:          rm.PartyId,
		field:            field.NewBuilder(rm.WorldId, rm.ChannelId, rm.MapId).SetInstance(rm.Instance).Build(),
		townMapId:        rm.TownMapId,
		slot:             rm.Slot,
		townPortalId:     rm.TownPortalId,
		areaX:            rm.AreaX,
		areaY:            rm.AreaY,
		townX:            rm.TownX,
		townY:            rm.TownY,
		skillId:          rm.SkillId,
		skillLevel:       rm.SkillLevel,
		expiresAt:        rm.ExpiresAt,
	}, nil
}
