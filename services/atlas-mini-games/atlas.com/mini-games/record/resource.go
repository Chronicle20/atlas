package record

import (
	"atlas-mini-games/rest"
	"net/http"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

const (
	GetGameRecords = "get_game_records"
)

// InitResource wires GET /characters/{characterId}/game-records. The route
// initializer signature intentionally takes only si (no *gorm.DB) per plan
// task 10's interface; handleGetGameRecords reaches the db via defaultDB,
// set once by Migration in main.go's database.Connect wiring.
func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		registerGet := rest.RegisterHandler(l)(si)
		r := router.PathPrefix("/characters/{characterId}/game-records").Subrouter()
		r.HandleFunc("", registerGet(GetGameRecords, handleGetGameRecords)).Methods(http.MethodGet)
	}
}

func handleGetGameRecords(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			t := tenant.MustFromContext(d.Context())

			ms, err := GetByCharacter(defaultDB, t.Id(), characterId)
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
