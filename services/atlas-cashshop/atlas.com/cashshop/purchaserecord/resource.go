package purchaserecord

import (
	"atlas-cashshop/rest"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)
			r := router.PathPrefix("/accounts/{accountId}/purchaseRecords").Subrouter()
			r.HandleFunc("/{serialNumber}", registerGet("get_purchase_record", handleGetPurchaseRecord(db))).Methods(http.MethodGet)
		}
	}
}

// handleGetPurchaseRecord answers whether an account has ever bought a given
// serial number, and how many times. A miss is 200 with Purchased=false --
// never 404 -- the client needs a definitive answer, not an error.
func handleGetPurchaseRecord(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseAccountId(d.Logger(), func(accountId uint32) http.HandlerFunc {
			return rest.ParseSerialNumber(d.Logger(), func(serialNumber uint32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					count, err := NewProcessor(d.Logger(), d.Context(), db).Get(accountId, serialNumber)
					if err != nil {
						d.Logger().WithError(err).Errorf("Unable to retrieve purchase record for account [%d] serial [%d].", accountId, serialNumber)
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					rm := RestModel{
						SerialNumber: serialNumber,
						Purchased:    count > 0,
						Count:        count,
					}

					query := r.URL.Query()
					queryParams := jsonapi.ParseQueryFields(&query)
					server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
				}
			})
		})
	}
}
