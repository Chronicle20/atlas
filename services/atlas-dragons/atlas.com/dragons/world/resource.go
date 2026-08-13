package world

import (
	"atlas-dragons/dragon"
	"atlas-dragons/rest"
	"net/http"
	"sort"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
)

const (
	getDragonsInMap = "get_dragons_in_map"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		r := router.PathPrefix("/worlds").Subrouter()
		r.HandleFunc("/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/dragons", rest.RegisterHandler(l)(si)(getDragonsInMap, handleGetDragonsInMap)).Methods(http.MethodGet)
	}
}

func handleGetDragonsInMap(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
		return rest.ParseChannelId(d.Logger(), func(channelId channel.Id) http.HandlerFunc {
			return rest.ParseMapId(d.Logger(), func(mapId _map.Id) http.HandlerFunc {
				return rest.ParseInstanceId(d.Logger(), func(instance uuid.UUID) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						page, err := paginate.ParseParams(r.URL.Query(), paginate.MaxPageSize, paginate.MaxPageSize)
						if err != nil {
							server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
							return
						}

						f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instance).Build()
						p := dragon.NewProcessor(d.Logger(), d.Context())
						ms, err := p.GetInField(f)
						if err != nil {
							d.Logger().WithError(err).Errorf("Unable to retrieve dragons in field.")
							w.WriteHeader(http.StatusInternalServerError)
							return
						}

						sorted := make([]dragon.Model, len(ms))
						copy(sorted, ms)
						sort.Slice(sorted, func(i, j int) bool { return sorted[i].OwnerCharacterId() < sorted[j].OwnerCharacterId() })
						paged := paginate.Slice(sorted, page)

						res, err := model.SliceMap(dragon.Transform)(model.FixedProvider(paged.Items))(model.ParallelMap())()
						if err != nil {
							d.Logger().WithError(err).Errorf("Creating REST model.")
							w.WriteHeader(http.StatusInternalServerError)
							return
						}

						server.MarshalPaginatedResponse[[]dragon.RestModel](d.Logger())(w)(c.ServerInformation())(r.URL.Query())(res, paginate.EnvelopeFor(paged), r)
					}
				})
			})
		})
	})
}
