package seed

import (
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"
)

// InitResource registers POST /<prefix>/seed and GET /<prefix>/seed/status
// for each transport configuration resource. Register this BEFORE
// configuration.RegisterRoutes so the literal seed paths are matched ahead
// of the /tenants/{tenantId}/... patterns.
func InitResource(db *gorm.DB) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		// The three transport resources are version-agnostic: one shared
		// set of files applies to every tenant regardless of region or
		// client version, so the source resolves deploy/seed/shared/all
		// in addition to the tenant's version root.
		src := seeder.NewFilesystemCatalogSourceWithShared("SEED_CATALOG_ROOT", "./deploy/seed", "shared/all")
		for _, g := range Groups(l) {
			seeder.RegisterRoutes(router, db, l, src, g)
		}
	}
}
