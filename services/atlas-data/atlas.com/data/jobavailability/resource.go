package jobavailability

import (
	"atlas-data/rest"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

// InitResource registers GET /data/job-availability -- the tenant version's
// RELEASED job identities (wire id + name), sourced directly from
// constants.For (no docType/db: this is not WZ document data).
func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		registerGet := rest.RegisterHandler(l)(si)

		r := router.PathPrefix("/data/job-availability").Subrouter()
		r.HandleFunc("", registerGet("get_job_availability", handleGetJobAvailability)).Methods(http.MethodGet)
	}
}

func handleGetJobAvailability(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ms := NewProcessor(d.Logger(), d.Context()).GetAvailable()
		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(ms)
	}
}
