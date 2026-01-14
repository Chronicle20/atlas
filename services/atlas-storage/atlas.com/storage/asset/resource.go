package asset

import (
	"atlas-storage/rest"
	"atlas-storage/stackable"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)

			// Assets nested under storage accounts
			r := router.PathPrefix("/storage/accounts/{accountId}/assets").Subrouter()
			r.HandleFunc("", registerGet("get_assets", handleGetAssetsRequest(db))).Methods(http.MethodGet)
			r.HandleFunc("/{assetId}", registerGet("get_asset", handleGetAssetRequest(db))).Methods(http.MethodGet)
		}
	}
}

func handleGetAssetsRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseAccountId(d.Logger(), func(accountId uint32) http.HandlerFunc {
			return rest.ParseWorldId(d.Logger(), func(worldId byte) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					processor := NewProcessor(d.Logger(), d.Context(), db)

					// Get or create storage using processor
					storageId, err := processor.GetOrCreateStorageId(worldId, accountId)
					if err != nil {
						d.Logger().WithError(err).Errorf("Unable to get or create storage for world %d account %d.", worldId, accountId)
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					// Get all assets for this storage using processor
					assets, err := processor.GetAssetsByStorageId(storageId)
					if err != nil {
						d.Logger().WithError(err).Errorf("Unable to get assets for storage %s.", storageId)
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					// Decorate assets with full reference data
					decoratedAssets, err := processor.DecorateAll(assets)
					if err != nil {
						d.Logger().WithError(err).Errorf("Unable to decorate assets for storage %s.", storageId)
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					// Transform to BaseRestModel with full reference data
					restModels, err := TransformAllToBaseRestModel(decoratedAssets)
					if err != nil {
						d.Logger().WithError(err).Errorf("Unable to transform assets to base rest model for storage %s.", storageId)
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					query := r.URL.Query()
					queryParams := jsonapi.ParseQueryFields(&query)
					server.MarshalResponse[[]BaseRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(restModels)
				}
			})
		})
	}
}

func handleGetAssetRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseAccountId(d.Logger(), func(accountId uint32) http.HandlerFunc {
			return rest.ParseWorldId(d.Logger(), func(worldId byte) http.HandlerFunc {
				return rest.ParseAssetId(d.Logger(), func(assetId uint32) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						processor := NewProcessor(d.Logger(), d.Context(), db)

						// Get asset using processor
						assetModel, err := processor.GetAssetById(assetId)
						if err != nil {
							d.Logger().WithError(err).Debugf("Unable to locate asset %d.", assetId)
							w.WriteHeader(http.StatusNotFound)
							return
						}

						// Transform with stackable data if applicable
						var restModel RestModel
						if assetModel.IsStackable() {
							s, err := stackable.GetByAssetId(d.Logger(), db)(assetId)
							if err != nil {
								d.Logger().WithError(err).Warnf("Unable to get stackable data for asset %d.", assetId)
								restModel = Transform(assetModel)
							} else {
								restModel = TransformWithStackable(assetModel, s.Quantity(), s.OwnerId(), s.Flag())
							}
						} else {
							restModel = Transform(assetModel)
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
