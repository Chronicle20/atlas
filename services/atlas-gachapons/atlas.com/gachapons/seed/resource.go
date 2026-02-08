package seed

import (
	"atlas-gachapons/rest"
	"context"
	"net/http"

	"github.com/Chronicle20/atlas-rest/server"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerHandler := rest.RegisterHandler(l)(db)(si)
			router.HandleFunc("/gachapons/seed", registerHandler("seed_gachapons", handleSeed)).Methods(http.MethodPost)
		}
	}
}

func handleSeed(d *rest.HandlerDependency, _ *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t := tenant.MustFromContext(d.Context())
		l := d.Logger()
		db := d.DB()

		go func() {
			bgCtx := tenant.WithContext(context.Background(), t)

			result, err := NewProcessor(l, bgCtx, db).Seed()
			if err != nil {
				l.WithError(err).Errorf("Seeding gachapons for tenant [%s].", t.Id())
				return
			}

			l.Infof("Seed complete for tenant [%s]: gachapons=%d, items=%d, global=%d",
				t.Id(),
				result.Gachapons.CreatedCount,
				result.Items.CreatedCount,
				result.GlobalItems.CreatedCount)
		}()

		w.WriteHeader(http.StatusAccepted)
	}
}
