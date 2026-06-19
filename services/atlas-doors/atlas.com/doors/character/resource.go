package character

import (
	"atlas-doors/door"
	"atlas-doors/rest"
	"net/http"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

const (
	getDoorsByOwner = "get_doors_by_owner"
)

// InitResource registers the /characters/{characterId}/doors route, returning
// the live doors owned by the character for the tenant in context.
func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		r := router.PathPrefix("/characters").Subrouter()
		r.HandleFunc("/{characterId}/doors",
			rest.RegisterHandler(l)(si)(getDoorsByOwner, handleGetDoorsByOwner)).Methods(http.MethodGet)
	}
}

func handleGetDoorsByOwner(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId character.Id) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			p := door.NewProcessor(d.Logger(), d.Context())
			ms, err := p.GetByOwner(characterId)
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to retrieve doors for owner [%d].", characterId)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			res, err := model.SliceMap(door.Transform)(model.FixedProvider(ms))()()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			server.MarshalResponse[[]door.RestModel](d.Logger())(w)(c.ServerInformation())(r.URL.Query())(res)
		}
	})
}
