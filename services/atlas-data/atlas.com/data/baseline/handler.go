package baseline

import (
	"errors"
	"fmt"
	"net/http"

	"atlas-data/rest"
	minio "atlas-data/storage/minio"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
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
			r.HandleFunc("/publish", rest.RegisterInputHandler[PublishInputModel](l)(si)("baseline_publish", publishInner(db, mc, l))).Methods(http.MethodPost)
			r.HandleFunc("/restore", rest.RegisterInputHandler[RestoreInputModel](l)(si)("baseline_restore", restoreInner(db, mc, l))).Methods(http.MethodPost)
		}
	}
}

func publishInner(db *gorm.DB, mc *minio.Client, _ logrus.FieldLogger) func(d *rest.HandlerDependency, c *rest.HandlerContext, input PublishInputModel) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input PublishInputModel) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if mc == nil {
				http.Error(w, "minio unavailable", http.StatusServiceUnavailable)
				return
			}
			if r.Header.Get("X-Atlas-Operator") != "1" {
				http.Error(w, "operator required", http.StatusForbidden)
				return
			}
			sum, err := (Publisher{DB: db, MC: mc, L: d.Logger()}).Publish(r.Context(), input.Region, input.MajorVersion, input.MinorVersion)
			if err != nil {
				d.Logger().WithError(err).Errorf("baseline publish failed")
				http.Error(w, fmt.Sprintf("publish failed: %s", err.Error()), http.StatusInternalServerError)
				return
			}
			out := PublishOutputModel{
				Id:     PublishOutputId(input.Region, input.MajorVersion, input.MinorVersion),
				Sha256: sum,
			}
			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			w.Header().Set("Content-Type", "application/vnd.api+json")
			w.WriteHeader(http.StatusAccepted)
			server.MarshalResponse[PublishOutputModel](d.Logger())(w)(c.ServerInformation())(queryParams)(out)
		}
	}
}

func restoreInner(db *gorm.DB, mc *minio.Client, _ logrus.FieldLogger) func(d *rest.HandlerDependency, c *rest.HandlerContext, input RestoreInputModel) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input RestoreInputModel) http.HandlerFunc {
		return func(w http.ResponseWriter, req *http.Request) {
			if mc == nil {
				http.Error(w, "minio unavailable", http.StatusServiceUnavailable)
				return
			}
			if req.Header.Get("X-Atlas-Operator") != "1" {
				http.Error(w, "operator required", http.StatusForbidden)
				return
			}
			if err := (Restorer{DB: db, MC: mc, L: d.Logger()}).Restore(req.Context(), input.Region, input.MajorVersion, input.MinorVersion, input.TenantID); err != nil {
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
