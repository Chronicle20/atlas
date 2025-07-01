package validation

import (
	"atlas-query-aggregator/rest"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"net/http"
)

// InitResource registers the routes with the router
func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(r *mux.Router, l logrus.FieldLogger) {
		r.HandleFunc("/validations", rest.RegisterInputHandler[RestModel](l)(si)("handle_validations", validationHandler)).Methods(http.MethodPost)
	}
}

func validationHandler(d *rest.HandlerDependency, c *rest.HandlerContext, im RestModel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Validate the conditions
		result, err := NewProcessor(d.Logger(), d.Context()).Validate()(im.CharacterId, im.Conditions)
		if err != nil {
			d.Logger().WithError(err).Errorln("Failed to validate conditions")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		rms, err := model.Map(Transform)(model.FixedProvider(result))()
		if err != nil {
			d.Logger().WithError(err).Error("Failed to retrieve sagas")
			w.WriteHeader(http.StatusInternalServerError)
		}

		// Marshal response
		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rms)
	}
}
