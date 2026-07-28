package transaction

import (
	"atlas-mts/rest"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
)

// InitResource registers the transaction-history read route:
//   - GET /characters/{characterId}/mts/transactions — a character's settled
//     purchase/sale history (My Page -> History), newest-first.
//
// Transaction rows are written server-side at settle, so this surface is
// read-only — there is no create/delete route.
func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(db)(si)

			r := router.PathPrefix("/characters/{characterId}/mts/transactions").Subrouter()
			r.HandleFunc("", registerGet("get_character_transactions", handleGetCharacterTransactions)).Methods(http.MethodGet)
		}
	}
}

// handleGetCharacterTransactions is a growing log (settled purchase/sale
// history accumulates over a character's lifetime) — page[size] defaults to
// paginate.DefaultPageSize, capped at paginate.MaxPageSize (task-117).
func handleGetCharacterTransactions(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			page, perr := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
			if perr != nil {
				server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
				return
			}

			p := NewProcessor(d.Logger(), d.Context(), d.DB())

			paged, err := p.ByCharacterPagedProvider(characterId, page)()
			if err != nil {
				d.Logger().WithError(err).Errorf("Retrieving transactions for character [%d].", characterId)
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
	})
}
