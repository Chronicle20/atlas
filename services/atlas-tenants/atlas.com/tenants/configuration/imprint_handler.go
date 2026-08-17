package configuration

import (
	"atlas-tenants/kafka/message"
	"atlas-tenants/rest"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

// This file is the FR-2.6 tenant-configurable pending-change expiry
// resource, `imprint-configs`. It is modelled on the trade-configs
// implementation spread across processor.go/resource.go/rest.go/kafka.go/
// provider.go/seed.go, but kept self-contained in one file because
// imprint-configs carries exactly one knob (pendingExpiryHours) rather than
// trade-configs' tax-tier table — there is no partial-PATCH merge concern to
// replicate. atlas-character (services/atlas-character/atlas.com/character/
// configuration) is the consumer: it GETs the single per-tenant document and
// falls back to its own 168h default on a fetch miss (FR-9.2-style
// resilience), so an unseeded tenant keeps working.

// CreateImprintConfig creates a new imprint config configuration
func (p *ProcessorImpl) CreateImprintConfig(mb *message.Buffer) func(tenantId uuid.UUID) func(config map[string]interface{}) (Model, error) {
	return func(tenantId uuid.UUID) func(config map[string]interface{}) (Model, error) {
		return func(config map[string]interface{}) (Model, error) {
			existingProvider := GetByTenantIdAndResourceNameProvider(tenantId, "imprint-configs")(p.db)
			existing, err := existingProvider()

			var resourceData json.RawMessage

			configID := ""
			if id, ok := config["id"].(string); ok {
				configID = id
			}

			if err == nil {
				var existingData map[string]interface{}
				if err := json.Unmarshal(existing.ResourceData, &existingData); err != nil {
					return Model{}, err
				}

				if resources, ok := existingData["data"].([]interface{}); ok {
					resources = append(resources, config)
					existingData["data"] = resources
					resourceData, err = json.Marshal(existingData)
					if err != nil {
						return Model{}, err
					}
				} else {
					resourceData, err = CreateImprintConfigJsonData([]map[string]interface{}{config})
					if err != nil {
						return Model{}, err
					}
				}

				existing.ResourceData = resourceData
				if err := UpdateConfiguration(p.db, existing); err != nil {
					return Model{}, err
				}

				m, err := Make(existing)
				if err != nil {
					return Model{}, err
				}

				if err := mb.Put(EventTopicConfigurationStatus, CreateImprintConfigStatusEventProvider(tenantId, EventTypeImprintConfigCreated, configID)); err != nil {
					return Model{}, err
				}

				return m, nil
			} else if errors.Is(err, gorm.ErrRecordNotFound) {
				resourceData, err = CreateSingleImprintConfigJsonData(config)
				if err != nil {
					return Model{}, err
				}

				entity := Entity{
					ID:           uuid.New(),
					TenantId:     tenantId,
					ResourceName: "imprint-configs",
					ResourceData: resourceData,
				}

				if err := CreateConfiguration(p.db, entity); err != nil {
					return Model{}, err
				}

				m, err := Make(entity)
				if err != nil {
					return Model{}, err
				}

				if err := mb.Put(EventTopicConfigurationStatus, CreateImprintConfigStatusEventProvider(tenantId, EventTypeImprintConfigCreated, configID)); err != nil {
					return Model{}, err
				}

				return m, nil
			} else {
				return Model{}, err
			}
		}
	}
}

// CreateImprintConfigAndEmit creates a new imprint config configuration and emits events
func (p *ProcessorImpl) CreateImprintConfigAndEmit(tenantId uuid.UUID, config map[string]interface{}) (Model, error) {
	ctx, err := p.tenantCtx(tenantId)
	if err != nil {
		return Model{}, err
	}
	var result Model
	txErr := database.ExecuteTransaction(p.db.WithContext(ctx), func(tx *gorm.DB) error {
		var err error
		result, err = message.EmitWithResult[Model, uuid.UUID](outbox.EmitProvider(p.l, ctx, tx))(func(mb *message.Buffer) func(uuid.UUID) (Model, error) {
			return func(tenantId uuid.UUID) (Model, error) {
				return NewProcessor(p.l, ctx, tx).CreateImprintConfig(mb)(tenantId)(config)
			}
		})(tenantId)
		return err
	})
	return result, txErr
}

// UpdateImprintConfig updates an existing imprint config configuration
func (p *ProcessorImpl) UpdateImprintConfig(mb *message.Buffer) func(tenantId uuid.UUID) func(configID string) func(config map[string]interface{}) (Model, error) {
	return func(tenantId uuid.UUID) func(configID string) func(config map[string]interface{}) (Model, error) {
		return func(configID string) func(config map[string]interface{}) (Model, error) {
			return func(config map[string]interface{}) (Model, error) {
				existingProvider := GetByTenantIdAndResourceNameProvider(tenantId, "imprint-configs")(p.db)
				existing, err := existingProvider()
				if err != nil {
					return Model{}, err
				}

				var existingData map[string]interface{}
				if err := json.Unmarshal(existing.ResourceData, &existingData); err != nil {
					return Model{}, err
				}

				config["id"] = configID

				// Unlike trade-configs there is only one attribute, so an
				// incoming entry simply replaces the stored one wholesale —
				// there is no other knob a plain overwrite could wipe.
				if resources, ok := existingData["data"].([]interface{}); ok {
					found := false
					for i, resource := range resources {
						if resourceMap, ok := resource.(map[string]interface{}); ok {
							if id, ok := resourceMap["id"].(string); ok && id == configID {
								resources[i] = config
								found = true
								break
							}
						}
					}

					if !found {
						return Model{}, errors.New("imprint config not found")
					}

					existingData["data"] = resources
				} else if data, ok := existingData["data"].(map[string]interface{}); ok {
					if id, ok := data["id"].(string); ok && id == configID {
						existingData["data"] = config
					} else {
						return Model{}, errors.New("imprint config not found")
					}
				} else {
					return Model{}, errors.New("invalid resource data format")
				}

				resourceData, err := json.Marshal(existingData)
				if err != nil {
					return Model{}, err
				}

				existing.ResourceData = resourceData
				if err := UpdateConfiguration(p.db, existing); err != nil {
					return Model{}, err
				}

				m, err := Make(existing)
				if err != nil {
					return Model{}, err
				}

				if err := mb.Put(EventTopicConfigurationStatus, CreateImprintConfigStatusEventProvider(tenantId, EventTypeImprintConfigUpdated, configID)); err != nil {
					return Model{}, err
				}

				return m, nil
			}
		}
	}
}

// UpdateImprintConfigAndEmit updates an existing imprint config configuration and emits events
func (p *ProcessorImpl) UpdateImprintConfigAndEmit(tenantId uuid.UUID, configID string, config map[string]interface{}) (Model, error) {
	ctx, err := p.tenantCtx(tenantId)
	if err != nil {
		return Model{}, err
	}
	var result Model
	txErr := database.ExecuteTransaction(p.db.WithContext(ctx), func(tx *gorm.DB) error {
		var err error
		result, err = message.EmitWithResult[Model, uuid.UUID](outbox.EmitProvider(p.l, ctx, tx))(func(mb *message.Buffer) func(uuid.UUID) (Model, error) {
			return func(tenantId uuid.UUID) (Model, error) {
				return NewProcessor(p.l, ctx, tx).UpdateImprintConfig(mb)(tenantId)(configID)(config)
			}
		})(tenantId)
		return err
	})
	return result, txErr
}

// DeleteImprintConfig deletes an imprint config configuration
func (p *ProcessorImpl) DeleteImprintConfig(mb *message.Buffer) func(tenantId uuid.UUID) func(configID string) error {
	return func(tenantId uuid.UUID) func(configID string) error {
		return func(configID string) error {
			if err := DeleteConfiguration(p.db, tenantId, "imprint-configs", configID); err != nil {
				return err
			}

			if err := mb.Put(EventTopicConfigurationStatus, CreateImprintConfigStatusEventProvider(tenantId, EventTypeImprintConfigDeleted, configID)); err != nil {
				return err
			}

			return nil
		}
	}
}

// DeleteImprintConfigAndEmit deletes an imprint config configuration and emits events
func (p *ProcessorImpl) DeleteImprintConfigAndEmit(tenantId uuid.UUID, configID string) error {
	ctx, err := p.tenantCtx(tenantId)
	if err != nil {
		return err
	}
	return database.ExecuteTransaction(p.db.WithContext(ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, ctx, tx))(func(mb *message.Buffer) error {
			return NewProcessor(p.l, ctx, tx).DeleteImprintConfig(mb)(tenantId)(configID)
		})
	})
}

