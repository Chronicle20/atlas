package record

import (
	"atlas-mini-games/rest"
	"net/http"
	"sort"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	GetGameRecords = "get_game_records"
)

// InitResource wires GET /characters/{characterId}/game-records. The db
// handle is curried in at wiring time (buddies list.InitResource shape,
// services/atlas-buddies/atlas.com/buddies/list/resource.go:27).
func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)
			r := router.PathPrefix("/characters/{characterId}/game-records").Subrouter()
			r.HandleFunc("", registerGet(GetGameRecords, handleGetGameRecords(db))).Methods(http.MethodGet)
		}
	}
}

func handleGetGameRecords(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				// Group C game-capped list: a character has one record per game
				// type (a handful), so default the page size to the cap and page
				// the materialized slice.
				page, err := paginate.ParseParams(r.URL.Query(), paginate.MaxPageSize, paginate.MaxPageSize)
				if err != nil {
					server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
					return
				}

				ms, err := NewProcessor(d.Logger(), d.Context(), db).GetByCharacter(characterId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to retrieve game records for character [%d].", characterId)
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				// Sort by GameType, the collection's stable unique key (one row
				// per game type per character), so paging is deterministic.
				sorted := make([]Model, len(ms))
				copy(sorted, ms)
				sort.Slice(sorted, func(i, j int) bool { return sorted[i].GameType() < sorted[j].GameType() })
				paged := paginate.Slice(sorted, page)

				res, err := model.SliceMap(Transform)(model.FixedProvider(paged.Items))()()
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
	}
}
