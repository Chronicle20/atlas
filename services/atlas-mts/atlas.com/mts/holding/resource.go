package holding

import (
	"atlas-mts/rest"
	"net/http"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// InitResource registers the read-only holding route:
//   - GET /characters/{characterId}/mts/holding — a character's take-home holdings
//
// An optional ?worldId= query param narrows the result to a single world; absent,
// all of the character's holdings are returned. The take-home POST route is added
// in Phase 4 (it initiates the WithdrawFromMts saga).
func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(db)(si)

			r := router.PathPrefix("/characters/{characterId}/mts/holding").Subrouter()
			r.HandleFunc("", registerGet("get_character_holdings", handleGetCharacterHoldings)).Methods(http.MethodGet)
		}
	}
}

func handleGetCharacterHoldings(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			p := NewProcessor(d.Logger(), d.Context(), d.DB())

			var ms []Model
			var err error
			if v := r.URL.Query().Get("worldId"); v != "" {
				worldId, perr := strconv.ParseUint(v, 10, 8)
				if perr != nil {
					d.Logger().WithError(perr).Errorf("Unable to parse worldId query for character [%d].", characterId)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				ms, err = p.GetByOwner(world.Id(byte(worldId)), characterId)
			} else {
				ms, err = p.GetByCharacter(characterId)
			}
			if err != nil {
				d.Logger().WithError(err).Errorf("Retrieving holdings for character [%d].", characterId)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			res, err := model.SliceMap(Transform)(model.FixedProvider(ms))(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	})
}
