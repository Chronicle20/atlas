package seed

import (
	continentdrop "atlas-drops-information/continent/drop"
	monsterdrop "atlas-drops-information/monster/drop"
	reactordrop "atlas-drops-information/reactor/drop"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitResource(_ jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			src := seeder.NewFilesystemCatalogSource("SEED_CATALOG_ROOT", "./deploy/seed")
			seeder.RegisterRoutes(router, db, l, src, seeder.Group{
				Name:      "drops",
				URLPrefix: "/drops",
				Subdomains: []seeder.SubdomainAny{
					seeder.AdaptSubdomain[monsterdrop.JSONModel, monsterdrop.Model](monsterdrop.Subdomain{}),
					seeder.AdaptSubdomain[continentdrop.JSONModel, continentdrop.Model](continentdrop.Subdomain{}),
					seeder.AdaptSubdomain[reactordrop.JSONModel, reactordrop.Model](reactordrop.Subdomain{}),
				},
			})
		}
	}
}
