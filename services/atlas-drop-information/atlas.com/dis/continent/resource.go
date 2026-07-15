package continent

import (
	"atlas-drops-information/rest"
	"net/http"
	"sort"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(db)(si)
			r := router.PathPrefix("/continents/drops").Subrouter()
			r.HandleFunc("", registerGet("get_continent_drops", handleGetContinents)).Methods(http.MethodGet)
		}
	}
}

// handleGetContinents paginates in-memory: GetAll() is a computed
// aggregation (every monster drop grouped by continentId, built from a Go
// map, so it has no natural order), not a single Where-filtered query, so
// it cannot be pushed down to database.PagedQuery. The list of continents
// is stable-sorted by id before slicing to make paging deterministic.
func handleGetContinents(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
		if err != nil {
			server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
			return
		}

		ms, err := NewProcessor(d.Logger(), d.Context(), d.DB()).GetAll()()
		if err != nil {
			d.Logger().WithError(err).Errorf("Retrieving continent drops.")
			server.WriteErrorResponse(d.Logger())(w)(err)
			return
		}

		sort.SliceStable(ms, func(i, j int) bool {
			return ms[i].Id() < ms[j].Id()
		})

		res, err := model.SliceMap(Transform)(model.FixedProvider(ms))(model.ParallelMap())()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model.")
			server.WriteErrorResponse(d.Logger())(w)(err)
			return
		}

		paged := paginate.Slice(res, page)
		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(paged.Items, paginate.EnvelopeFor(paged), r)
	}
}
