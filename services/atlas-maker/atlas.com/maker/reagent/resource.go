package reagent

import (
	"atlas-maker/rest"
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
)

// itemPath is the single-reagent route. The id is constrained to digits so a
// write to it can be rejected without shadowing the /reagents/seed route the
// seed catalog mounts under the same prefix.
const itemPath = "/reagents/{itemId:[0-9]+}"

// writeMethods are the methods a read-only resource must refuse.
var writeMethods = []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}

func handleMethodNotAllowed(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusMethodNotAllowed)
}

// InitResource registers the reagent routes. They are read-only: reagents are
// tenant reference data, retuned through the seed catalog rather than over the
// wire, so only GET is registered and every other method falls through to
// gorilla/mux's 405.
func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(db)(si)

			router.HandleFunc("/reagents", registerGet("get_all_reagents", handleGetAllReagents)).Methods(http.MethodGet)
			router.HandleFunc(itemPath, registerGet("get_reagent", handleGetReagent)).Methods(http.MethodGet)

			// The write methods are rejected explicitly rather than left to
			// gorilla/mux's implicit method-mismatch handling: that only
			// reports the mismatch of the LAST route it tried, so once the
			// seed routes mount under the same /reagents prefix a write
			// would fall through to a 404 and the read-only contract would
			// go unenforced. The {itemId} pattern is numeric so that
			// /reagents/seed still reaches the seed router.
			for _, m := range writeMethods {
				router.HandleFunc("/reagents", handleMethodNotAllowed).Methods(m)
				router.HandleFunc(itemPath, handleMethodNotAllowed).Methods(m)
			}
		}
	}
}

func handleGetAllReagents(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
		if err != nil {
			server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
			return
		}

		paged, err := NewProcessor(d.Logger(), d.Context(), d.DB()).GetAllPaged(page)()
		if err != nil {
			d.Logger().WithError(err).Errorf("Retrieving all reagents.")
			server.WriteErrorResponse(d.Logger())(w)(err)
			return
		}

		res, err := model.SliceMap(Transform)(model.FixedProvider(paged.Items))(model.ParallelMap())()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model.")
			server.WriteErrorResponse(d.Logger())(w)(err)
			return
		}

		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res, paginate.EnvelopeFor(paged), r)
	}
}

func handleGetReagent(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseItemId(d.Logger(), func(itemId item.Id) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			m, err := NewProcessor(d.Logger(), d.Context(), d.DB()).GetByItemId(itemId)
			if err != nil {
				if errors.Is(err, ErrNotFound) {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				d.Logger().WithError(err).Errorf("Retrieving reagent [%d].", itemId)
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			rm, err := Transform(m)
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
		}
	})
}
