package configuration

import (
	"atlas-tenants/kafka/message"
	tenants "atlas-tenants/tenant"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	atlastenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Processor defines the interface for configuration operations
type Processor interface {
	// Route operations
	// CreateRoute creates a new route configuration
	CreateRoute(mb *message.Buffer) func(tenantId uuid.UUID) func(route map[string]interface{}) (Model, error)
	// CreateRouteAndEmit creates a new route configuration and emits events
	CreateRouteAndEmit(tenantId uuid.UUID, route map[string]interface{}) (Model, error)
	// UpdateRoute updates an existing route configuration
	UpdateRoute(mb *message.Buffer) func(tenantId uuid.UUID) func(routeID string) func(route map[string]interface{}) (Model, error)
	// UpdateRouteAndEmit updates an existing route configuration and emits events
	UpdateRouteAndEmit(tenantId uuid.UUID, routeID string, route map[string]interface{}) (Model, error)
	// DeleteRoute deletes a route configuration
	DeleteRoute(mb *message.Buffer) func(tenantId uuid.UUID) func(routeID string) error
	// DeleteRouteAndEmit deletes a route configuration and emits events
	DeleteRouteAndEmit(tenantId uuid.UUID, routeID string) error
	// GetRouteById gets a route by ID
	GetRouteById(tenantId uuid.UUID, routeID string) (map[string]interface{}, error)
	// GetAllRoutes gets all routes for a tenant
	GetAllRoutes(tenantId uuid.UUID) ([]map[string]interface{}, error)
	// RouteByIdProvider returns a provider for a route by ID
	RouteByIdProvider(tenantId uuid.UUID, routeID string) model.Provider[map[string]interface{}]
	// AllRoutesProvider returns a provider for all routes for a tenant
	AllRoutesProvider(tenantId uuid.UUID) model.Provider[[]map[string]interface{}]

	// Instance route operations
	// CreateInstanceRoute creates a new instance route configuration
	CreateInstanceRoute(mb *message.Buffer) func(tenantId uuid.UUID) func(route map[string]interface{}) (Model, error)
	// CreateInstanceRouteAndEmit creates a new instance route configuration and emits events
	CreateInstanceRouteAndEmit(tenantId uuid.UUID, route map[string]interface{}) (Model, error)
	// UpdateInstanceRoute updates an existing instance route configuration
	UpdateInstanceRoute(mb *message.Buffer) func(tenantId uuid.UUID) func(routeID string) func(route map[string]interface{}) (Model, error)
	// UpdateInstanceRouteAndEmit updates an existing instance route configuration and emits events
	UpdateInstanceRouteAndEmit(tenantId uuid.UUID, routeID string, route map[string]interface{}) (Model, error)
	// DeleteInstanceRoute deletes an instance route configuration
	DeleteInstanceRoute(mb *message.Buffer) func(tenantId uuid.UUID) func(routeID string) error
	// DeleteInstanceRouteAndEmit deletes an instance route configuration and emits events
	DeleteInstanceRouteAndEmit(tenantId uuid.UUID, routeID string) error
	// GetInstanceRouteById gets an instance route by ID
	GetInstanceRouteById(tenantId uuid.UUID, routeID string) (map[string]interface{}, error)
	// GetAllInstanceRoutes gets all instance routes for a tenant
	GetAllInstanceRoutes(tenantId uuid.UUID) ([]map[string]interface{}, error)
	// InstanceRouteByIdProvider returns a provider for an instance route by ID
	InstanceRouteByIdProvider(tenantId uuid.UUID, routeID string) model.Provider[map[string]interface{}]
	// AllInstanceRoutesProvider returns a provider for all instance routes for a tenant
	AllInstanceRoutesProvider(tenantId uuid.UUID) model.Provider[[]map[string]interface{}]

	// Vessel operations
	// CreateVessel creates a new vessel configuration
	CreateVessel(mb *message.Buffer) func(tenantId uuid.UUID) func(vessel map[string]interface{}) (Model, error)
	// CreateVesselAndEmit creates a new vessel configuration and emits events
	CreateVesselAndEmit(tenantId uuid.UUID, vessel map[string]interface{}) (Model, error)
	// UpdateVessel updates an existing vessel configuration
	UpdateVessel(mb *message.Buffer) func(tenantId uuid.UUID) func(vesselID string) func(vessel map[string]interface{}) (Model, error)
	// UpdateVesselAndEmit updates an existing vessel configuration and emits events
	UpdateVesselAndEmit(tenantId uuid.UUID, vesselID string, vessel map[string]interface{}) (Model, error)
	// DeleteVessel deletes a vessel configuration
	DeleteVessel(mb *message.Buffer) func(tenantId uuid.UUID) func(vesselID string) error
	// DeleteVesselAndEmit deletes a vessel configuration and emits events
	DeleteVesselAndEmit(tenantId uuid.UUID, vesselID string) error
	// GetVesselById gets a vessel by ID
	GetVesselById(tenantId uuid.UUID, vesselID string) (map[string]interface{}, error)
	// GetAllVessels gets all vessels for a tenant
	GetAllVessels(tenantId uuid.UUID) ([]map[string]interface{}, error)
	// VesselByIdProvider returns a provider for a vessel by ID
	VesselByIdProvider(tenantId uuid.UUID, vesselID string) model.Provider[map[string]interface{}]
	// AllVesselsProvider returns a provider for all vessels for a tenant
	AllVesselsProvider(tenantId uuid.UUID) model.Provider[[]map[string]interface{}]

	// RPS reward operations
	// CreateRpsReward creates a new rps-reward configuration
	CreateRpsReward(mb *message.Buffer) func(tenantId uuid.UUID) func(rpsReward map[string]interface{}) (Model, error)
	// CreateRpsRewardAndEmit creates a new rps-reward configuration and emits events
	CreateRpsRewardAndEmit(tenantId uuid.UUID, rpsReward map[string]interface{}) (Model, error)
	// UpdateRpsReward updates an existing rps-reward configuration
	UpdateRpsReward(mb *message.Buffer) func(tenantId uuid.UUID) func(rpsRewardID string) func(rpsReward map[string]interface{}) (Model, error)
	// UpdateRpsRewardAndEmit updates an existing rps-reward configuration and emits events
	UpdateRpsRewardAndEmit(tenantId uuid.UUID, rpsRewardID string, rpsReward map[string]interface{}) (Model, error)
	// DeleteRpsReward deletes a rps-reward configuration
	DeleteRpsReward(mb *message.Buffer) func(tenantId uuid.UUID) func(rpsRewardID string) error
	// DeleteRpsRewardAndEmit deletes a rps-reward configuration and emits events
	DeleteRpsRewardAndEmit(tenantId uuid.UUID, rpsRewardID string) error
	// GetRpsRewardById gets a rps-reward by ID
	GetRpsRewardById(tenantId uuid.UUID, rpsRewardID string) (map[string]interface{}, error)
	// GetAllRpsRewards gets all rps-rewards for a tenant
	GetAllRpsRewards(tenantId uuid.UUID) ([]map[string]interface{}, error)
	// RpsRewardByIdProvider returns a provider for a rps-reward by ID
	RpsRewardByIdProvider(tenantId uuid.UUID, rpsRewardID string) model.Provider[map[string]interface{}]
	// AllRpsRewardsProvider returns a provider for all rps-rewards for a tenant
	AllRpsRewardsProvider(tenantId uuid.UUID) model.Provider[[]map[string]interface{}]

	// MTS config operations
	// CreateMtsConfig creates a new mts config configuration
	CreateMtsConfig(mb *message.Buffer) func(tenantId uuid.UUID) func(config map[string]interface{}) (Model, error)
	// CreateMtsConfigAndEmit creates a new mts config configuration and emits events
	CreateMtsConfigAndEmit(tenantId uuid.UUID, config map[string]interface{}) (Model, error)
	// UpdateMtsConfig updates an existing mts config configuration
	UpdateMtsConfig(mb *message.Buffer) func(tenantId uuid.UUID) func(configID string) func(config map[string]interface{}) (Model, error)
	// UpdateMtsConfigAndEmit updates an existing mts config configuration and emits events
	UpdateMtsConfigAndEmit(tenantId uuid.UUID, configID string, config map[string]interface{}) (Model, error)
	// DeleteMtsConfig deletes an mts config configuration
	DeleteMtsConfig(mb *message.Buffer) func(tenantId uuid.UUID) func(configID string) error
	// DeleteMtsConfigAndEmit deletes an mts config configuration and emits events
	DeleteMtsConfigAndEmit(tenantId uuid.UUID, configID string) error
	// GetMtsConfigById gets an mts config by ID
	GetMtsConfigById(tenantId uuid.UUID, configID string) (map[string]interface{}, error)
	// GetAllMtsConfigs gets all mts configs for a tenant
	GetAllMtsConfigs(tenantId uuid.UUID) ([]map[string]interface{}, error)
	// MtsConfigByIdProvider returns a provider for an mts config by ID
	MtsConfigByIdProvider(tenantId uuid.UUID, configID string) model.Provider[map[string]interface{}]
	// AllMtsConfigsProvider returns a provider for all mts configs for a tenant
	AllMtsConfigsProvider(tenantId uuid.UUID) model.Provider[[]map[string]interface{}]

	// Rankings operations
	// CreateRankings creates (or replaces) the tenant's rankings configuration
	CreateRankings(mb *message.Buffer) func(tenantId uuid.UUID) func(rankings map[string]interface{}) (Model, error)
	// CreateRankingsAndEmit creates the rankings configuration and emits events
	CreateRankingsAndEmit(tenantId uuid.UUID, rankings map[string]interface{}) (Model, error)
	// UpdateRankings updates the existing rankings configuration
	UpdateRankings(mb *message.Buffer) func(tenantId uuid.UUID) func(rankings map[string]interface{}) (Model, error)
	// UpdateRankingsAndEmit updates the rankings configuration and emits events
	UpdateRankingsAndEmit(tenantId uuid.UUID, rankings map[string]interface{}) (Model, error)
	// DeleteRankings deletes the rankings configuration
	DeleteRankings(mb *message.Buffer) func(tenantId uuid.UUID) error
	// DeleteRankingsAndEmit deletes the rankings configuration and emits events
	DeleteRankingsAndEmit(tenantId uuid.UUID) error
	// GetRankings gets the rankings configuration for a tenant
	GetRankings(tenantId uuid.UUID) (map[string]interface{}, error)
	// RankingsProvider returns a provider for the rankings configuration
	RankingsProvider(tenantId uuid.UUID) model.Provider[map[string]interface{}]

	// Kite config operations
	// CreateKiteConfig creates (or replaces) the tenant's kite-configs configuration
	CreateKiteConfig(mb *message.Buffer) func(tenantId uuid.UUID) func(cfg map[string]interface{}) (Model, error)
	// CreateKiteConfigAndEmit creates the kite-configs configuration and emits events
	CreateKiteConfigAndEmit(tenantId uuid.UUID, cfg map[string]interface{}) (Model, error)
	// UpdateKiteConfig updates the existing kite-configs configuration
	UpdateKiteConfig(mb *message.Buffer) func(tenantId uuid.UUID) func(cfg map[string]interface{}) (Model, error)
	// UpdateKiteConfigAndEmit updates the kite-configs configuration and emits events
	UpdateKiteConfigAndEmit(tenantId uuid.UUID, cfg map[string]interface{}) (Model, error)
	// DeleteKiteConfig deletes the kite-configs configuration
	DeleteKiteConfig(mb *message.Buffer) func(tenantId uuid.UUID) error
	// DeleteKiteConfigAndEmit deletes the kite-configs configuration and emits events
	DeleteKiteConfigAndEmit(tenantId uuid.UUID) error
	// GetKiteConfig gets the kite-configs configuration for a tenant
	GetKiteConfig(tenantId uuid.UUID) (map[string]interface{}, error)
	// KiteConfigProvider returns a provider for the kite-configs configuration
	KiteConfigProvider(tenantId uuid.UUID) model.Provider[map[string]interface{}]

	// Seed operations
	// SeedRpsRewards clears existing rps-rewards for a tenant and loads them from seed files
	SeedRpsRewards(tenantId uuid.UUID) (SeedResult, error)
	// SeedMtsConfigs clears existing mts configs for a tenant and loads them from seed files
	SeedMtsConfigs(tenantId uuid.UUID) (SeedResult, error)
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
}

// NewProcessor creates a new Processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

// tenantCtx returns p.ctx enriched with a fully-populated tenant.Model
// for tenantId.
//
// The configuration processor threads the tenant as a bare uuid.UUID
// (the REST layer is path-scoped), so the server context it was built
// with has no tenant at all. Without this, producer.TenantHeaderDecorator
// silently drops all four tenant headers and the downstream reload runs
// against the zero tenant. atlas-tenants owns the tenants table, so it
// has the region and versions needed to build a real model.
//
// A resolution failure aborts the caller rather than emitting
// tenant-free: the operation is meaningless for an unknown tenant.
func (p *ProcessorImpl) tenantCtx(tenantId uuid.UUID) (context.Context, error) {
	m, err := tenants.NewProcessor(p.l, p.ctx, p.db).GetById(tenantId)
	if err != nil {
		return nil, fmt.Errorf("resolve tenant %s for configuration emit: %w", tenantId, err)
	}
	t, err := atlastenant.Create(m.Id(), m.Region(), m.MajorVersion(), m.MinorVersion())
	if err != nil {
		return nil, fmt.Errorf("build tenant model %s: %w", tenantId, err)
	}
	return atlastenant.WithContext(p.ctx, t), nil
}

// Create creates a new route configuration
func (p *ProcessorImpl) CreateRoute(mb *message.Buffer) func(tenantId uuid.UUID) func(route map[string]interface{}) (Model, error) {
	return func(tenantId uuid.UUID) func(route map[string]interface{}) (Model, error) {
		return func(route map[string]interface{}) (Model, error) {
			// Check if configuration already exists
			existingProvider := GetByTenantIdAndResourceNameProvider(tenantId, "routes")(p.db)
			existing, err := existingProvider()

			var resourceData json.RawMessage

			if err == nil {
				// Configuration exists, update it
				var existingData map[string]interface{}
				if err := json.Unmarshal(existing.ResourceData, &existingData); err != nil {
					return Model{}, err
				}

				// Check if it's an array of resources
				if resources, ok := existingData["data"].([]interface{}); ok {
					// Add the new route to the array
					resources = append(resources, route)
					existingData["data"] = resources
					resourceData, err = json.Marshal(existingData)
					if err != nil {
						return Model{}, err
					}
				} else {
					// CreateRoute a new array with the existing resource and the new one
					resourceData, err = CreateRouteJsonData([]map[string]interface{}{route})
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

				// Add event to message buffer
				routeID := ""
				if id, ok := route["id"].(string); ok {
					routeID = id
				}
				if err := mb.Put(EventTopicConfigurationStatus, CreateRouteStatusEventProvider(tenantId, EventTypeRouteCreated, routeID)); err != nil {
					return Model{}, err
				}

				return m, nil
			} else if errors.Is(err, gorm.ErrRecordNotFound) {
				// Configuration doesn't exist, create it
				resourceData, err = CreateSingleRouteJsonData(route)
				if err != nil {
					return Model{}, err
				}

				entity := Entity{
					ID:           uuid.New(),
					TenantId:     tenantId,
					ResourceName: "routes",
					ResourceData: resourceData,
				}

				if err := CreateConfiguration(p.db, entity); err != nil {
					return Model{}, err
				}

				m, err := Make(entity)
				if err != nil {
					return Model{}, err
				}

				// Add event to message buffer
				routeID := ""
				if id, ok := route["id"].(string); ok {
					routeID = id
				}
				if err := mb.Put(EventTopicConfigurationStatus, CreateRouteStatusEventProvider(tenantId, EventTypeRouteCreated, routeID)); err != nil {
					return Model{}, err
				}

				return m, nil
			} else {
				// Other error
				return Model{}, err
			}
		}
	}
}

// CreateRouteAndEmit creates a new route configuration and emits events
func (p *ProcessorImpl) CreateRouteAndEmit(tenantId uuid.UUID, route map[string]interface{}) (Model, error) {
	ctx, err := p.tenantCtx(tenantId)
	if err != nil {
		return Model{}, err
	}
	var result Model
	txErr := database.ExecuteTransaction(p.db.WithContext(ctx), func(tx *gorm.DB) error {
		var err error
		result, err = message.EmitWithResult[Model, uuid.UUID](outbox.EmitProvider(p.l, ctx, tx))(func(mb *message.Buffer) func(uuid.UUID) (Model, error) {
			return func(tenantId uuid.UUID) (Model, error) {
				return NewProcessor(p.l, ctx, tx).CreateRoute(mb)(tenantId)(route)
			}
		})(tenantId)
		return err
	})
	return result, txErr
}

// Update updates an existing route configuration
func (p *ProcessorImpl) UpdateRoute(mb *message.Buffer) func(tenantId uuid.UUID) func(routeID string) func(route map[string]interface{}) (Model, error) {
	return func(tenantId uuid.UUID) func(routeID string) func(route map[string]interface{}) (Model, error) {
		return func(routeID string) func(route map[string]interface{}) (Model, error) {
			return func(route map[string]interface{}) (Model, error) {
				// Check if configuration exists
				existingProvider := GetByTenantIdAndResourceNameProvider(tenantId, "routes")(p.db)
				existing, err := existingProvider()
				if err != nil {
					return Model{}, err
				}

				var existingData map[string]interface{}
				if err := json.Unmarshal(existing.ResourceData, &existingData); err != nil {
					return Model{}, err
				}

				// Ensure the route ID matches
				route["id"] = routeID

				// Check if it's an array of resources
				if resources, ok := existingData["data"].([]interface{}); ok {
					found := false
					for i, resource := range resources {
						if resourceMap, ok := resource.(map[string]interface{}); ok {
							if id, ok := resourceMap["id"].(string); ok && id == routeID {
								resources[i] = route
								found = true
								break
							}
						}
					}

					if !found {
						return Model{}, errors.New("route not found")
					}

					existingData["data"] = resources
				} else if data, ok := existingData["data"].(map[string]interface{}); ok {
					if id, ok := data["id"].(string); ok && id == routeID {
						existingData["data"] = route
					} else {
						return Model{}, errors.New("route not found")
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

				// Add event to message buffer
				if err := mb.Put(EventTopicConfigurationStatus, CreateRouteStatusEventProvider(tenantId, EventTypeRouteUpdated, routeID)); err != nil {
					return Model{}, err
				}

				return m, nil
			}
		}
	}
}

// UpdateRouteAndEmit updates an existing route configuration and emits events
func (p *ProcessorImpl) UpdateRouteAndEmit(tenantId uuid.UUID, routeID string, route map[string]interface{}) (Model, error) {
	ctx, err := p.tenantCtx(tenantId)
	if err != nil {
		return Model{}, err
	}
	var result Model
	txErr := database.ExecuteTransaction(p.db.WithContext(ctx), func(tx *gorm.DB) error {
		var err error
		result, err = message.EmitWithResult[Model, uuid.UUID](outbox.EmitProvider(p.l, ctx, tx))(func(mb *message.Buffer) func(uuid.UUID) (Model, error) {
			return func(tenantId uuid.UUID) (Model, error) {
				return NewProcessor(p.l, ctx, tx).UpdateRoute(mb)(tenantId)(routeID)(route)
			}
		})(tenantId)
		return err
	})
	return result, txErr
}

// Delete deletes a route configuration
func (p *ProcessorImpl) DeleteRoute(mb *message.Buffer) func(tenantId uuid.UUID) func(routeID string) error {
	return func(tenantId uuid.UUID) func(routeID string) error {
		return func(routeID string) error {
			if err := DeleteConfiguration(p.db, tenantId, "routes", routeID); err != nil {
				return err
			}

			// Add event to message buffer
			if err := mb.Put(EventTopicConfigurationStatus, CreateRouteStatusEventProvider(tenantId, EventTypeRouteDeleted, routeID)); err != nil {
				return err
			}

			return nil
		}
	}
}

// DeleteRouteAndEmit deletes a route configuration and emits events
func (p *ProcessorImpl) DeleteRouteAndEmit(tenantId uuid.UUID, routeID string) error {
	ctx, err := p.tenantCtx(tenantId)
	if err != nil {
		return err
	}
	return database.ExecuteTransaction(p.db.WithContext(ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, ctx, tx))(func(mb *message.Buffer) error {
			return NewProcessor(p.l, ctx, tx).DeleteRoute(mb)(tenantId)(routeID)
		})
	})
}

// GetRouteById gets a route by ID
func (p *ProcessorImpl) GetRouteById(tenantId uuid.UUID, routeID string) (map[string]interface{}, error) {
	return p.RouteByIdProvider(tenantId, routeID)()
}

// GetAllRoutes gets all routes for a tenant
func (p *ProcessorImpl) GetAllRoutes(tenantId uuid.UUID) ([]map[string]interface{}, error) {
	return p.AllRoutesProvider(tenantId)()
}

// RouteByIdProvider returns a provider for a route by ID
func (p *ProcessorImpl) RouteByIdProvider(tenantId uuid.UUID, routeID string) model.Provider[map[string]interface{}] {
	return GetRouteByIdProvider(tenantId, routeID)(p.db)
}

// AllRoutesProvider returns a provider for all routes for a tenant
func (p *ProcessorImpl) AllRoutesProvider(tenantId uuid.UUID) model.Provider[[]map[string]interface{}] {
	return GetAllRoutesProvider(tenantId)(p.db)
}

// CreateVessel creates a new vessel configuration
func (p *ProcessorImpl) CreateVessel(mb *message.Buffer) func(tenantId uuid.UUID) func(vessel map[string]interface{}) (Model, error) {
	return func(tenantId uuid.UUID) func(vessel map[string]interface{}) (Model, error) {
		return func(vessel map[string]interface{}) (Model, error) {
			// Check if configuration already exists
			existingProvider := GetByTenantIdAndResourceNameProvider(tenantId, "vessels")(p.db)
			existing, err := existingProvider()

			var resourceData json.RawMessage

			if err == nil {
				// Configuration exists, update it
				var existingData map[string]interface{}
				if err := json.Unmarshal(existing.ResourceData, &existingData); err != nil {
					return Model{}, err
				}

				// Check if it's an array of resources
				if resources, ok := existingData["data"].([]interface{}); ok {
					// Add the new vessel to the array
					resources = append(resources, vessel)
					existingData["data"] = resources
					resourceData, err = json.Marshal(existingData)
					if err != nil {
						return Model{}, err
					}
				} else {
					// CreateRoute a new array with the existing resource and the new one
					resourceData, err = CreateVesselJsonData([]map[string]interface{}{vessel})
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

				// Add event to message buffer
				vesselID := ""
				if id, ok := vessel["id"].(string); ok {
					vesselID = id
				}
				if err := mb.Put(EventTopicConfigurationStatus, CreateVesselStatusEventProvider(tenantId, EventTypeVesselCreated, vesselID)); err != nil {
					return Model{}, err
				}

				return m, nil
			} else if errors.Is(err, gorm.ErrRecordNotFound) {
				// Configuration doesn't exist, create it
				resourceData, err = CreateSingleVesselJsonData(vessel)
				if err != nil {
					return Model{}, err
				}

				entity := Entity{
					ID:           uuid.New(),
					TenantId:     tenantId,
					ResourceName: "vessels",
					ResourceData: resourceData,
				}

				if err := CreateConfiguration(p.db, entity); err != nil {
					return Model{}, err
				}

				m, err := Make(entity)
				if err != nil {
					return Model{}, err
				}

				// Add event to message buffer
				vesselID := ""
				if id, ok := vessel["id"].(string); ok {
					vesselID = id
				}
				if err := mb.Put(EventTopicConfigurationStatus, CreateVesselStatusEventProvider(tenantId, EventTypeVesselCreated, vesselID)); err != nil {
					return Model{}, err
				}

				return m, nil
			} else {
				// Other error
				return Model{}, err
			}
		}
	}
}

// CreateVesselAndEmit creates a new vessel configuration and emits events
func (p *ProcessorImpl) CreateVesselAndEmit(tenantId uuid.UUID, vessel map[string]interface{}) (Model, error) {
	ctx, err := p.tenantCtx(tenantId)
	if err != nil {
		return Model{}, err
	}
	var result Model
	txErr := database.ExecuteTransaction(p.db.WithContext(ctx), func(tx *gorm.DB) error {
		var err error
		result, err = message.EmitWithResult[Model, uuid.UUID](outbox.EmitProvider(p.l, ctx, tx))(func(mb *message.Buffer) func(uuid.UUID) (Model, error) {
			return func(tenantId uuid.UUID) (Model, error) {
				return NewProcessor(p.l, ctx, tx).CreateVessel(mb)(tenantId)(vessel)
			}
		})(tenantId)
		return err
	})
	return result, txErr
}

// UpdateVessel updates an existing vessel configuration
func (p *ProcessorImpl) UpdateVessel(mb *message.Buffer) func(tenantId uuid.UUID) func(vesselID string) func(vessel map[string]interface{}) (Model, error) {
	return func(tenantId uuid.UUID) func(vesselID string) func(vessel map[string]interface{}) (Model, error) {
		return func(vesselID string) func(vessel map[string]interface{}) (Model, error) {
			return func(vessel map[string]interface{}) (Model, error) {
				// Check if configuration exists
				existingProvider := GetByTenantIdAndResourceNameProvider(tenantId, "vessels")(p.db)
				existing, err := existingProvider()
				if err != nil {
					return Model{}, err
				}

				var existingData map[string]interface{}
				if err := json.Unmarshal(existing.ResourceData, &existingData); err != nil {
					return Model{}, err
				}

				// Ensure the vessel ID matches
				vessel["id"] = vesselID

				// Check if it's an array of resources
				if resources, ok := existingData["data"].([]interface{}); ok {
					found := false
					for i, resource := range resources {
						if resourceMap, ok := resource.(map[string]interface{}); ok {
							if id, ok := resourceMap["id"].(string); ok && id == vesselID {
								resources[i] = vessel
								found = true
								break
							}
						}
					}

					if !found {
						return Model{}, errors.New("vessel not found")
					}

					existingData["data"] = resources
				} else if data, ok := existingData["data"].(map[string]interface{}); ok {
					if id, ok := data["id"].(string); ok && id == vesselID {
						existingData["data"] = vessel
					} else {
						return Model{}, errors.New("vessel not found")
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

				// Add event to message buffer
				if err := mb.Put(EventTopicConfigurationStatus, CreateVesselStatusEventProvider(tenantId, EventTypeVesselUpdated, vesselID)); err != nil {
					return Model{}, err
				}

				return m, nil
			}
		}
	}
}

// UpdateVesselAndEmit updates an existing vessel configuration and emits events
func (p *ProcessorImpl) UpdateVesselAndEmit(tenantId uuid.UUID, vesselID string, vessel map[string]interface{}) (Model, error) {
	ctx, err := p.tenantCtx(tenantId)
	if err != nil {
		return Model{}, err
	}
	var result Model
	txErr := database.ExecuteTransaction(p.db.WithContext(ctx), func(tx *gorm.DB) error {
		var err error
		result, err = message.EmitWithResult[Model, uuid.UUID](outbox.EmitProvider(p.l, ctx, tx))(func(mb *message.Buffer) func(uuid.UUID) (Model, error) {
			return func(tenantId uuid.UUID) (Model, error) {
				return NewProcessor(p.l, ctx, tx).UpdateVessel(mb)(tenantId)(vesselID)(vessel)
			}
		})(tenantId)
		return err
	})
	return result, txErr
}

// DeleteVessel deletes a vessel configuration
func (p *ProcessorImpl) DeleteVessel(mb *message.Buffer) func(tenantId uuid.UUID) func(vesselID string) error {
	return func(tenantId uuid.UUID) func(vesselID string) error {
		return func(vesselID string) error {
			if err := DeleteConfiguration(p.db, tenantId, "vessels", vesselID); err != nil {
				return err
			}

			// Add event to message buffer
			if err := mb.Put(EventTopicConfigurationStatus, CreateVesselStatusEventProvider(tenantId, EventTypeVesselDeleted, vesselID)); err != nil {
				return err
			}

			return nil
		}
	}
}

// DeleteVesselAndEmit deletes a vessel configuration and emits events
func (p *ProcessorImpl) DeleteVesselAndEmit(tenantId uuid.UUID, vesselID string) error {
	ctx, err := p.tenantCtx(tenantId)
	if err != nil {
		return err
	}
	return database.ExecuteTransaction(p.db.WithContext(ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, ctx, tx))(func(mb *message.Buffer) error {
			return NewProcessor(p.l, ctx, tx).DeleteVessel(mb)(tenantId)(vesselID)
		})
	})
}

// GetVesselById gets a vessel by ID
func (p *ProcessorImpl) GetVesselById(tenantId uuid.UUID, vesselID string) (map[string]interface{}, error) {
	return p.VesselByIdProvider(tenantId, vesselID)()
}

// GetAllVessels gets all vessels for a tenant
func (p *ProcessorImpl) GetAllVessels(tenantId uuid.UUID) ([]map[string]interface{}, error) {
	return p.AllVesselsProvider(tenantId)()
}

// VesselByIdProvider returns a provider for a vessel by ID
func (p *ProcessorImpl) VesselByIdProvider(tenantId uuid.UUID, vesselID string) model.Provider[map[string]interface{}] {
	return GetVesselByIdProvider(tenantId, vesselID)(p.db)
}

// AllVesselsProvider returns a provider for all vessels for a tenant
func (p *ProcessorImpl) AllVesselsProvider(tenantId uuid.UUID) model.Provider[[]map[string]interface{}] {
	return GetAllVesselsProvider(tenantId)(p.db)
}

// CreateMtsConfig creates a new mts config configuration
func (p *ProcessorImpl) CreateMtsConfig(mb *message.Buffer) func(tenantId uuid.UUID) func(config map[string]interface{}) (Model, error) {
	return func(tenantId uuid.UUID) func(config map[string]interface{}) (Model, error) {
		return func(config map[string]interface{}) (Model, error) {
			// Check if configuration already exists
			existingProvider := GetByTenantIdAndResourceNameProvider(tenantId, "mts-configs")(p.db)
			existing, err := existingProvider()

			var resourceData json.RawMessage

			if err == nil {
				// Configuration exists, update it
				var existingData map[string]interface{}
				if err := json.Unmarshal(existing.ResourceData, &existingData); err != nil {
					return Model{}, err
				}

				// Check if it's an array of resources
				if resources, ok := existingData["data"].([]interface{}); ok {
					// Add the new config to the array
					resources = append(resources, config)
					existingData["data"] = resources
					resourceData, err = json.Marshal(existingData)
					if err != nil {
						return Model{}, err
					}
				} else {
					// Create a new array with the existing resource and the new one
					resourceData, err = CreateMtsConfigJsonData([]map[string]interface{}{config})
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

				// Add event to message buffer
				configID := ""
				if id, ok := config["id"].(string); ok {
					configID = id
				}
				if err := mb.Put(EventTopicConfigurationStatus, CreateMtsConfigStatusEventProvider(tenantId, EventTypeMtsConfigCreated, configID)); err != nil {
					return Model{}, err
				}

				return m, nil
			} else if errors.Is(err, gorm.ErrRecordNotFound) {
				// Configuration doesn't exist, create it
				resourceData, err = CreateSingleMtsConfigJsonData(config)
				if err != nil {
					return Model{}, err
				}

				entity := Entity{
					ID:           uuid.New(),
					TenantId:     tenantId,
					ResourceName: "mts-configs",
					ResourceData: resourceData,
				}

				if err := CreateConfiguration(p.db, entity); err != nil {
					return Model{}, err
				}

				m, err := Make(entity)
				if err != nil {
					return Model{}, err
				}

				// Add event to message buffer
				configID := ""
				if id, ok := config["id"].(string); ok {
					configID = id
				}
				if err := mb.Put(EventTopicConfigurationStatus, CreateMtsConfigStatusEventProvider(tenantId, EventTypeMtsConfigCreated, configID)); err != nil {
					return Model{}, err
				}

				return m, nil
			} else {
				// Other error
				return Model{}, err
			}
		}
	}
}

// CreateMtsConfigAndEmit creates a new mts config configuration and emits events
func (p *ProcessorImpl) CreateMtsConfigAndEmit(tenantId uuid.UUID, config map[string]interface{}) (Model, error) {
	ctx, err := p.tenantCtx(tenantId)
	if err != nil {
		return Model{}, err
	}
	var result Model
	txErr := database.ExecuteTransaction(p.db.WithContext(ctx), func(tx *gorm.DB) error {
		var err error
		result, err = message.EmitWithResult[Model, uuid.UUID](outbox.EmitProvider(p.l, ctx, tx))(func(mb *message.Buffer) func(uuid.UUID) (Model, error) {
			return func(tenantId uuid.UUID) (Model, error) {
				return NewProcessor(p.l, ctx, tx).CreateMtsConfig(mb)(tenantId)(config)
			}
		})(tenantId)
		return err
	})
	return result, txErr
}

// UpdateMtsConfig updates an existing mts config configuration
func (p *ProcessorImpl) UpdateMtsConfig(mb *message.Buffer) func(tenantId uuid.UUID) func(configID string) func(config map[string]interface{}) (Model, error) {
	return func(tenantId uuid.UUID) func(configID string) func(config map[string]interface{}) (Model, error) {
		return func(configID string) func(config map[string]interface{}) (Model, error) {
			return func(config map[string]interface{}) (Model, error) {
				// Check if configuration exists
				existingProvider := GetByTenantIdAndResourceNameProvider(tenantId, "mts-configs")(p.db)
				existing, err := existingProvider()
				if err != nil {
					return Model{}, err
				}

				var existingData map[string]interface{}
				if err := json.Unmarshal(existing.ResourceData, &existingData); err != nil {
					return Model{}, err
				}

				// Ensure the config ID matches
				config["id"] = configID

				// Check if it's an array of resources
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
						return Model{}, errors.New("mts config not found")
					}

					existingData["data"] = resources
				} else if data, ok := existingData["data"].(map[string]interface{}); ok {
					if id, ok := data["id"].(string); ok && id == configID {
						existingData["data"] = config
					} else {
						return Model{}, errors.New("mts config not found")
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

				// Add event to message buffer
				if err := mb.Put(EventTopicConfigurationStatus, CreateMtsConfigStatusEventProvider(tenantId, EventTypeMtsConfigUpdated, configID)); err != nil {
					return Model{}, err
				}

				return m, nil
			}
		}
	}
}

// UpdateMtsConfigAndEmit updates an existing mts config configuration and emits events
func (p *ProcessorImpl) UpdateMtsConfigAndEmit(tenantId uuid.UUID, configID string, config map[string]interface{}) (Model, error) {
	ctx, err := p.tenantCtx(tenantId)
	if err != nil {
		return Model{}, err
	}
	var result Model
	txErr := database.ExecuteTransaction(p.db.WithContext(ctx), func(tx *gorm.DB) error {
		var err error
		result, err = message.EmitWithResult[Model, uuid.UUID](outbox.EmitProvider(p.l, ctx, tx))(func(mb *message.Buffer) func(uuid.UUID) (Model, error) {
			return func(tenantId uuid.UUID) (Model, error) {
				return NewProcessor(p.l, ctx, tx).UpdateMtsConfig(mb)(tenantId)(configID)(config)
			}
		})(tenantId)
		return err
	})
	return result, txErr
}

// DeleteMtsConfig deletes an mts config configuration
func (p *ProcessorImpl) DeleteMtsConfig(mb *message.Buffer) func(tenantId uuid.UUID) func(configID string) error {
	return func(tenantId uuid.UUID) func(configID string) error {
		return func(configID string) error {
			if err := DeleteConfiguration(p.db, tenantId, "mts-configs", configID); err != nil {
				return err
			}

			// Add event to message buffer
			if err := mb.Put(EventTopicConfigurationStatus, CreateMtsConfigStatusEventProvider(tenantId, EventTypeMtsConfigDeleted, configID)); err != nil {
				return err
			}

			return nil
		}
	}
}

// DeleteMtsConfigAndEmit deletes an mts config configuration and emits events
func (p *ProcessorImpl) DeleteMtsConfigAndEmit(tenantId uuid.UUID, configID string) error {
	ctx, err := p.tenantCtx(tenantId)
	if err != nil {
		return err
	}
	return database.ExecuteTransaction(p.db.WithContext(ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, ctx, tx))(func(mb *message.Buffer) error {
			return NewProcessor(p.l, ctx, tx).DeleteMtsConfig(mb)(tenantId)(configID)
		})
	})
}

// GetMtsConfigById gets an mts config by ID
func (p *ProcessorImpl) GetMtsConfigById(tenantId uuid.UUID, configID string) (map[string]interface{}, error) {
	return p.MtsConfigByIdProvider(tenantId, configID)()
}

// GetAllMtsConfigs gets all mts configs for a tenant
func (p *ProcessorImpl) GetAllMtsConfigs(tenantId uuid.UUID) ([]map[string]interface{}, error) {
	return p.AllMtsConfigsProvider(tenantId)()
}

// MtsConfigByIdProvider returns a provider for an mts config by ID
func (p *ProcessorImpl) MtsConfigByIdProvider(tenantId uuid.UUID, configID string) model.Provider[map[string]interface{}] {
	return GetMtsConfigByIdProvider(tenantId, configID)(p.db)
}

// AllMtsConfigsProvider returns a provider for all mts configs for a tenant
func (p *ProcessorImpl) AllMtsConfigsProvider(tenantId uuid.UUID) model.Provider[[]map[string]interface{}] {
	return GetAllMtsConfigsProvider(tenantId)(p.db)
}

// CreateInstanceRoute creates a new instance route configuration
func (p *ProcessorImpl) CreateInstanceRoute(mb *message.Buffer) func(tenantId uuid.UUID) func(route map[string]interface{}) (Model, error) {
	return func(tenantId uuid.UUID) func(route map[string]interface{}) (Model, error) {
		return func(route map[string]interface{}) (Model, error) {
			existingProvider := GetByTenantIdAndResourceNameProvider(tenantId, "instance-routes")(p.db)
			existing, err := existingProvider()

			var resourceData json.RawMessage

			if err == nil {
				var existingData map[string]interface{}
				if err := json.Unmarshal(existing.ResourceData, &existingData); err != nil {
					return Model{}, err
				}

				if resources, ok := existingData["data"].([]interface{}); ok {
					resources = append(resources, route)
					existingData["data"] = resources
					resourceData, err = json.Marshal(existingData)
					if err != nil {
						return Model{}, err
					}
				} else {
					resourceData, err = CreateInstanceRouteJsonData([]map[string]interface{}{route})
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

				routeID := ""
				if id, ok := route["id"].(string); ok {
					routeID = id
				}
				if err := mb.Put(EventTopicConfigurationStatus, CreateInstanceRouteStatusEventProvider(tenantId, EventTypeInstanceRouteCreated, routeID)); err != nil {
					return Model{}, err
				}

				return m, nil
			} else if errors.Is(err, gorm.ErrRecordNotFound) {
				resourceData, err = CreateSingleInstanceRouteJsonData(route)
				if err != nil {
					return Model{}, err
				}

				entity := Entity{
					ID:           uuid.New(),
					TenantId:     tenantId,
					ResourceName: "instance-routes",
					ResourceData: resourceData,
				}

				if err := CreateConfiguration(p.db, entity); err != nil {
					return Model{}, err
				}

				m, err := Make(entity)
				if err != nil {
					return Model{}, err
				}

				routeID := ""
				if id, ok := route["id"].(string); ok {
					routeID = id
				}
				if err := mb.Put(EventTopicConfigurationStatus, CreateInstanceRouteStatusEventProvider(tenantId, EventTypeInstanceRouteCreated, routeID)); err != nil {
					return Model{}, err
				}

				return m, nil
			} else {
				return Model{}, err
			}
		}
	}
}

// CreateInstanceRouteAndEmit creates a new instance route configuration and emits events
func (p *ProcessorImpl) CreateInstanceRouteAndEmit(tenantId uuid.UUID, route map[string]interface{}) (Model, error) {
	ctx, err := p.tenantCtx(tenantId)
	if err != nil {
		return Model{}, err
	}
	var result Model
	txErr := database.ExecuteTransaction(p.db.WithContext(ctx), func(tx *gorm.DB) error {
		var err error
		result, err = message.EmitWithResult[Model, uuid.UUID](outbox.EmitProvider(p.l, ctx, tx))(func(mb *message.Buffer) func(uuid.UUID) (Model, error) {
			return func(tenantId uuid.UUID) (Model, error) {
				return NewProcessor(p.l, ctx, tx).CreateInstanceRoute(mb)(tenantId)(route)
			}
		})(tenantId)
		return err
	})
	return result, txErr
}

// UpdateInstanceRoute updates an existing instance route configuration
func (p *ProcessorImpl) UpdateInstanceRoute(mb *message.Buffer) func(tenantId uuid.UUID) func(routeID string) func(route map[string]interface{}) (Model, error) {
	return func(tenantId uuid.UUID) func(routeID string) func(route map[string]interface{}) (Model, error) {
		return func(routeID string) func(route map[string]interface{}) (Model, error) {
			return func(route map[string]interface{}) (Model, error) {
				existingProvider := GetByTenantIdAndResourceNameProvider(tenantId, "instance-routes")(p.db)
				existing, err := existingProvider()
				if err != nil {
					return Model{}, err
				}

				var existingData map[string]interface{}
				if err := json.Unmarshal(existing.ResourceData, &existingData); err != nil {
					return Model{}, err
				}

				route["id"] = routeID

				if resources, ok := existingData["data"].([]interface{}); ok {
					found := false
					for i, resource := range resources {
						if resourceMap, ok := resource.(map[string]interface{}); ok {
							if id, ok := resourceMap["id"].(string); ok && id == routeID {
								resources[i] = route
								found = true
								break
							}
						}
					}

					if !found {
						return Model{}, errors.New("instance route not found")
					}

					existingData["data"] = resources
				} else if data, ok := existingData["data"].(map[string]interface{}); ok {
					if id, ok := data["id"].(string); ok && id == routeID {
						existingData["data"] = route
					} else {
						return Model{}, errors.New("instance route not found")
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

				if err := mb.Put(EventTopicConfigurationStatus, CreateInstanceRouteStatusEventProvider(tenantId, EventTypeInstanceRouteUpdated, routeID)); err != nil {
					return Model{}, err
				}

				return m, nil
			}
		}
	}
}

// UpdateInstanceRouteAndEmit updates an existing instance route configuration and emits events
func (p *ProcessorImpl) UpdateInstanceRouteAndEmit(tenantId uuid.UUID, routeID string, route map[string]interface{}) (Model, error) {
	ctx, err := p.tenantCtx(tenantId)
	if err != nil {
		return Model{}, err
	}
	var result Model
	txErr := database.ExecuteTransaction(p.db.WithContext(ctx), func(tx *gorm.DB) error {
		var err error
		result, err = message.EmitWithResult[Model, uuid.UUID](outbox.EmitProvider(p.l, ctx, tx))(func(mb *message.Buffer) func(uuid.UUID) (Model, error) {
			return func(tenantId uuid.UUID) (Model, error) {
				return NewProcessor(p.l, ctx, tx).UpdateInstanceRoute(mb)(tenantId)(routeID)(route)
			}
		})(tenantId)
		return err
	})
	return result, txErr
}

// DeleteInstanceRoute deletes an instance route configuration
func (p *ProcessorImpl) DeleteInstanceRoute(mb *message.Buffer) func(tenantId uuid.UUID) func(routeID string) error {
	return func(tenantId uuid.UUID) func(routeID string) error {
		return func(routeID string) error {
			if err := DeleteConfiguration(p.db, tenantId, "instance-routes", routeID); err != nil {
				return err
			}

			if err := mb.Put(EventTopicConfigurationStatus, CreateInstanceRouteStatusEventProvider(tenantId, EventTypeInstanceRouteDeleted, routeID)); err != nil {
				return err
			}

			return nil
		}
	}
}

// DeleteInstanceRouteAndEmit deletes an instance route configuration and emits events
func (p *ProcessorImpl) DeleteInstanceRouteAndEmit(tenantId uuid.UUID, routeID string) error {
	ctx, err := p.tenantCtx(tenantId)
	if err != nil {
		return err
	}
	return database.ExecuteTransaction(p.db.WithContext(ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, ctx, tx))(func(mb *message.Buffer) error {
			return NewProcessor(p.l, ctx, tx).DeleteInstanceRoute(mb)(tenantId)(routeID)
		})
	})
}

// GetInstanceRouteById gets an instance route by ID
func (p *ProcessorImpl) GetInstanceRouteById(tenantId uuid.UUID, routeID string) (map[string]interface{}, error) {
	return p.InstanceRouteByIdProvider(tenantId, routeID)()
}

// GetAllInstanceRoutes gets all instance routes for a tenant
func (p *ProcessorImpl) GetAllInstanceRoutes(tenantId uuid.UUID) ([]map[string]interface{}, error) {
	return p.AllInstanceRoutesProvider(tenantId)()
}

// InstanceRouteByIdProvider returns a provider for an instance route by ID
func (p *ProcessorImpl) InstanceRouteByIdProvider(tenantId uuid.UUID, routeID string) model.Provider[map[string]interface{}] {
	return GetInstanceRouteByIdProvider(tenantId, routeID)(p.db)
}

// AllInstanceRoutesProvider returns a provider for all instance routes for a tenant
func (p *ProcessorImpl) AllInstanceRoutesProvider(tenantId uuid.UUID) model.Provider[[]map[string]interface{}] {
	return GetAllInstanceRoutesProvider(tenantId)(p.db)
}

// CreateRpsReward creates a new rps-reward configuration
func (p *ProcessorImpl) CreateRpsReward(mb *message.Buffer) func(tenantId uuid.UUID) func(rpsReward map[string]interface{}) (Model, error) {
	return func(tenantId uuid.UUID) func(rpsReward map[string]interface{}) (Model, error) {
		return func(rpsReward map[string]interface{}) (Model, error) {
			// Check if configuration already exists
			existingProvider := GetByTenantIdAndResourceNameProvider(tenantId, "rps-rewards")(p.db)
			existing, err := existingProvider()

			var resourceData json.RawMessage

			if err == nil {
				// Configuration exists, update it
				var existingData map[string]interface{}
				if err := json.Unmarshal(existing.ResourceData, &existingData); err != nil {
					return Model{}, err
				}

				// Check if it's an array of resources
				if resources, ok := existingData["data"].([]interface{}); ok {
					// Add the new rps-reward to the array
					resources = append(resources, rpsReward)
					existingData["data"] = resources
					resourceData, err = json.Marshal(existingData)
					if err != nil {
						return Model{}, err
					}
				} else {
					// Create a new array with the existing resource and the new one
					resourceData, err = CreateRpsRewardJsonData([]map[string]interface{}{rpsReward})
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

				// Add event to message buffer
				rpsRewardID := ""
				if id, ok := rpsReward["id"].(string); ok {
					rpsRewardID = id
				}
				if err := mb.Put(EventTopicConfigurationStatus, CreateRpsRewardStatusEventProvider(tenantId, EventTypeRpsRewardCreated, rpsRewardID)); err != nil {
					return Model{}, err
				}

				return m, nil
			} else if errors.Is(err, gorm.ErrRecordNotFound) {
				// Configuration doesn't exist, create it
				resourceData, err = CreateSingleRpsRewardJsonData(rpsReward)
				if err != nil {
					return Model{}, err
				}

				entity := Entity{
					ID:           uuid.New(),
					TenantId:     tenantId,
					ResourceName: "rps-rewards",
					ResourceData: resourceData,
				}

				if err := CreateConfiguration(p.db, entity); err != nil {
					return Model{}, err
				}

				m, err := Make(entity)
				if err != nil {
					return Model{}, err
				}

				// Add event to message buffer
				rpsRewardID := ""
				if id, ok := rpsReward["id"].(string); ok {
					rpsRewardID = id
				}
				if err := mb.Put(EventTopicConfigurationStatus, CreateRpsRewardStatusEventProvider(tenantId, EventTypeRpsRewardCreated, rpsRewardID)); err != nil {
					return Model{}, err
				}

				return m, nil
			} else {
				// Other error
				return Model{}, err
			}
		}
	}
}

// CreateRpsRewardAndEmit creates a new rps-reward configuration and emits events
func (p *ProcessorImpl) CreateRpsRewardAndEmit(tenantId uuid.UUID, rpsReward map[string]interface{}) (Model, error) {
	ctx, err := p.tenantCtx(tenantId)
	if err != nil {
		return Model{}, err
	}
	var result Model
	txErr := database.ExecuteTransaction(p.db.WithContext(ctx), func(tx *gorm.DB) error {
		var err error
		result, err = message.EmitWithResult[Model, uuid.UUID](outbox.EmitProvider(p.l, ctx, tx))(func(mb *message.Buffer) func(uuid.UUID) (Model, error) {
			return func(tenantId uuid.UUID) (Model, error) {
				return NewProcessor(p.l, ctx, tx).CreateRpsReward(mb)(tenantId)(rpsReward)
			}
		})(tenantId)
		return err
	})
	return result, txErr
}

// UpdateRpsReward updates an existing rps-reward configuration
func (p *ProcessorImpl) UpdateRpsReward(mb *message.Buffer) func(tenantId uuid.UUID) func(rpsRewardID string) func(rpsReward map[string]interface{}) (Model, error) {
	return func(tenantId uuid.UUID) func(rpsRewardID string) func(rpsReward map[string]interface{}) (Model, error) {
		return func(rpsRewardID string) func(rpsReward map[string]interface{}) (Model, error) {
			return func(rpsReward map[string]interface{}) (Model, error) {
				// Check if configuration exists
				existingProvider := GetByTenantIdAndResourceNameProvider(tenantId, "rps-rewards")(p.db)
				existing, err := existingProvider()
				if err != nil {
					return Model{}, err
				}

				var existingData map[string]interface{}
				if err := json.Unmarshal(existing.ResourceData, &existingData); err != nil {
					return Model{}, err
				}

				// Ensure the rps-reward ID matches
				rpsReward["id"] = rpsRewardID

				// Check if it's an array of resources
				if resources, ok := existingData["data"].([]interface{}); ok {
					found := false
					for i, resource := range resources {
						if resourceMap, ok := resource.(map[string]interface{}); ok {
							if id, ok := resourceMap["id"].(string); ok && id == rpsRewardID {
								resources[i] = rpsReward
								found = true
								break
							}
						}
					}

					if !found {
						return Model{}, errors.New("rps-reward not found")
					}

					existingData["data"] = resources
				} else if data, ok := existingData["data"].(map[string]interface{}); ok {
					if id, ok := data["id"].(string); ok && id == rpsRewardID {
						existingData["data"] = rpsReward
					} else {
						return Model{}, errors.New("rps-reward not found")
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

				// Add event to message buffer
				if err := mb.Put(EventTopicConfigurationStatus, CreateRpsRewardStatusEventProvider(tenantId, EventTypeRpsRewardUpdated, rpsRewardID)); err != nil {
					return Model{}, err
				}

				return m, nil
			}
		}
	}
}

// UpdateRpsRewardAndEmit updates an existing rps-reward configuration and emits events
func (p *ProcessorImpl) UpdateRpsRewardAndEmit(tenantId uuid.UUID, rpsRewardID string, rpsReward map[string]interface{}) (Model, error) {
	ctx, err := p.tenantCtx(tenantId)
	if err != nil {
		return Model{}, err
	}
	var result Model
	txErr := database.ExecuteTransaction(p.db.WithContext(ctx), func(tx *gorm.DB) error {
		var err error
		result, err = message.EmitWithResult[Model, uuid.UUID](outbox.EmitProvider(p.l, ctx, tx))(func(mb *message.Buffer) func(uuid.UUID) (Model, error) {
			return func(tenantId uuid.UUID) (Model, error) {
				return NewProcessor(p.l, ctx, tx).UpdateRpsReward(mb)(tenantId)(rpsRewardID)(rpsReward)
			}
		})(tenantId)
		return err
	})
	return result, txErr
}

// DeleteRpsReward deletes a rps-reward configuration
func (p *ProcessorImpl) DeleteRpsReward(mb *message.Buffer) func(tenantId uuid.UUID) func(rpsRewardID string) error {
	return func(tenantId uuid.UUID) func(rpsRewardID string) error {
		return func(rpsRewardID string) error {
			if err := DeleteConfiguration(p.db, tenantId, "rps-rewards", rpsRewardID); err != nil {
				return err
			}

			// Add event to message buffer
			if err := mb.Put(EventTopicConfigurationStatus, CreateRpsRewardStatusEventProvider(tenantId, EventTypeRpsRewardDeleted, rpsRewardID)); err != nil {
				return err
			}

			return nil
		}
	}
}

// DeleteRpsRewardAndEmit deletes a rps-reward configuration and emits events
func (p *ProcessorImpl) DeleteRpsRewardAndEmit(tenantId uuid.UUID, rpsRewardID string) error {
	ctx, err := p.tenantCtx(tenantId)
	if err != nil {
		return err
	}
	return database.ExecuteTransaction(p.db.WithContext(ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, ctx, tx))(func(mb *message.Buffer) error {
			return NewProcessor(p.l, ctx, tx).DeleteRpsReward(mb)(tenantId)(rpsRewardID)
		})
	})
}

// GetRpsRewardById gets a rps-reward by ID
func (p *ProcessorImpl) GetRpsRewardById(tenantId uuid.UUID, rpsRewardID string) (map[string]interface{}, error) {
	return p.RpsRewardByIdProvider(tenantId, rpsRewardID)()
}

// GetAllRpsRewards gets all rps-rewards for a tenant
func (p *ProcessorImpl) GetAllRpsRewards(tenantId uuid.UUID) ([]map[string]interface{}, error) {
	return p.AllRpsRewardsProvider(tenantId)()
}

// RpsRewardByIdProvider returns a provider for a rps-reward by ID
func (p *ProcessorImpl) RpsRewardByIdProvider(tenantId uuid.UUID, rpsRewardID string) model.Provider[map[string]interface{}] {
	return GetRpsRewardByIdProvider(tenantId, rpsRewardID)(p.db)
}

// AllRpsRewardsProvider returns a provider for all rps-rewards for a tenant
func (p *ProcessorImpl) AllRpsRewardsProvider(tenantId uuid.UUID) model.Provider[[]map[string]interface{}] {
	return GetAllRpsRewardsProvider(tenantId)(p.db)
}

// SeedRpsRewards clears existing rps-rewards for a tenant and loads them from seed files
func (p *ProcessorImpl) SeedRpsRewards(tenantId uuid.UUID) (SeedResult, error) {
	p.l.Infof("Seeding rps-rewards for tenant [%s]", tenantId)

	result := SeedResult{}

	// Delete all existing rps-rewards for this tenant
	deletedCount, err := DeleteConfigurationByResourceName(p.db, tenantId, "rps-rewards")
	if err != nil {
		return result, fmt.Errorf("failed to clear existing rps-rewards: %w", err)
	}
	result.DeletedCount = int(deletedCount)

	// Load rps-reward files from the filesystem
	rpsRewards, loadErrors := LoadRpsRewardFiles()
	for _, err := range loadErrors {
		result.Errors = append(result.Errors, err.Error())
		result.FailedCount++
	}

	// Create each rps-reward
	for _, rpsReward := range rpsRewards {
		id, _ := rpsReward["id"].(string)
		_, err := p.CreateRpsRewardAndEmit(tenantId, rpsReward)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: failed to create: %v", id, err))
			result.FailedCount++
			continue
		}
		result.CreatedCount++
	}

	p.l.Infof("RpsReward seed complete for tenant [%s]: deleted=%d, created=%d, failed=%d",
		tenantId, result.DeletedCount, result.CreatedCount, result.FailedCount)

	return result, nil
}

// SeedMtsConfigs clears existing mts configs for a tenant and loads them from seed files
func (p *ProcessorImpl) SeedMtsConfigs(tenantId uuid.UUID) (SeedResult, error) {
	p.l.Infof("Seeding mts configs for tenant [%s]", tenantId)

	result := SeedResult{}

	// Delete all existing mts configs for this tenant
	deletedCount, err := DeleteConfigurationByResourceName(p.db, tenantId, "mts-configs")
	if err != nil {
		return result, fmt.Errorf("failed to clear existing mts configs: %w", err)
	}
	result.DeletedCount = int(deletedCount)

	// Load mts config files from the filesystem
	configs, loadErrors := LoadMtsConfigFiles()
	for _, err := range loadErrors {
		result.Errors = append(result.Errors, err.Error())
		result.FailedCount++
	}

	// Create each mts config
	for _, config := range configs {
		id, _ := config["id"].(string)
		_, err := p.CreateMtsConfigAndEmit(tenantId, config)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: failed to create: %v", id, err))
			result.FailedCount++
			continue
		}
		result.CreatedCount++
	}

	p.l.Infof("MTS config seed complete for tenant [%s]: deleted=%d, created=%d, failed=%d",
		tenantId, result.DeletedCount, result.CreatedCount, result.FailedCount)

	return result, nil
}

// CreateRankings creates (or replaces) the tenant's rankings configuration
func (p *ProcessorImpl) CreateRankings(mb *message.Buffer) func(tenantId uuid.UUID) func(rankings map[string]interface{}) (Model, error) {
	return func(tenantId uuid.UUID) func(rankings map[string]interface{}) (Model, error) {
		return func(rankings map[string]interface{}) (Model, error) {
			rankingsId := ""
			if id, ok := rankings["id"].(string); ok {
				rankingsId = id
			}

			resourceData, err := CreateSingleRankingsJsonData(rankings)
			if err != nil {
				return Model{}, err
			}

			existingProvider := GetByTenantIdAndResourceNameProvider(tenantId, "rankings")(p.db)
			existing, err := existingProvider()
			if err == nil {
				existing.ResourceData = resourceData
				if err := UpdateConfiguration(p.db, existing); err != nil {
					return Model{}, err
				}
				m, err := Make(existing)
				if err != nil {
					return Model{}, err
				}
				if err := mb.Put(EventTopicConfigurationStatus, CreateRankingsStatusEventProvider(tenantId, EventTypeRankingsUpdated, rankingsId)); err != nil {
					return Model{}, err
				}
				return m, nil
			} else if errors.Is(err, gorm.ErrRecordNotFound) {
				entity := Entity{
					ID:           uuid.New(),
					TenantId:     tenantId,
					ResourceName: "rankings",
					ResourceData: resourceData,
				}
				if err := CreateConfiguration(p.db, entity); err != nil {
					return Model{}, err
				}
				m, err := Make(entity)
				if err != nil {
					return Model{}, err
				}
				if err := mb.Put(EventTopicConfigurationStatus, CreateRankingsStatusEventProvider(tenantId, EventTypeRankingsCreated, rankingsId)); err != nil {
					return Model{}, err
				}
				return m, nil
			}
			return Model{}, err
		}
	}
}

// CreateRankingsAndEmit creates the rankings configuration and emits events
func (p *ProcessorImpl) CreateRankingsAndEmit(tenantId uuid.UUID, rankings map[string]interface{}) (Model, error) {
	ctx, err := p.tenantCtx(tenantId)
	if err != nil {
		return Model{}, err
	}
	var result Model
	txErr := database.ExecuteTransaction(p.db.WithContext(ctx), func(tx *gorm.DB) error {
		var err error
		result, err = message.EmitWithResult[Model, uuid.UUID](outbox.EmitProvider(p.l, ctx, tx))(func(mb *message.Buffer) func(uuid.UUID) (Model, error) {
			return func(tenantId uuid.UUID) (Model, error) {
				return NewProcessor(p.l, ctx, tx).CreateRankings(mb)(tenantId)(rankings)
			}
		})(tenantId)
		return err
	})
	return result, txErr
}

// UpdateRankings updates the existing rankings configuration
func (p *ProcessorImpl) UpdateRankings(mb *message.Buffer) func(tenantId uuid.UUID) func(rankings map[string]interface{}) (Model, error) {
	return func(tenantId uuid.UUID) func(rankings map[string]interface{}) (Model, error) {
		return func(rankings map[string]interface{}) (Model, error) {
			existingProvider := GetByTenantIdAndResourceNameProvider(tenantId, "rankings")(p.db)
			existing, err := existingProvider()
			if err != nil {
				return Model{}, err
			}

			rankingsId := ""
			if id, ok := rankings["id"].(string); ok {
				rankingsId = id
			}

			resourceData, err := CreateSingleRankingsJsonData(rankings)
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
			if err := mb.Put(EventTopicConfigurationStatus, CreateRankingsStatusEventProvider(tenantId, EventTypeRankingsUpdated, rankingsId)); err != nil {
				return Model{}, err
			}
			return m, nil
		}
	}
}

// UpdateRankingsAndEmit updates the rankings configuration and emits events
func (p *ProcessorImpl) UpdateRankingsAndEmit(tenantId uuid.UUID, rankings map[string]interface{}) (Model, error) {
	ctx, err := p.tenantCtx(tenantId)
	if err != nil {
		return Model{}, err
	}
	var result Model
	txErr := database.ExecuteTransaction(p.db.WithContext(ctx), func(tx *gorm.DB) error {
		var err error
		result, err = message.EmitWithResult[Model, uuid.UUID](outbox.EmitProvider(p.l, ctx, tx))(func(mb *message.Buffer) func(uuid.UUID) (Model, error) {
			return func(tenantId uuid.UUID) (Model, error) {
				return NewProcessor(p.l, ctx, tx).UpdateRankings(mb)(tenantId)(rankings)
			}
		})(tenantId)
		return err
	})
	return result, txErr
}

// DeleteRankings deletes the rankings configuration
func (p *ProcessorImpl) DeleteRankings(mb *message.Buffer) func(tenantId uuid.UUID) error {
	return func(tenantId uuid.UUID) error {
		if _, err := DeleteConfigurationByResourceName(p.db, tenantId, "rankings"); err != nil {
			return err
		}
		return mb.Put(EventTopicConfigurationStatus, CreateRankingsStatusEventProvider(tenantId, EventTypeRankingsDeleted, ""))
	}
}

// DeleteRankingsAndEmit deletes the rankings configuration and emits events
func (p *ProcessorImpl) DeleteRankingsAndEmit(tenantId uuid.UUID) error {
	ctx, err := p.tenantCtx(tenantId)
	if err != nil {
		return err
	}
	return database.ExecuteTransaction(p.db.WithContext(ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, ctx, tx))(func(mb *message.Buffer) error {
			return NewProcessor(p.l, ctx, tx).DeleteRankings(mb)(tenantId)
		})
	})
}