// GetImprintConfigById gets an imprint config by ID
func (p *ProcessorImpl) GetImprintConfigById(tenantId uuid.UUID, configID string) (map[string]interface{}, error) {
	return p.ImprintConfigByIdProvider(tenantId, configID)()
}

// GetAllImprintConfigs gets all imprint configs for a tenant
func (p *ProcessorImpl) GetAllImprintConfigs(tenantId uuid.UUID) ([]map[string]interface{}, error) {
	return p.AllImprintConfigsProvider(tenantId)()
}

// ImprintConfigByIdProvider returns a provider for an imprint config by ID
func (p *ProcessorImpl) ImprintConfigByIdProvider(tenantId uuid.UUID, configID string) model.Provider[map[string]interface{}] {
	return GetImprintConfigByIdProvider(tenantId, configID)(p.db)
}

// AllImprintConfigsProvider returns a provider for all imprint configs for a tenant
func (p *ProcessorImpl) AllImprintConfigsProvider(tenantId uuid.UUID) model.Provider[[]map[string]interface{}] {
	return GetAllImprintConfigsProvider(tenantId)(p.db)
}

// SeedImprintConfigs clears existing imprint configs for a tenant and loads them from seed files
func (p *ProcessorImpl) SeedImprintConfigs(tenantId uuid.UUID) (SeedResult, error) {
	p.l.Infof("Seeding imprint-configs for tenant [%s]", tenantId)

	result := SeedResult{}

	deletedCount, err := DeleteConfigurationByResourceName(p.db, tenantId, "imprint-configs")
	if err != nil {
		return result, fmt.Errorf("failed to clear existing imprint-configs: %w", err)
	}
	result.DeletedCount = int(deletedCount)

	configs, loadErrors := LoadImprintConfigFiles()
	for _, err := range loadErrors {
		result.Errors = append(result.Errors, err.Error())
		result.FailedCount++
	}

	for _, config := range configs {
		id, _ := config["id"].(string)
		_, err := p.CreateImprintConfigAndEmit(tenantId, config)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: failed to create: %v", id, err))
			result.FailedCount++
			continue
		}
		result.CreatedCount++
	}

	p.l.Infof("Imprint config seed complete for tenant [%s]: deleted=%d, created=%d, failed=%d",
		tenantId, result.DeletedCount, result.CreatedCount, result.FailedCount)

	return result, nil
}

