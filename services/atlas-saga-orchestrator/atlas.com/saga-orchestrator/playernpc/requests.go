package playernpc

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Resource is atlas-maps' durable character-location lookup. deploy_player_npc
// (FR-6.2) uses it to resolve DeployPlayerNpcPayload's default map (MapId ==
// nil means "deploy on the character's current map") -- the payload cannot
// simply omit WorldId/MapId the way StartNpcConversationPayload does, because
// this step's originator (a conversation script) may not know the
// character's current position.
const Resource = "characters/%d/location"

// LocationRestModel mirrors the JSON:API shape returned by atlas-maps'
// GET /characters/{id}/location endpoint.
type LocationRestModel struct {
	Id        uint32     `json:"-"`
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
}

func (r LocationRestModel) GetName() string { return "character-locations" }
func (r LocationRestModel) GetID() string   { return fmt.Sprintf("%d", r.Id) }
func (r *LocationRestModel) SetID(s string) error {
	var id uint32
	if _, err := fmt.Sscanf(s, "%d", &id); err != nil {
		return err
	}
	r.Id = id
	return nil
}
func (r *LocationRestModel) SetToOneReferenceID(_, _ string) error            { return nil }
func (r *LocationRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "MAPS")
}

func requestLocationByCharacterId(ctx context.Context, characterId uint32) requests.Request[LocationRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[LocationRestModel](err)
	}
	return requests.GetRequest[LocationRestModel](fmt.Sprintf(root+Resource, characterId))
}
