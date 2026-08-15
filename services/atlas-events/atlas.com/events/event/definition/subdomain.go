package definition

import (
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// compile-time assertion
var _ seeder.Subdomain[RestModel, Model] = DefinitionSubdomain{}

// DefinitionSubdomain implements seeder.Subdomain for event definition seed
// data (FR-D7). Every seeded definition ships disabled — turning one on is an
// administrative action taken through PATCH after the fact.
type DefinitionSubdomain struct{}

func (DefinitionSubdomain) Name() string { return "definition.event" }
func (DefinitionSubdomain) Path() string { return "events/definitions" }
func (DefinitionSubdomain) Type() string { return "event-definition" }
func (DefinitionSubdomain) EntityIDPattern() *regexp.Regexp {
	return regexp.MustCompile(`^event-(.+)\.json$`)
}

func (DefinitionSubdomain) DeleteAllForTenant(db *gorm.DB) (int64, error) {
	result := db.Unscoped().Where("1 = 1").Delete(&Entity{})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func (DefinitionSubdomain) Decode(payload []byte) (RestModel, error) {
	var rm RestModel
	if err := seeder.DecodeAttributes(payload, &rm); err != nil {
		return RestModel{}, fmt.Errorf("definition.event: decode: %w", err)
	}
	return rm, nil
}

func (DefinitionSubdomain) Build(t tenant.Model, _ string, rm RestModel) ([]Model, error) {
	m, err := Extract(rm)
	if err != nil {
		return nil, fmt.Errorf("definition.event: build %s: %w", rm.Type, err)
	}
	return []Model{m}, nil
}

func (DefinitionSubdomain) BulkCreate(db *gorm.DB, models []Model) error {
	if len(models) == 0 {
		return nil
	}

	tenantId := extractDefinitionTenantId(db)
	for _, m := range models {
		entity, err := ToEntity(m, tenantId)
		if err != nil {
			return fmt.Errorf("definition.event: to entity %s: %w", m.Type(), err)
		}
		entity.ID = uuid.New()
		if result := db.Create(&entity); result.Error != nil {
			return fmt.Errorf("definition.event: create %s: %w", m.Type(), result.Error)
		}
	}
	return nil
}

func (DefinitionSubdomain) Count(db *gorm.DB) (int64, *time.Time, error) {
	var count int64
	if err := db.Model(&Entity{}).Count(&count).Error; err != nil {
		return 0, nil, err
	}
	return count, nil, nil
}

// extractDefinitionTenantId retrieves the tenant ID embedded in the GORM context.
func extractDefinitionTenantId(db *gorm.DB) uuid.UUID {
	if db.Statement != nil && db.Statement.Context != nil {
		t := tenant.MustFromContext(db.Statement.Context)
		return t.Id()
	}
	return uuid.Nil
}

// InitSeedResource registers the seeder catalog routes for event definitions.
// Event definitions are game-rule data, not client-version-specific content,
// so this resolves through the "shared/all" root (deploy/seed/shared/all/…)
// via NewFilesystemCatalogSourceWithShared rather than the per-region/version
// root every party-quests-style group uses — see atlas-tenants'
// configuration/seed for the same shared-root pattern.
func InitSeedResource(_ jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			src := seeder.NewFilesystemCatalogSourceWithShared("SEED_CATALOG_ROOT", "./deploy/seed", "shared/all")
			seeder.RegisterRoutes(router, db, l, src, seeder.Group{
				Name:      "events",
				URLPrefix: "/events/definitions",
				Subdomains: []seeder.SubdomainAny{
					seeder.AdaptSubdomain[RestModel, Model](DefinitionSubdomain{}),
				},
			})
		}
	}
}
