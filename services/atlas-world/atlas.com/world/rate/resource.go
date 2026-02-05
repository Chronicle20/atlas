package rate

import (
	"atlas-world/rest"
	"fmt"
	"net/http"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		registerGet := rest.RegisterHandler(l)(si)
		registerInput := rest.RegisterInputHandler[RestModel](l)(si)

		r := router.PathPrefix("/worlds/{worldId}/rates").Subrouter()
		r.HandleFunc("", registerGet("get_world_rates", handleGetWorldRates)).Methods(http.MethodGet)
		r.HandleFunc("", registerInput("update_world_rate", handleUpdateWorldRate)).Methods(http.MethodPatch)
	}
}

func handleGetWorldRates(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			rates := NewProcessor(d.Logger(), d.Context()).GetWorldRates(worldId)
			rm, err := model.Map(Transform)(model.FixedProvider(rates))()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating rate REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			rm.Id = fmt.Sprintf("world-%d", worldId)

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
		}
	})
}

func handleUpdateWorldRate(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			rateType := Type(input.RateType)
			if !isValidRateType(rateType) {
				d.Logger().Errorf("Invalid rate type: %s", input.RateType)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			err := NewProcessor(d.Logger(), d.Context()).UpdateWorldRate(worldId, rateType, input.Multiplier)
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to update world rate.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}
	})
}

func isValidRateType(t Type) bool {
	switch t {
	case TypeExp, TypeMeso, TypeItemDrop, TypeQuestExp:
		return true
	default:
		return false
	}
}
