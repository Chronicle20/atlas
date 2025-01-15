package character

import (
	"atlas-invites/invite"
	"atlas-invites/rest"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/manyminds/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"net/http"
)

const (
	GetCharacterInvites = "get_character_invites"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		registerGet := rest.RegisterHandler(l)(si)
		r := router.PathPrefix("/characters").Subrouter()
		r.HandleFunc("/{characterId}/invites", registerGet(GetCharacterInvites, handleGetCharacterInvites)).Methods(http.MethodGet)
	}
}

func handleGetCharacterInvites(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			is, err := invite.GetByCharacterId(d.Logger())(d.Context())(characterId)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			res, err := model.SliceMap(invite.Transform)(model.FixedProvider(is))()()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			server.Marshal[[]invite.RestModel](d.Logger())(w)(c.ServerInformation())(res)
		}
	})
}
