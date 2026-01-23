package seed

import (
	"atlas-drops-information/rest"
	"context"
	"net/http"

	"github.com/Chronicle20/atlas-rest/server"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerHandler := rest.RegisterHandler(l)(db)(si)
			router.HandleFunc("/drops/seed", registerHandler("seed_drops", handleSeedDrops)).Methods(http.MethodPost)
		}
	}
}

func handleSeedDrops(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract tenant before spawning goroutine (request context will be cancelled after response)
		t := tenant.MustFromContext(d.Context())
		l := d.Logger()
		db := d.DB()

		// Spawn background goroutine for processing
		go func() {
			// Create new context with tenant for background processing
			bgCtx := tenant.WithContext(context.Background(), t)

			result, err := NewProcessor(l, bgCtx, db).Seed()
			if err != nil {
				l.WithError(err).Errorf("Seeding drops for tenant [%s].", t.Id())
				return
			}

			l.Infof("Seed complete for tenant [%s]: monster=%d/%d, continent=%d/%d, reactor=%d/%d",
				t.Id(),
				result.MonsterDrops.CreatedCount, result.MonsterDrops.DeletedCount,
				result.ContinentDrops.CreatedCount, result.ContinentDrops.DeletedCount,
				result.ReactorDrops.CreatedCount, result.ReactorDrops.DeletedCount)
		}()

		// Return immediately with 202 Accepted
		w.WriteHeader(http.StatusAccepted)
	}
}
