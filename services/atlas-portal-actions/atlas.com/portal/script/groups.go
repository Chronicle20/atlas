package script

import (
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"
)

func InitSeedResource(_ jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			src := seeder.NewFilesystemCatalogSource("SEED_CATALOG_ROOT", "./deploy/seed")
			seeder.RegisterRoutes(router, db, l, src, seeder.Group{
				Name:      "portal-actions",
				URLPrefix: "/portals/scripts",
				Subdomains: []seeder.SubdomainAny{
					seeder.AdaptSubdomain[jsonPortalScript, PortalScript](PortalSubdomain{}),
				},
			})
		}
	}
}
