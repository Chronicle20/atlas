package skillavailability

import (
	"atlas-data/rest"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
)

// InitResource registers GET /data/skill-availability -- the tenant
// version's RELEASED skill identities (wire id + name), sourced directly
// from constants.For (no docType/db: this is not WZ document data).
func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		registerGet := rest.RegisterHandler(l)(si)

		r := router.PathPrefix("/data/skill-availability").Subrouter()
		r.HandleFunc("", registerGet("get_skill_availability", handleGetSkillAvailability)).Methods(http.MethodGet)
	}
}

func handleGetSkillAvailability(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
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
