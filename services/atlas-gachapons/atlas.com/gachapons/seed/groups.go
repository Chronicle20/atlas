package seed

import (
	"atlas-gachapons/gachapon"
	"atlas-gachapons/global"
	"atlas-gachapons/item"

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
				Name:      "gachapons",
				URLPrefix: "/gachapons",
				Subdomains: []seeder.SubdomainAny{
					seeder.AdaptSubdomain[gachapon.GachaponAttributes, gachapon.Model](gachapon.Subdomain{}),
					seeder.AdaptSubdomain[gachapon.GachaponAttributes, item.Model](item.Subdomain{}),
					seeder.AdaptSubdomain[global.GlobalPoolAttributes, global.Model](global.Subdomain{}),
				},
			})
		}
	}
}