// GetImprintConfigHandler handles GET /tenants/{tenantId}/configurations/imprint-configs
// and returns the single per-tenant imprint (pending-change expiry)
// configuration. atlas-character decodes this as a single JSON:API object
// and falls back to its shipped 168h default on the 404 below.
func GetImprintConfigHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				processor := NewProcessor(d.Logger(), d.Context(), db)

				configs, err := processor.GetAllImprintConfigs(tenantId)
				if err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						d.Logger().Info("No imprint config found for tenant")
						w.WriteHeader(http.StatusNotFound)
						return
					}
					d.Logger().WithError(err).Error("Failed to get imprint config")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				if len(configs) == 0 {
					d.Logger().Info("No imprint config found for tenant")
					w.WriteHeader(http.StatusNotFound)
					return
				}

				rm, err := TransformImprintConfig(configs[0])
				if err != nil {
					d.Logger().WithError(err).Error("Failed to transform imprint config")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[ImprintConfigRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
			}
		})
	}
}

// GetImprintConfigByIdHandler handles GET /tenants/{tenantId}/configurations/imprint-configs/{imprintConfigId}
func GetImprintConfigByIdHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return rest.ParseImprintConfigId(d.Logger(), func(imprintConfigId string) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					processor := NewProcessor(d.Logger(), d.Context(), db)

					config, err := processor.GetImprintConfigById(tenantId, imprintConfigId)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to get imprint config")
						w.WriteHeader(http.StatusNotFound)
						return
					}

					rm, err := TransformImprintConfig(config)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to transform imprint config")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					query := r.URL.Query()
					queryParams := jsonapi.ParseQueryFields(&query)
					server.MarshalResponse[ImprintConfigRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
				}
			})
		})
	}
}

