package monster

import (
	"atlas-party-quests/rest"
	"fmt"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/google/uuid"
)

const (
	MonstersInField = "worlds/%d/channels/%d/maps/%d/instances/%s/monsters"
)

func getBaseRequest() string {
	return requests.RootUrl("MONSTERS")
}

func requestDestroyInField(worldId world.Id, channelId channel.Id, mapId _map.Id, instance uuid.UUID) requests.EmptyBodyRequest {
	return rest.MakeDeleteRequest(fmt.Sprintf(getBaseRequest()+MonstersInField, worldId, channelId, mapId, instance.String()))
}

func requestSpawnInField(f field.Model, input SpawnInputRestModel) requests.Request[SpawnResponseRestModel] {
	return rest.MakePostRequest[SpawnResponseRestModel](fmt.Sprintf(getBaseRequest()+MonstersInField, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String()), input)
}

type SpawnInputRestModel struct {
	Id        string `json:"-"`
	MonsterId uint32 `json:"monsterId"`
	X         int16  `json:"x"`
	Y         int16  `json:"y"`
	Fh        int16  `json:"fh"`
	Team      int8   `json:"team"`
}

func (r SpawnInputRestModel) GetName() string {
	return "monsters"
}

func (r SpawnInputRestModel) GetID() string {
	return r.Id
}

type SpawnResponseRestModel struct {
	Id        string `json:"-"`
	UniqueId  uint32 `json:"uniqueId"`
	MonsterId uint32 `json:"monsterId"`
}

func (r SpawnResponseRestModel) GetName() string {
	return "monsters"
}

func (r SpawnResponseRestModel) GetID() string {
	return r.Id
}

func (r *SpawnResponseRestModel) SetID(id string) error {
	r.Id = id
	return nil
}
