package _map

import (
	"atlas-maps/map/monster"
	"atlas-maps/rest"
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
	getCharactersInMap             = "get_characters_in_map"
	getCharactersInMapAllInstances = "get_characters_in_map_all_instances"
	resetField                     = "reset_field"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		r := router.PathPrefix("/worlds").Subrouter()
		r.HandleFunc("/{worldId}/channels/{channelId}/maps/{mapId}/characters", rest.RegisterHandler(l)(si)(getCharactersInMapAllInstances, handleGetCharactersInMapAllInstances)).Methods(http.MethodGet)
		r.HandleFunc("/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/characters", rest.RegisterHandler(l)(si)(getCharactersInMap, handleGetCharactersInMap)).Methods(http.MethodGet)
		r.HandleFunc("/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/reset", rest.RegisterInputHandler[ResetFieldInputRestModel](l)(si)(resetField, handleResetField)).Methods(http.MethodPost)
	}
}

func handleGetCharactersInMap(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
		return rest.ParseChannelId(d.Logger(), func(channelId channel.Id) http.HandlerFunc {
			return rest.ParseMapId(d.Logger(), func(mapId _map.Id) http.HandlerFunc {
				return rest.ParseInstanceId(d.Logger(), func(instanceId uuid.UUID) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						page, err := paginate.ParseParams(r.URL.Query(), paginate.MaxPageSize, paginate.MaxPageSize)
						if err != nil {
							server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
							return
						}

						transactionId := uuid.New()
						f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instanceId).Build()
						mp := NewProcessor(d.Logger(), d.Context(), nil, nil)
						ids, err := mp.GetCharactersInMap(transactionId, f)
						if err != nil {
							server.WriteErrorResponse(d.Logger())(w)(err)
							return
						}

						sorted := make([]uint32, len(ids))
						copy(sorted, ids)
						sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
						paged := paginate.Slice(sorted, page)

						res, err := model.SliceMap(Transform)(model.FixedProvider(paged.Items))(model.ParallelMap())()
						if err != nil {
							d.Logger().WithError(err).Errorf("Creating REST model.")
							server.WriteErrorResponse(d.Logger())(w)(err)
							return
						}

						server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(r.URL.Query())(res, paginate.EnvelopeFor(paged), r)
					}
				})
			})
		})
	})
}

func handleGetCharactersInMapAllInstances(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
		return rest.ParseChannelId(d.Logger(), func(channelId channel.Id) http.HandlerFunc {
			return rest.ParseMapId(d.Logger(), func(mapId _map.Id) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					page, err := paginate.ParseParams(r.URL.Query(), paginate.MaxPageSize, paginate.MaxPageSize)
					if err != nil {
						server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
						return
					}

					transactionId := uuid.New()
					mp := NewProcessor(d.Logger(), d.Context(), nil, nil)
					ids, err := mp.GetCharactersInMapAllInstances(transactionId, worldId, channelId, mapId)
					if err != nil {
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					sorted := make([]uint32, len(ids))
					copy(sorted, ids)
					sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
					paged := paginate.Slice(sorted, page)

					res, err := model.SliceMap(Transform)(model.FixedProvider(paged.Items))(model.ParallelMap())()
					if err != nil {
						d.Logger().WithError(err).Errorf("Creating REST model.")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(r.URL.Query())(res, paginate.EnvelopeFor(paged), r)
				}
			})
		})
	})
}

// handleResetField is Cosmic's resetPQ(difficulty): clear the field's
// monsters and restore its spawn points. See
// map/monster.ProcessorImpl.ResetField for the exact composition and the
// documented object-class/difficulty scope.
func handleResetField(d *rest.HandlerDependency, _ *rest.HandlerContext, i ResetFieldInputRestModel) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
		return rest.ParseChannelId(d.Logger(), func(channelId channel.Id) http.HandlerFunc {
			return rest.ParseMapId(d.Logger(), func(mapId _map.Id) http.HandlerFunc {
				return rest.ParseInstanceId(d.Logger(), func(instanceId uuid.UUID) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instanceId).Build()
						err := monster.NewProcessor(d.Logger(), d.Context()).ResetField(f, i.Difficulty)
						if err != nil {
							d.Logger().WithError(err).Errorf("Unable to reset field [%s].", f.Id())
							server.WriteErrorResponse(d.Logger())(w)(err)
							return
						}

						w.WriteHeader(http.StatusNoContent)
					}
				})
			})
		})
	})
}
