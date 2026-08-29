package seed

import (
	"atlas-maker/reagent"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"
)

// InitResource mounts the seed-catalog routes for atlas-maker's reference
// data. Reagents are read-only over the wire, so the seed catalog is the only
// way a tenant retunes them.
func InitResource(_ jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			src := seeder.NewFilesystemCatalogSource("SEED_CATALOG_ROOT", "./deploy/seed")
			seeder.RegisterRoutes(router, db, l, src, seeder.Group{
				Name:      "reagents",
				URLPrefix: "/reagents",
				Subdomains: []seeder.SubdomainAny{
					seeder.AdaptSubdomain[reagent.ReagentAttributes, reagent.Model](reagent.Subdomain{}),
				},
			})
		}
	}
}
