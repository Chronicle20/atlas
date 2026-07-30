package jobavailability

import (
	"atlas-data/rest"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
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
		query := r.URL.Query()
		page, err := paginate.ParseParams(query, paginate.DefaultPageSize, paginate.MaxPageSize)
		if err != nil {
			server.WriteBadRequest(d.Logger(), w, err.Error())
			return
		}

		ms := NewProcessor(d.Logger(), d.Context()).GetAvailable()
		paged := paginate.Slice(ms, page)

		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(paged.Items, paginate.EnvelopeFor(paged), r)
	}
}