// GetRankings gets the rankings configuration for a tenant
func (p *ProcessorImpl) GetRankings(tenantId uuid.UUID) (map[string]interface{}, error) {
	return p.RankingsProvider(tenantId)()
}

// RankingsProvider returns a provider for the rankings configuration
func (p *ProcessorImpl) RankingsProvider(tenantId uuid.UUID) model.Provider[map[string]interface{}] {
	return GetRankingsProvider(tenantId)(p.db)
}

// CreateKiteConfig creates (or replaces) the tenant's kite-configs configuration
func (p *ProcessorImpl) CreateKiteConfig(mb *message.Buffer) func(tenantId uuid.UUID) func(cfg map[string]interface{}) (Model, error) {
	return func(tenantId uuid.UUID) func(cfg map[string]interface{}) (Model, error) {
		return func(cfg map[string]interface{}) (Model, error) {
			kiteConfigId := ""
			if id, ok := cfg["id"].(string); ok {
				kiteConfigId = id
			}

			resourceData, err := CreateSingleKiteConfigJsonData(cfg)
			if err != nil {
				return Model{}, err
			}

			existingProvider := GetByTenantIdAndResourceNameProvider(tenantId, "kite-configs")(p.db)
			existing, err := existingProvider()
			if err == nil {
				existing.ResourceData = resourceData
				if err := UpdateConfiguration(p.db, existing); err != nil {
					return Model{}, err
				}
				m, err := Make(existing)
				if err != nil {
					return Model{}, err
				}
				if err := mb.Put(EventTopicConfigurationStatus, CreateKiteConfigStatusEventProvider(tenantId, EventTypeKiteConfigUpdated, kiteConfigId)); err != nil {
					return Model{}, err
				}
				return m, nil
			} else if errors.Is(err, gorm.ErrRecordNotFound) {
				entity := Entity{
					ID:           uuid.New(),
					TenantId:     tenantId,
					ResourceName: "kite-configs",
					ResourceData: resourceData,
				}
				if err := CreateConfiguration(p.db, entity); err != nil {
					return Model{}, err
				}
				m, err := Make(entity)
				if err != nil {
					return Model{}, err
				}
				if err := mb.Put(EventTopicConfigurationStatus, CreateKiteConfigStatusEventProvider(tenantId, EventTypeKiteConfigCreated, kiteConfigId)); err != nil {
					return Model{}, err
				}
				return m, nil
			}
			return Model{}, err
		}
	}
}

