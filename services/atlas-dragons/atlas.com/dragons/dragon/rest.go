package dragon

import (
	"atlas-dragons/rest"
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"
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
// every non-Evan in the game. Consumers must treat atlasredis.ErrNotFound as
// "no dragon" and continue. A genuine infrastructure failure (e.g. Redis
// unreachable) is a different case entirely: it must NOT collapse into the
// same 404, or an outage on this endpoint produces zero signal. It is logged
// and returned as 500.
func handleGetDragonByCharacterId(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			p := NewProcessor(d.Logger(), d.Context())
			m, err := p.GetByCharacterId(characterId)
			if err != nil {
				if errors.Is(err, atlasredis.ErrNotFound) {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				d.Logger().WithError(err).Errorf("Retrieving dragon for character [%d].", characterId)
				w.WriteHeader(http.StatusInternalServerError)
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
