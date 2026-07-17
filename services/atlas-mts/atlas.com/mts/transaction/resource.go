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

func handleGetCharacterTransactions(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			p := NewProcessor(d.Logger(), d.Context(), d.DB())

			ms, err := p.GetByCharacter(characterId)
			if err != nil {
				d.Logger().WithError(err).Errorf("Retrieving transactions for character [%d].", characterId)
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			res, err := model.SliceMap(Transform)(model.FixedProvider(ms))(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	})
}
