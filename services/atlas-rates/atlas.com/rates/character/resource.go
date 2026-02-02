package character

import (
	"atlas-rates/rate"
	"atlas-rates/rest"
	"net/http"

	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		registerGet := rest.RegisterHandler(l)(si)

		r := router.PathPrefix("/worlds/{worldId}/channels/{channelId}/characters/{characterId}").Subrouter()
		r.HandleFunc("/rates", registerGet("get_character_rates", handleGetRates)).Methods(http.MethodGet)
	}
}

func handleGetRates(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseWorldChannel(d.Logger(), func(worldId byte, channelId byte) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				computed, factors, err := NewProcessor(d.Logger(), d.Context()).GetRates(worldId, channelId, characterId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to get rates for character [%d].", characterId)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				res := rate.Transform(characterId, computed, factors)

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[rate.RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
			}
		})
	})
}
