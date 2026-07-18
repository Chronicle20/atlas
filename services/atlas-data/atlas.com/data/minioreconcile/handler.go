package minioreconcile

import (
	"atlas-data/rest"
	"errors"
	"net/http"
	"time"

	minio "atlas-data/storage/minio"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

// InitResource installs POST /data/minio/reconcile.
func InitResource(mc *minio.Client) func(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(si jsonapi.ServerInformation) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			r := router.PathPrefix("/data/minio").Subrouter()
			r.HandleFunc("/reconcile",
				rest.RegisterInputHandler[ReconcileInputModel](l)(si)("minio_reconcile", reconcileInner(mcStoreOrNil(mc), time.Now)),
			).Methods(http.MethodPost)
		}
	}
}

// mcStoreOrNil returns a Store for a non-nil client, else nil (handler 503s).
func mcStoreOrNil(mc *minio.Client) Store {
	if mc == nil {
		return nil
	}
	return NewStore(mc)
}

func reconcileInner(store Store, clock func() time.Time) func(d *rest.HandlerDependency, c *rest.HandlerContext, input ReconcileInputModel) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input ReconcileInputModel) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if store == nil {
				http.Error(w, "minio unavailable", http.StatusServiceUnavailable)
				return
			}
			if r.Header.Get("X-Atlas-Operator") != "1" {
				http.Error(w, "operator required", http.StatusForbidden)
				return
			}
			rep, err := Reconcile(r.Context(), d.Logger(), store, input.ToRequest(), clock())
			if err != nil {
				if errors.Is(err, ErrEmptyKeepList) {
					http.Error(w, err.Error(), http.StatusUnprocessableEntity)
					return
				}
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}
			out := toOutput(rep)
			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			w.Header().Set("Content-Type", "application/vnd.api+json")
			w.WriteHeader(http.StatusOK)
			server.MarshalResponse[ReconcileOutputModel](d.Logger())(w)(c.ServerInformation())(queryParams)(out)
		}
	}
}
