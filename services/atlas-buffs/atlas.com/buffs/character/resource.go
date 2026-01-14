package character

import (
	"atlas-buffs/buff"
	"atlas-buffs/rest"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"net/http"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		registerGet := rest.RegisterHandler(l)(si)
		r := router.PathPrefix("/characters").Subrouter()
		r.HandleFunc("/{characterId}/buffs", registerGet("get_character_buffs", handleGetBuffs)).Methods(http.MethodGet)
	}
}

func handleGetBuffs(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			cm, err := NewProcessor(d.Logger(), d.Context()).GetById(characterId)
			if errors.Is(err, ErrNotFound) {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			var res []buff.RestModel
			for _, bs := range cm.Buffs() {
				tb, err := model.Map(buff.Transform)(model.FixedProvider(bs))()
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				res = append(res, tb)
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]buff.RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	})
}
