package record

import (
	"atlas-mini-games/rest"
	"net/http"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
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
				ms, err := NewProcessor(d.Logger(), d.Context(), db).GetByCharacter(characterId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to retrieve game records for character [%d].", characterId)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				res, err := model.SliceMap(Transform)(model.FixedProvider(ms))()()
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
}
