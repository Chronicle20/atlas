package npc

import (
	"atlas-maps/rest"
	"net/http"

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
)

const (
	getNpcsInMap   = "get_npcs_in_map"
	createNpcInMap = "create_npc_in_map"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		r := router.PathPrefix("/worlds").Subrouter()
		r.HandleFunc("/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/npcs", rest.RegisterHandler(l)(si)(getNpcsInMap, handleGetNpcsInMap)).Methods(http.MethodGet)
		r.HandleFunc("/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/npcs", rest.RegisterInputHandler[RestModel](l)(si)(createNpcInMap, handleCreateNpcInMap)).Methods(http.MethodPost)
	}
}

func handleGetNpcsInMap(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
		return rest.ParseChannelId(d.Logger(), func(channelId channel.Id) http.HandlerFunc {
			return rest.ParseMapId(d.Logger(), func(mapId _map.Id) http.HandlerFunc {
				return rest.ParseInstanceId(d.Logger(), func(instanceId uuid.UUID) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instanceId).Build()
						p := NewProcessor(d.Logger(), d.Context())
						ns, err := p.GetInField(f)
						if err != nil {
							d.Logger().WithError(err).Errorf("Unable to retrieve npcs in field.")
							w.WriteHeader(http.StatusInternalServerError)
							return
						}

						res, err := model.SliceMap(Transform)(model.FixedProvider(ns))()()
						if err != nil {
							d.Logger().WithError(err).Errorf("Creating REST model.")
							server.WriteErrorResponse(d.Logger())(w)(err)
							return
						}

						server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(r.URL.Query())(res)
					}
				})
			})
		})
	})
}

func handleCreateNpcInMap(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
		return rest.ParseChannelId(d.Logger(), func(channelId channel.Id) http.HandlerFunc {
			return rest.ParseMapId(d.Logger(), func(mapId _map.Id) http.HandlerFunc {
				return rest.ParseInstanceId(d.Logger(), func(instanceId uuid.UUID) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instanceId).Build()
						p := NewProcessor(d.Logger(), d.Context())
						m, err := p.Create(f, input)
						if err != nil {
							d.Logger().WithError(err).Errorf("Unable to create npc.")
							w.WriteHeader(http.StatusBadRequest)
							return
						}
						if m.UniqueId() == 0 {
							// SpawnIfAbsent suppressed the spawn (see
							// Processor.Create); a zero Model with a nil error is
							// not an error, but a 200 with a zero-valued npc body
							// would be misleading.
							w.WriteHeader(http.StatusNoContent)
							return
						}
						res, err := model.Map(Transform)(model.FixedProvider(m))()
						if err != nil {
							d.Logger().WithError(err).Errorf("Creating REST model.")
							w.WriteHeader(http.StatusInternalServerError)
							return
						}

						server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(r.URL.Query())(res)
					}
				})
			})
		})
	})
}
