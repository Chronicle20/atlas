package asset

import (
	"atlas-storage/rest"
	"atlas-storage/stackable"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
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
					t := tenant.MustFromContext(d.Context())

					// Get or create storage using provider
					storageId, err := getOrCreateStorageId(d.Logger(), db, t.Id(), worldId, accountId)
					if err != nil {
						d.Logger().WithError(err).Errorf("Unable to get or create storage for world %d account %d.", worldId, accountId)
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					// Get all assets for this storage using provider
					assets, err := GetByStorageId(d.Logger(), db, t.Id())(storageId)
					if err != nil {
						d.Logger().WithError(err).Errorf("Unable to get assets for storage %s.", storageId)
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					// Get stackable data for stackable items
					var stackableAssetIds []uint32
					for _, a := range assets {
						if a.IsStackable() {
							stackableAssetIds = append(stackableAssetIds, a.Id())
						}
					}

					stackableMap := make(map[uint32]stackable.Model)
					if len(stackableAssetIds) > 0 {
						stackables, err := stackable.GetByAssetIds(d.Logger(), db)(stackableAssetIds)
						if err != nil {
							d.Logger().WithError(err).Warnf("Unable to get stackable data for assets.")
						} else {
							for _, s := range stackables {
								stackableMap[s.AssetId()] = s
							}
						}
					}

					// Transform assets with stackable data
					restModels := make([]RestModel, 0, len(assets))
					for _, a := range assets {
						if s, ok := stackableMap[a.Id()]; ok {
							restModels = append(restModels, TransformWithStackable(a, s.Quantity(), s.OwnerId(), s.Flag()))
						} else {
							restModels = append(restModels, Transform(a))
						}
					}

					query := r.URL.Query()
					queryParams := jsonapi.ParseQueryFields(&query)
					server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(restModels)
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
						t := tenant.MustFromContext(d.Context())

						// Get asset using provider
						assetModel, err := GetById(d.Logger(), db, t.Id())(assetId)
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

// getOrCreateStorageId retrieves or creates storage ID
// NOTE: This uses raw DB queries to avoid circular package dependency with storage package
func getOrCreateStorageId(l logrus.FieldLogger, db *gorm.DB, tenantId uuid.UUID, worldId byte, accountId uint32) (uuid.UUID, error) {
	// Try to get existing storage
	var storageId uuid.UUID
	err := db.Table("storages").
		Select("id").
		Where("tenant_id = ? AND world_id = ? AND account_id = ?", tenantId, worldId, accountId).
		Scan(&storageId).Error

	if err == nil && storageId != uuid.Nil {
		return storageId, nil
	}

	// Storage not found, create it
	if err == gorm.ErrRecordNotFound || storageId == uuid.Nil {
		newId := uuid.New()
		createErr := db.Table("storages").Create(map[string]interface{}{
			"tenant_id":  tenantId,
			"id":         newId,
			"world_id":   worldId,
			"account_id": accountId,
			"capacity":   4,
			"mesos":      0,
		}).Error
		if createErr != nil {
			return uuid.Nil, createErr
		}
		return newId, nil
	}

	return uuid.Nil, err
}
