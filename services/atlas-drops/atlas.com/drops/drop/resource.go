package drop

import (
	"atlas-drops/rest"
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
		r := router.PathPrefix("/drops/{id}").Subrouter()
		r.HandleFunc("", registerGet("get_drop", handleGetDropById)).Methods(http.MethodGet)
	}
}

func handleGetDropById(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseDropId(d.Logger(), func(dropId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			dr, err := NewProcessor(d.Logger(), d.Context()).GetById(dropId)
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			res, err := model.Map(Transform)(model.FixedProvider(dr))()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	})
}
