package game

import (
	"atlas-mini-games/rest"
	"net/http"
	"sort"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
)

const (
	GetGamesInField     = "get_games_in_field"
	GetGameForCharacter = "get_game_for_character"
)

// InitResource wires GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/games
// (chalkboards resource.go:55-84 shape), returning every mini-game room
// currently registered in that field so atlas-channel can reconcile its
// local view on portal-enter (task-19 REST client).
// InitResource wires GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/games.
// The db handle is curried in at wiring time so the handler can construct the
// game.Processor (which owns the rooms-in-field read); the read itself is
// served from the in-memory registry.
func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)

			r := router.PathPrefix("/worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/games").Subrouter()
			r.HandleFunc("", registerGet(GetGamesInField, handleGetGamesInField(db))).Methods(http.MethodGet)

			cr := router.PathPrefix("/characters/{characterId}/games").Subrouter()
			cr.HandleFunc("", registerGet(GetGameForCharacter, handleGetGameForCharacter(db))).Methods(http.MethodGet)
		}
	}
}

// handleGetGameForCharacter serves GET /characters/{characterId}/games — the
// (0-or-1) room the character is currently seated in (owner or visitor). It
// backs atlas-channel's membership check that blocks cash-shop / MTS entry
// while in a mini-game room. A collection (not a single resource) so the empty
// case is a normal empty list rather than a 404, matching the channel's
// SliceProvider read.
func handleGetGameForCharacter(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				var rooms []Room
				if room, ok := NewProcessor(d.Logger(), d.Context(), db).RoomForCharacter(characterId); ok {
					rooms = append(rooms, room)
				}

				res, err := model.SliceMap(Transform)(model.FixedProvider(rooms))(model.ParallelMap())()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
			}
		})
	}
}

func handleGetGamesInField(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
			return rest.ParseChannelId(d.Logger(), func(channelId channel.Id) http.HandlerFunc {
				return rest.ParseMapId(d.Logger(), func(mapId _map.Id) http.HandlerFunc {
					return rest.ParseInstanceId(d.Logger(), func(instanceId uuid.UUID) http.HandlerFunc {
						return func(w http.ResponseWriter, r *http.Request) {
							// Group C game-capped list: a field holds few rooms, so
							// default the page size to the cap (chairs resource.go shape).
							page, err := paginate.ParseParams(r.URL.Query(), paginate.MaxPageSize, paginate.MaxPageSize)
							if err != nil {
								server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
								return
							}

							f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instanceId).Build()

							rooms := NewProcessor(d.Logger(), d.Context(), db).RoomsInField(f)

							// Sort by room Id (unique — one room per owner, Id == OwnerId)
							// so the page slice is deterministic across requests.
							sorted := make([]Room, len(rooms))
							copy(sorted, rooms)
							sort.Slice(sorted, func(i, j int) bool { return sorted[i].Id() < sorted[j].Id() })
							paged := paginate.Slice(sorted, page)

							res, err := model.SliceMap(Transform)(model.FixedProvider(paged.Items))(model.ParallelMap())()
							if err != nil {
								d.Logger().WithError(err).Errorf("Creating REST model.")
								server.WriteErrorResponse(d.Logger())(w)(err)
								return
							}

							query := r.URL.Query()
							queryParams := jsonapi.ParseQueryFields(&query)
							server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res, paginate.EnvelopeFor(paged), r)
						}
					})
				})
			})
		})
	}
}
