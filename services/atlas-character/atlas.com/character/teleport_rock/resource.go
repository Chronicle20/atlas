package teleport_rock

import (
	"atlas-character/rest"
	"context"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

// WorldIdOf resolves a character's current worldId. The teleport_rock package
// cannot import the character package directly (character already imports
// teleport_rock, which would create an import cycle), so InitResource takes
// this as an injected dependency; main.go wires it via character.NewProcessor.
// It takes the request-scoped logger (not a captured bootstrap logger) so the
// resolver's downstream processor call carries the same originator/tenant/span
// fields as the rest of the request.
type WorldIdOf func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (world.Id, error)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) func(worldIdOf WorldIdOf) server.RouteInitializer {
	return func(db *gorm.DB) func(worldIdOf WorldIdOf) server.RouteInitializer {
		return func(worldIdOf WorldIdOf) server.RouteInitializer {
			return func(router *mux.Router, l logrus.FieldLogger) {
				registerGet := rest.RegisterHandler(l)(db)(si)
				r := router.PathPrefix("/characters/{characterId}/teleport-rock-maps").Subrouter()
				r.HandleFunc("", registerGet("get_teleport_rock_maps", handleGetTeleportRockMaps)).Methods(http.MethodGet)
				r.HandleFunc("", rest.RegisterInputHandler[AddMapInputRestModel](l)(db)(si)("add_teleport_rock_map", handleAddTeleportRockMap(worldIdOf))).Methods(http.MethodPost)
				r.HandleFunc("/{list}/{mapId}", registerGet("remove_teleport_rock_map", handleRemoveTeleportRockMap(worldIdOf))).Methods(http.MethodDelete)
			}
		}
	}
}

func handleGetTeleportRockMaps(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			m, err := NewProcessor(d.Logger(), d.Context(), d.DB()).GetByCharacterId(characterId)
			if err != nil {
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			res, err := model.Map(Transform)(model.FixedProvider(m))()
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

func handleAddTeleportRockMap(worldIdOf WorldIdOf) rest.InputHandler[AddMapInputRestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input AddMapInputRestModel) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				vip, ok := listVip(input.List)
				if !ok {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				worldId, err := worldIdOf(d.Logger(), d.Context(), characterId)
				if err != nil {
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				m, err := NewProcessor(d.Logger(), d.Context(), d.DB()).
					Add(uuid.New(), worldId, characterId, _map.Id(input.MapId), vip)
				if err != nil {
					if s := statusForError(err); s != 0 {
						w.WriteHeader(s)
						return
					}
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				writeModel(d, c, w, r, m)
			}
		})
	}
}

func handleRemoveTeleportRockMap(worldIdOf WorldIdOf) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				vip, ok := listVip(mux.Vars(r)["list"])
				if !ok {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				mapId, err := strconv.ParseUint(mux.Vars(r)["mapId"], 10, 32)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				worldId, err := worldIdOf(d.Logger(), d.Context(), characterId)
				if err != nil {
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				m, err := NewProcessor(d.Logger(), d.Context(), d.DB()).
					Remove(uuid.New(), worldId, characterId, _map.Id(mapId), vip)
				if err != nil {
					if s := statusForError(err); s != 0 {
						w.WriteHeader(s)
						return
					}
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				writeModel(d, c, w, r, m)
			}
		})
	}
}

func listVip(list string) (bool, bool) {
	switch list {
	case ListTypeRegular:
		return false, true
	case ListTypeVip:
		return true, true
	default:
		return false, false
	}
}

func writeModel(d *rest.HandlerDependency, c *rest.HandlerContext, w http.ResponseWriter, r *http.Request, m Model) {
	res, err := model.Map(Transform)(model.FixedProvider(m))()
	if err != nil {
		d.Logger().WithError(err).Errorf("Creating REST model.")
		server.WriteErrorResponse(d.Logger())(w)(err)
		return
	}
	query := r.URL.Query()
	queryParams := jsonapi.ParseQueryFields(&query)
	server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
}
