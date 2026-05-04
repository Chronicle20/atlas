package location

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const (
	Resource = "characters/%d/location"
)

// RestModel mirrors the JSON:API shape returned by atlas-maps's
// GET /characters/{id}/location endpoint. Required no-op relationship
// stubs are implemented per the api2go contract (see libs/atlas-rest CLAUDE.md).
type RestModel struct {
	Id        uint32     `json:"-"`
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
}

func (r RestModel) GetName() string { return "character-locations" }

func (r RestModel) GetID() string { return strconv.FormatUint(uint64(r.Id), 10) }

func (r *RestModel) SetID(s string) error {
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return err
	}
	r.Id = uint32(v)
	return nil
}

func (r *RestModel) SetToOneReferenceID(_, _ string) error            { return nil }
func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

func getBaseRequest() string {
	return requests.RootUrl("MAPS")
}

func requestByCharacterId(characterId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+Resource, characterId))
}

// GetField returns the durable field stored in atlas-maps for the given
// character. Caller must pass a logger and a context with tenant.
func GetField(l logrus.FieldLogger, ctx context.Context, characterId uint32) (field.Model, error) {
	rm, err := requestByCharacterId(characterId)(l, ctx)
	if err != nil {
		return field.Model{}, err
	}
	return field.NewBuilder(rm.WorldId, rm.ChannelId, rm.MapId).SetInstance(rm.Instance).Build(), nil
}
