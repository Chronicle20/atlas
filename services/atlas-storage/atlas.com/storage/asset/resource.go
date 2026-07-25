package asset

import (
	"atlas-storage/rest"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)

			r := router.PathPrefix("/storage/accounts/{accountId}/assets").Subrouter()
			r.HandleFunc("", registerGet("get_assets", handleGetAssetsRequest(db))).Methods(http.MethodGet)
			r.HandleFunc("/{assetId}", registerGet("get_asset", handleGetAssetRequest(db))).Methods(http.MethodGet)
		}
	}
}

func handleGetAssetsRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseAccountId(d.Logger(), func(accountId uint32) http.HandlerFunc {
			return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					page, err := paginate.ParseParams(r.URL.Query(), paginate.MaxPageSize, paginate.MaxPageSize)
					if err != nil {
						server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
						return
					}

					processor := NewProcessor(d.Logger(), d.Context(), db)

					storageId, err := processor.GetOrCreateStorageId(worldId, accountId)
					if err != nil {
						d.Logger().WithError(err).Errorf("Unable to get or create storage for world %d account %d.", worldId, accountId)
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					// GetAssetsByStorageId materializes every asset in the storage,
					// ordered by inventory_type/template_id/id, and assigns each a
					// dynamic slot number from its position in that global order
					// (see MakeWithDynamicSlot). That slot assignment is only
					// correct over the FULL ordered list, so pagination is applied
					// afterward via paginate.Slice rather than pushing OFFSET/LIMIT
					// into the DB query -- an OFFSET-based fetch would restart the
					// slot index at 0 on every page, producing duplicate/wrong slot
					// numbers on page 2+ (task-117 hidden-decoration hazard).
					assets, err := processor.GetAssetsByStorageId(storageId)
					if err != nil {
						d.Logger().WithError(err).Errorf("Unable to get assets for storage %s.", storageId)
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					paged := paginate.Slice(assets, page)

					restModels, err := TransformAll(paged.Items)
					if err != nil {
						d.Logger().WithError(err).Errorf("Unable to transform assets for storage %s.", storageId)
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					query := r.URL.Query()
					queryParams := jsonapi.ParseQueryFields(&query)
					server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(restModels, paginate.EnvelopeFor(paged), r)
				}
			})
		})
	}
}

func handleGetAssetRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseAccountId(d.Logger(), func(accountId uint32) http.HandlerFunc {
			return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
				return rest.ParseAssetId(d.Logger(), func(assetId uint32) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						processor := NewProcessor(d.Logger(), d.Context(), db)

						assetModel, err := processor.GetAssetById(assetId)
						if err != nil {
							d.Logger().WithError(err).Debugf("Unable to locate asset %d.", assetId)
							w.WriteHeader(http.StatusNotFound)
							return
						}

						restModel, err := Transform(assetModel)
						if err != nil {
							d.Logger().WithError(err).Errorf("Unable to transform asset %d.", assetId)
							server.WriteErrorResponse(d.Logger())(w)(err)
							return
						}

						query := r.URL.Query()
						queryParams := jsonapi.ParseQueryFields(&query)
						server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(restModel)
					}
				})
			})
		})
	}
}
