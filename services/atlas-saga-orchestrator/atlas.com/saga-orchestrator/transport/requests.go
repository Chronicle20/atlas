package transport

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const BaseUrl = "TRANSPORTS_URL"

func getBaseRequest() string {
	return requests.RootUrl(BaseUrl)
}

// requestAllRoutes fetches all instance routes from atlas-transports
func requestAllRoutes() requests.Request[[]RouteRestModel] {
	return requests.GetRequest[[]RouteRestModel](getBaseRequest() + "transports/instance-routes")
}

// GetRouteByName fetches all routes and finds the one matching the given name
func GetRouteByName(l logrus.FieldLogger, ctx context.Context) func(name string) (RouteRestModel, error) {
	return func(name string) (RouteRestModel, error) {
		resp, err := requestAllRoutes()(l, ctx)
		if err != nil {
			return RouteRestModel{}, fmt.Errorf("failed to fetch instance routes: %w", err)
		}

		for _, route := range resp {
			if route.Name == name {
				return route, nil
			}
		}

		return RouteRestModel{}, fmt.Errorf("route not found: %s", name)
	}
}

// StartTransport calls the transport service to start a transport for a character
func StartTransport(l logrus.FieldLogger, ctx context.Context) func(routeId uuid.UUID, characterId uint32, worldId world.Id, channelId channel.Id) error {
	return func(routeId uuid.UUID, characterId uint32, worldId world.Id, channelId channel.Id) error {
		body := StartTransportRestModel{
			CharacterId: characterId,
			WorldId:     worldId,
			ChannelId:   channelId,
		}

		url := fmt.Sprintf("%stransports/instance-routes/%s/start", getBaseRequest(), routeId.String())
		// Use struct{} as the response type since this endpoint returns 204 No Content
		_, err := requests.PostRequest[struct{}](url, body)(l, ctx)
		return err
	}
}
