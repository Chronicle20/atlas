package trade

import (
	"atlas-trades/rest"
	"net/http"
	"net/url"
	"sort"
	"strconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
)

const (
	GetRooms    = "get_trade_rooms"
	GetRoomById = "get_trade_room_by_id"
)

// maxPageSize is PRD §5's page-size cap for the room list. It is also the
// default: the list is an in-memory registry dump that a tenant holds few of,
// so the common case fits one page (docs/rest-pagination.md, game-capped shape).
// A request above the cap is a 400, never a silent clamp.
const maxPageSize = 100

// InitResource wires PRD §5's two read-only room endpoints:
// GET /trades/rooms and GET /trades/rooms/{roomId}. Both read the process-local
// registry, so they describe THIS pod's rooms — which is why atlas-trades runs
// single-replica (design §9). There are no write routes: rooms are created and
// mutated exclusively by Kafka commands.
func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		registerGet := rest.RegisterHandler(l)(si)

		r := router.PathPrefix("/trades/rooms").Subrouter()
		r.HandleFunc("", registerGet(GetRooms, handleGetRooms())).Methods(http.MethodGet)
		r.HandleFunc("/{roomId}", registerGet(GetRoomById, handleGetRoomById())).Methods(http.MethodGet)
	}
}

// roomFilters is PRD §5's filter[...] set for GET /trades/rooms. Each filter is
// optional; an absent one matches every room.
type roomFilters struct {
	characterId    character.Id
	hasCharacterId bool
	worldId        world.Id
	hasWorldId     bool
	channelId      channel.Id
	hasChannelId   bool
	mapId          _map.Id
	hasMapId       bool
}

// parseRoomFilters reads the four room filters. A filter that is present but
// does not parse — or does not fit its field's width — is an error rather than
// a silently dropped filter, which would answer a narrow question with every
// room in the tenant.
func parseRoomFilters(query url.Values) (roomFilters, error) {
	var f roomFilters
	var err error
	if f.characterId, f.hasCharacterId, err = parseUintFilter[character.Id](query, "filter[characterId]", 32); err != nil {
		return roomFilters{}, err
	}
	if f.worldId, f.hasWorldId, err = parseUintFilter[world.Id](query, "filter[worldId]", 8); err != nil {
		return roomFilters{}, err
	}
	if f.channelId, f.hasChannelId, err = parseUintFilter[channel.Id](query, "filter[channelId]", 8); err != nil {
		return roomFilters{}, err
	}
	if f.mapId, f.hasMapId, err = parseUintFilter[_map.Id](query, "filter[mapId]", 32); err != nil {
		return roomFilters{}, err
	}
	return f, nil
}

// unsignedId is the set of shared id types the room filters parse into.
type unsignedId interface {
	~uint8 | ~uint32
}

// parseUintFilter reads one optional unsigned filter. It reports whether the
// filter was supplied, and errors when it was supplied but unparseable.
func parseUintFilter[T unsignedId](query url.Values, name string, bits int) (T, bool, error) {
	raw := query.Get(name)
	if raw == "" {
		return 0, false, nil
	}
	v, err := strconv.ParseUint(raw, 10, bits)
	if err != nil {
		return 0, false, err
	}
	return T(v), true, nil
}

// matches reports whether the room satisfies every supplied filter. The
// character filter matches EITHER side, so a GM looking a character up finds
// the room whether they own it or were invited into it.
func (f roomFilters) matches(r Room) bool {
	if f.hasWorldId && r.Field().WorldId() != f.worldId {
		return false
	}
	if f.hasChannelId && r.Field().ChannelId() != f.channelId {
		return false
	}
	if f.hasMapId && r.Field().MapId() != f.mapId {
		return false
	}
	if f.hasCharacterId {
		if _, ok := r.ParticipantFor(f.characterId); !ok {
			return false
		}
	}
	return true
}

// handleGetRooms serves GET /trades/rooms — the tenant's live rooms, filtered,
// deterministically ordered and paged.
func handleGetRooms() rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()

			filters, err := parseRoomFilters(query)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, "invalid filter[characterId]/filter[worldId]/filter[channelId]/filter[mapId]")
				return
			}

			page, err := paginate.ParseParams(query, maxPageSize, maxPageSize)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
				return
			}

			var matched []Room
			for _, room := range NewProcessor(d.Logger(), d.Context()).RoomsForTenant() {
				if filters.matches(room) {
					matched = append(matched, room)
				}
			}

			// The registry is a map, so its iteration order is random. Sorting by
			// the room id — unique within a tenant — is what makes two requests
			// for the same page return the same rooms.
			sort.Slice(matched, func(i, j int) bool { return matched[i].Id().String() < matched[j].Id().String() })
			paged := paginate.Slice(matched, page)

			res, err := model.SliceMap(Transform)(model.FixedProvider(paged.Items))(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res, paginate.EnvelopeFor(paged), r)
		}
	}
}

// handleGetRoomById serves GET /trades/rooms/{roomId}. A settled or cancelled
// room has been removed from the registry, so it 404s rather than serving a
// stale snapshot (PRD §5).
func handleGetRoomById() rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseRoomId(d.Logger(), func(roomId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				room, ok := NewProcessor(d.Logger(), d.Context()).RoomById(roomId)
				if !ok {
					w.WriteHeader(http.StatusNotFound)
					return
				}

				res, err := Transform(room)
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
			}
		})
	}
}
