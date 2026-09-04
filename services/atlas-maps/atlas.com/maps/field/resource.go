package field

import (
	"atlas-maps/rest"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const getFields = "get_fields"

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		r := router.PathPrefix("/fields").Subrouter()
		r.HandleFunc("", rest.RegisterHandler(l)(si)(getFields, handleGetFields)).Methods(http.MethodGet)
	}
}

// parseFilters extracts optional filter[worldId]/filter[channelId]/filter[mapId]
// query parameters. An absent key means no constraint (nil); a present but
// unparseable value is an error.
func parseFilters(q url.Values) (*world.Id, *channel.Id, *_map.Id, error) {
	var worldId *world.Id
	if raw := q.Get("filter[worldId]"); raw != "" {
		v, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return nil, nil, nil, err
		}
		id := world.Id(v)
		worldId = &id
	}

	var channelId *channel.Id
	if raw := q.Get("filter[channelId]"); raw != "" {
		v, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return nil, nil, nil, err
		}
		id := channel.Id(v)
		channelId = &id
	}

	var mapId *_map.Id
	if raw := q.Get("filter[mapId]"); raw != "" {
		v, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return nil, nil, nil, err
		}
		id := _map.Id(v)
		mapId = &id
	}

	return worldId, channelId, mapId, nil
}

func handleGetFields(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, err := paginate.ParseParams(r.URL.Query(), paginate.MaxPageSize, paginate.MaxPageSize)
		if err != nil {
			server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
			return
		}

		t := tenant.MustFromContext(d.Context())

		worldId, channelId, mapId, err := parseFilters(r.URL.Query())
		if err != nil {
			d.Logger().WithFields(logrus.Fields{
				"tenant":            t.Id(),
				"filter[worldId]":   r.URL.Query().Get("filter[worldId]"),
				"filter[channelId]": r.URL.Query().Get("filter[channelId]"),
				"filter[mapId]":     r.URL.Query().Get("filter[mapId]"),
			}).WithError(err).Errorf("Failed to enumerate fields: invalid filter value.")
			server.WriteBadRequest(d.Logger(), w, "invalid filter value")
			return
		}

		occ := NewProcessor(d.Logger(), d.Context()).GetFields(t, worldId, channelId, mapId)

		models, err := TransformSlice(occ)
		if err != nil {
			d.Logger().WithError(err).Errorf("Failed to enumerate fields: unable to build REST models.")
			server.WriteErrorResponse(d.Logger())(w)(err)
			return
		}

		paged := paginate.Slice(models, page)

		server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(r.URL.Query())(paged.Items, paginate.EnvelopeFor(paged), r)
	}
}