// CreateKiteConfigAndEmit creates the kite-configs configuration and emits events
func (p *ProcessorImpl) CreateKiteConfigAndEmit(tenantId uuid.UUID, cfg map[string]interface{}) (Model, error) {
	ctx, err := p.tenantCtx(tenantId)
	if err != nil {
		return Model{}, err
	}
	var result Model
	txErr := database.ExecuteTransaction(p.db.WithContext(ctx), func(tx *gorm.DB) error {
		var err error
		result, err = message.EmitWithResult[Model, uuid.UUID](outbox.EmitProvider(p.l, ctx, tx))(func(mb *message.Buffer) func(uuid.UUID) (Model, error) {
			return func(tenantId uuid.UUID) (Model, error) {
				return NewProcessor(p.l, ctx, tx).CreateKiteConfig(mb)(tenantId)(cfg)
			}
		})(tenantId)
		return err
	})
	return result, txErr
}

// UpdateKiteConfig updates the existing kite-configs configuration
func (p *ProcessorImpl) UpdateKiteConfig(mb *message.Buffer) func(tenantId uuid.UUID) func(cfg map[string]interface{}) (Model, error) {
	return func(tenantId uuid.UUID) func(cfg map[string]interface{}) (Model, error) {
		return func(cfg map[string]interface{}) (Model, error) {
			existingProvider := GetByTenantIdAndResourceNameProvider(tenantId, "kite-configs")(p.db)
			existing, err := existingProvider()
			if err != nil {
				return Model{}, err
			}

			kiteConfigId := ""
			if id, ok := cfg["id"].(string); ok {
				kiteConfigId = id
			}

			resourceData, err := CreateSingleKiteConfigJsonData(cfg)
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
			if err := mb.Put(EventTopicConfigurationStatus, CreateKiteConfigStatusEventProvider(tenantId, EventTypeKiteConfigUpdated, kiteConfigId)); err != nil {
				return Model{}, err
			}
			return m, nil
		}
	}
}

