package tenantpurge

import (
	"errors"
	"net/http"

	"atlas-data/rest"
	minio "atlas-data/storage/minio"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// InitResource installs DELETE /data/tenants/{id}.
func InitResource(db *gorm.DB, mc *minio.Client) func(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(si jsonapi.ServerInformation) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			r := router.PathPrefix("/data/tenants").Subrouter()
			r.HandleFunc("/{id}", rest.RegisterHandler(l)(si)("tenant_purge", purgeInner(db, mc))).Methods(http.MethodDelete)
		}
	}
}

func purgeInner(db *gorm.DB, mc *minio.Client) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if mc == nil {
				http.Error(w, "minio unavailable", http.StatusServiceUnavailable)
				return
			}
			if r.Header.Get("X-Atlas-Operator") != "1" {
				http.Error(w, "operator required", http.StatusForbidden)
				return
			}
			idStr := mux.Vars(r)["id"]
			id, err := uuid.Parse(idStr)
			if err != nil {
				http.Error(w, "bad tenant id", http.StatusBadRequest)
				return
			}
			if err := Purge(r.Context(), d.Logger(), db, mc, id); err != nil {
				if errors.Is(err, ErrCanonicalRefused) {
					http.Error(w, err.Error(), http.StatusForbidden)
					return
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusAccepted)
		}
	}
}