// CreateImprintConfigHandler handles POST /tenants/{tenantId}/configurations/imprint-configs
func CreateImprintConfigHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext, model ImprintConfigRestModel) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, m ImprintConfigRestModel) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				config, err := ExtractImprintConfig(m)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to extract imprint config data")
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				processor := NewProcessor(d.Logger(), d.Context(), db)
				_, err = processor.CreateImprintConfigAndEmit(tenantId, config)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to create imprint config")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				configId, _ := config["id"].(string)

				createdConfig, err := processor.GetImprintConfigById(tenantId, configId)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to get created imprint config")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				rm, err := TransformImprintConfig(createdConfig)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to transform imprint config")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				w.WriteHeader(http.StatusCreated)
				server.MarshalResponse[ImprintConfigRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
			}
		})
	}
}

// UpdateImprintConfigHandler handles PATCH /tenants/{tenantId}/configurations/imprint-configs/{imprintConfigId}
func UpdateImprintConfigHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext, model ImprintConfigRestModel) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, m ImprintConfigRestModel) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return rest.ParseImprintConfigId(d.Logger(), func(imprintConfigId string) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					config, err := ExtractImprintConfig(m)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to extract imprint config data")
						w.WriteHeader(http.StatusBadRequest)
						return
					}

					processor := NewProcessor(d.Logger(), d.Context(), db)
					_, err = processor.UpdateImprintConfigAndEmit(tenantId, imprintConfigId, config)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to update imprint config")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					updatedConfig, err := processor.GetImprintConfigById(tenantId, imprintConfigId)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to get updated imprint config")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					rm, err := TransformImprintConfig(updatedConfig)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to transform imprint config")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					query := r.URL.Query()
					queryParams := jsonapi.ParseQueryFields(&query)
					server.MarshalResponse[ImprintConfigRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
				}
			})
		})
	}
}

// DeleteImprintConfigHandler handles DELETE /tenants/{tenantId}/configurations/imprint-configs/{imprintConfigId}
func DeleteImprintConfigHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return rest.ParseImprintConfigId(d.Logger(), func(imprintConfigId string) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					processor := NewProcessor(d.Logger(), d.Context(), db)
					err := processor.DeleteImprintConfigAndEmit(tenantId, imprintConfigId)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to delete imprint config")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					w.WriteHeader(http.StatusNoContent)
				}
			})
		})
	}
}

// SeedImprintConfigsHandler handles POST /tenants/{tenantId}/configurations/imprint-configs/seed
func SeedImprintConfigsHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				processor := NewProcessor(d.Logger(), d.Context(), db)
				result, err := processor.SeedImprintConfigs(tenantId)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to seed imprint configs")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(result)
			}
		})
	}
}

// RegisterImprintConfigRoutes registers the imprint-configs routes. Called
// from RegisterRoutes in resource.go, alongside the trade-configs block —
// "/seed" is registered BEFORE the "{imprintConfigId}" pattern so it is not
// shadowed by a config whose id happens to be "seed" (see the routes-endpoint
// comment in resource.go for why).
func RegisterImprintConfigRoutes(db *gorm.DB, si jsonapi.ServerInformation, l logrus.FieldLogger, r *mux.Router) {
	registerHandler := rest.RegisterHandler(l)(si)
	registerImprintConfigInputHandler := rest.RegisterInputHandler[ImprintConfigRestModel](l)(si)

	r.HandleFunc("/tenants/{tenantId}/configurations/imprint-configs/seed", registerHandler("seed_imprint_configs", SeedImprintConfigsHandler(db))).Methods(http.MethodPost)
	r.HandleFunc("/tenants/{tenantId}/configurations/imprint-configs", registerHandler("get_imprint_config", GetImprintConfigHandler(db))).Methods(http.MethodGet)
	r.HandleFunc("/tenants/{tenantId}/configurations/imprint-configs/{imprintConfigId}", registerHandler("get_imprint_config_by_id", GetImprintConfigByIdHandler(db))).Methods(http.MethodGet)
	r.HandleFunc("/tenants/{tenantId}/configurations/imprint-configs", registerImprintConfigInputHandler("create_imprint_config", CreateImprintConfigHandler(db))).Methods(http.MethodPost)
	r.HandleFunc("/tenants/{tenantId}/configurations/imprint-configs/{imprintConfigId}", registerImprintConfigInputHandler("update_imprint_config", UpdateImprintConfigHandler(db))).Methods(http.MethodPatch)
	r.HandleFunc("/tenants/{tenantId}/configurations/imprint-configs/{imprintConfigId}", registerHandler("delete_imprint_config", DeleteImprintConfigHandler(db))).Methods(http.MethodDelete)
}
