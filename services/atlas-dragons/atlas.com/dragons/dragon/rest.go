package dragon

import (
	"atlas-dragons/rest"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

const getDragon = "get_dragon"

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		r := router.PathPrefix("/dragons").Subrouter()
		r.HandleFunc("/{characterId}", rest.RegisterHandler(l)(si)(getDragon, handleGetDragonByCharacterId)).Methods(http.MethodGet)
	}
}

// handleGetDragonByCharacterId returns 404 for a character with no dragon.
// THAT IS THE NORMAL ANSWER for the overwhelming majority of characters —
// every non-Evan in the game. Consumers must treat requests.ErrNotFound as
// "no dragon" and continue; a consumer that logs it as a fetch failure emits
// one error line per non-Evan character.
func handleGetDragonByCharacterId(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			p := NewProcessor(d.Logger(), d.Context())
			m, err := p.GetByCharacterId(characterId)
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
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
}
