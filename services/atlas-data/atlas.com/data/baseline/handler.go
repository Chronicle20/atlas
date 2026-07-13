package baseline

import (
	"errors"
	"net/http"

	"atlas-data/rest"
	minio "atlas-data/storage/minio"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// InitResource installs POST /data/baseline/publish, POST /data/baseline/restore,
// and GET /data/baselines.
func InitResource(db *gorm.DB, mc *minio.Client) func(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(si jsonapi.ServerInformation) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			r := router.PathPrefix("/data/baseline").Subrouter()
			r.HandleFunc("/publish", rest.RegisterInputHandler[PublishInputModel](l)(si)("baseline_publish", publishInner(db, mc, l))).Methods(http.MethodPost)
			r.HandleFunc("/restore", rest.RegisterInputHandler[RestoreInputModel](l)(si)("baseline_restore", restoreInner(db, mc, l))).Methods(http.MethodPost)
			// Plural collection route deliberately outside the /data/baseline
			// subrouter: GET /data/baselines lists published canonical baselines.
			router.HandleFunc("/data/baselines", rest.RegisterHandler(l)(si)("baselines_list", listInner(mc))).Methods(http.MethodGet)
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
				server.WriteErrorResponse(d.Logger())(w)(err)
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

// listInner serves GET /data/baselines. Gate order matches publishInner:
// nil-mc 503 first, then the operator 403, then the listing. The ParseTenant
// middleware runs on the route (all RegisterHandler routes get it) but the
// handler never reads the tenant — the nil-UUID synthetic tenant the UI sends
// is accepted and ignored.
func listInner(mc *minio.Client) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
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
			items, err := (Lister{MC: mc, Bucket: mc.Cfg().BucketCanonical, L: d.Logger()}).List(r.Context())
			if err != nil {
				d.Logger().WithError(err).Errorf("baseline list failed")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}
			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			w.Header().Set("Content-Type", "application/vnd.api+json")
			server.MarshalResponse[[]ListItemModel](d.Logger())(w)(c.ServerInformation())(queryParams)(items)
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
				if errors.Is(err, ErrSchemaMismatch) || errors.Is(err, ErrShaMismatch) {
					http.Error(w, err.Error(), http.StatusUnprocessableEntity)
					return
				}
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}
			w.WriteHeader(http.StatusAccepted)
		}
	}
}