// UpdateKiteConfigAndEmit updates the kite-configs configuration and emits events
func (p *ProcessorImpl) UpdateKiteConfigAndEmit(tenantId uuid.UUID, cfg map[string]interface{}) (Model, error) {
	ctx, err := p.tenantCtx(tenantId)
	if err != nil {
		return Model{}, err
	}
	var result Model
	txErr := database.ExecuteTransaction(p.db.WithContext(ctx), func(tx *gorm.DB) error {
		var err error
		result, err = message.EmitWithResult[Model, uuid.UUID](outbox.EmitProvider(p.l, ctx, tx))(func(mb *message.Buffer) func(uuid.UUID) (Model, error) {
			return func(tenantId uuid.UUID) (Model, error) {
				return NewProcessor(p.l, ctx, tx).UpdateKiteConfig(mb)(tenantId)(cfg)
			}
		})(tenantId)
		return err
	})
	return result, txErr
}

// DeleteKiteConfig deletes the kite-configs configuration
func (p *ProcessorImpl) DeleteKiteConfig(mb *message.Buffer) func(tenantId uuid.UUID) error {
	return func(tenantId uuid.UUID) error {
		if _, err := DeleteConfigurationByResourceName(p.db, tenantId, "kite-configs"); err != nil {
			return err
		}
		return mb.Put(EventTopicConfigurationStatus, CreateKiteConfigStatusEventProvider(tenantId, EventTypeKiteConfigDeleted, ""))
	}
}

// DeleteKiteConfigAndEmit deletes the kite-configs configuration and emits events
func (p *ProcessorImpl) DeleteKiteConfigAndEmit(tenantId uuid.UUID) error {
	ctx, err := p.tenantCtx(tenantId)
	if err != nil {
		return err
	}
	return database.ExecuteTransaction(p.db.WithContext(ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, ctx, tx))(func(mb *message.Buffer) error {
			return NewProcessor(p.l, ctx, tx).DeleteKiteConfig(mb)(tenantId)
		})
	})
}

// GetKiteConfig gets the kite-configs configuration for a tenant
func (p *ProcessorImpl) GetKiteConfig(tenantId uuid.UUID) (map[string]interface{}, error) {
	return p.KiteConfigProvider(tenantId)()
}

// KiteConfigProvider returns a provider for the kite-configs configuration
func (p *ProcessorImpl) KiteConfigProvider(tenantId uuid.UUID) model.Provider[map[string]interface{}] {
	return GetKiteConfigProvider(tenantId)(p.db)
}
