package transport

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const BaseUrl = "TRANSPORTS_URL"

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, BaseUrl)
}

// allInstanceRoutesUrl is a bare URL (not a requests.Request) because the
// list is now paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func allInstanceRoutesUrl(ctx context.Context) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return root + "transports/instance-routes", nil
}

func identityTransform(r RouteRestModel) (RouteRestModel, error) {
	return r, nil
}

// GetRouteByName fetches all routes and finds the one matching the given
// name. atlas-transports' GET /transports/instance-routes is now paginated
// (task-117); this scans every route in the tenant by name, a genuine
// semantic-all consumer, so it drains every page rather than fetching just
// the first.
func GetRouteByName(l logrus.FieldLogger, ctx context.Context) func(name string) (RouteRestModel, error) {
	return func(name string) (RouteRestModel, error) {
		url, err := allInstanceRoutesUrl(ctx)
		if err != nil {
			return RouteRestModel{}, err
		}
		resp, err := requests.DrainProvider[RouteRestModel, RouteRestModel](l, ctx)(url, 250, identityTransform, model.Filters[RouteRestModel]())()
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

		root, err := getBaseRequest(ctx)
		if err != nil {
			return err
		}
		url := fmt.Sprintf("%stransports/instance-routes/%s/start", root, routeId.String())
		// Use struct{} as the response type since this endpoint returns 204 No Content
		_, err = requests.PostRequest[struct{}](url, body)(l, ctx)
		return err
	}
}
