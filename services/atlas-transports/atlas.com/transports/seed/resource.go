package seed

import (
	"atlas-transports/rest"
	"encoding/json"
	"net/http"

	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

// InitResource registers the seed routes with the router
func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(r *mux.Router, l logrus.FieldLogger) {
		registerHandler := rest.RegisterHandler(l)(si)
		r.HandleFunc("/transports/routes/seed", registerHandler("seed_routes", handleSeedRoutes)).Methods(http.MethodPost)
	}
}

// handleSeedRoutes handles the POST /transports/routes/seed endpoint
func handleSeedRoutes(d *rest.HandlerDependency, _ *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := NewProcessor(d.Logger(), d.Context()).Seed()
		if err != nil {
			d.Logger().WithError(err).Errorf("Seeding routes.")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(result)
	}
}
