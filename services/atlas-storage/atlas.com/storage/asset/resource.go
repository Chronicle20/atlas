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

					// Decorate assets with full reference data
					processor := NewProcessor(d.Logger(), d.Context(), db)
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

// StorageEntity is a minimal storage entity to avoid circular dependency with storage package
type StorageEntity struct {
	TenantId  uuid.UUID `gorm:"not null;uniqueIndex:idx_tenant_world_account"`
	Id        uuid.UUID `gorm:"primaryKey;type:uuid"`
	WorldId   byte      `gorm:"not null;uniqueIndex:idx_tenant_world_account"`
	AccountId uint32    `gorm:"not null;uniqueIndex:idx_tenant_world_account"`
	Capacity  uint32    `gorm:"not null;default:4"`
	Mesos     uint32    `gorm:"not null;default:0"`
}

func (StorageEntity) TableName() string {
	return "storages"
}

// getOrCreateStorageId retrieves or creates storage ID
func getOrCreateStorageId(l logrus.FieldLogger, db *gorm.DB, tenantId uuid.UUID, worldId byte, accountId uint32) (uuid.UUID, error) {
	// Try to get existing storage
	var storage StorageEntity
	err := db.Where("tenant_id = ? AND world_id = ? AND account_id = ?", tenantId, worldId, accountId).
		First(&storage).Error

	if err == nil {
		return storage.Id, nil
	}

	// Storage not found, create it
	if err == gorm.ErrRecordNotFound {
		storage = StorageEntity{
			TenantId:  tenantId,
			Id:        uuid.New(),
			WorldId:   worldId,
			AccountId: accountId,
			Capacity:  4,
			Mesos:     0,
		}
		createErr := db.Create(&storage).Error
		if createErr != nil {
			return uuid.Nil, createErr
		}
		return storage.Id, nil
	}

	return uuid.Nil, err
}
