package baseline

import (
	"encoding/json"
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

// InitResource installs POST /data/baseline/publish and /data/baseline/restore.
func InitResource(db *gorm.DB, mc *minio.Client) func(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(si jsonapi.ServerInformation) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			r := router.PathPrefix("/data/baseline").Subrouter()
			r.HandleFunc("/publish", rest.RegisterHandler(l)(si)("baseline_publish", publishInner(db, mc, l))).Methods(http.MethodPost)
			r.HandleFunc("/restore", rest.RegisterHandler(l)(si)("baseline_restore", restoreInner(db, mc, l))).Methods(http.MethodPost)
		}
	}
}

func publishInner(db *gorm.DB, mc *minio.Client, _ logrus.FieldLogger) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
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
			var body struct {
				Region       string `json:"region"`
				MajorVersion int    `json:"majorVersion"`
				MinorVersion int    `json:"minorVersion"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			sum, err := (Publisher{DB: db, MC: mc, L: d.Logger()}).Publish(r.Context(), body.Region, body.MajorVersion, body.MinorVersion)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]string{"sha256": sum})
		}
	}
}

func restoreInner(db *gorm.DB, mc *minio.Client, _ logrus.FieldLogger) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, req *http.Request) {
			if mc == nil {
				http.Error(w, "minio unavailable", http.StatusServiceUnavailable)
				return
			}
			var body struct {
				Region       string    `json:"region"`
				MajorVersion int       `json:"majorVersion"`
				MinorVersion int       `json:"minorVersion"`
				TenantID     uuid.UUID `json:"tenantId"`
			}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := (Restorer{DB: db, MC: mc, L: d.Logger()}).Restore(req.Context(), body.Region, body.MajorVersion, body.MinorVersion, body.TenantID); err != nil {
				code := http.StatusInternalServerError
				if errors.Is(err, ErrSchemaMismatch) || errors.Is(err, ErrShaMismatch) {
					code = http.StatusUnprocessableEntity
				}
				http.Error(w, err.Error(), code)
				return
			}
			w.WriteHeader(http.StatusAccepted)
		}
	}
}
