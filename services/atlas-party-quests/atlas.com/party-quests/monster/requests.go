package monster

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	MonstersInField = "worlds/%d/channels/%d/maps/%d/instances/%s/monsters"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "MONSTERS")
}

func requestDestroyInField(ctx context.Context, worldId world.Id, channelId channel.Id, mapId _map.Id, instance uuid.UUID) requests.EmptyBodyRequest {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return func(_ logrus.FieldLogger, _ context.Context) error { return err }
	}
	return requests.DeleteRequest(fmt.Sprintf(root+MonstersInField, worldId, channelId, mapId, instance.String()))
}

func requestSpawnInField(ctx context.Context, f field.Model, input SpawnInputRestModel) requests.Request[SpawnResponseRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[SpawnResponseRestModel](err)
	}
	return requests.PostRequest[SpawnResponseRestModel](fmt.Sprintf(root+MonstersInField, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String()), input)
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
