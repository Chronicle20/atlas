package rest

import (
	"encoding/json"
	"fmt"
	"net/http"

	"atlas-data/rest"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

// InitResource installs POST/GET /data/process. When jc is nil (k8s unavailable)
// the create handler responds 503.
func InitResource(jc *JobCreator) func(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(si jsonapi.ServerInformation) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			r := router.PathPrefix("/data").Subrouter()
			r.HandleFunc("/process", rest.RegisterHandler(l)(si)("process_create", processCreate(jc))).Methods(http.MethodPost)
			r.HandleFunc("/process", rest.RegisterHandler(l)(si)("process_status", processStatus())).Methods(http.MethodGet)
		}
	}
}

func processCreate(jc *JobCreator) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if jc == nil {
				http.Error(w, "k8s unavailable", http.StatusServiceUnavailable)
				return
			}
			t := tenant.MustFromContext(d.Context())
			scope := r.URL.Query().Get("scope")
			switch scope {
			case "", "tenant":
				scope = "tenants/" + t.Id().String()
			case "shared":
				if r.Header.Get("X-Atlas-Operator") != "1" {
					http.Error(w, "operator required", http.StatusForbidden)
					return
				}
			default:
				http.Error(w, "invalid scope", http.StatusBadRequest)
				return
			}
			name, err := jc.Create(
				r.Context(),
				scope,
				t.Region(),
				int(t.MajorVersion()),
				int(t.MinorVersion()),
				t.Id().String(),
				r.Header.Get("traceparent"),
			)
			if err != nil {
				http.Error(w, fmt.Sprintf("create job: %v", err), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jobName": name,
				"scope":   scope,
				"version": fmt.Sprintf("%d.%d", t.MajorVersion(), t.MinorVersion()),
			})
		}
	}
}

func processStatus() func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// TODO Task 12 follow-up: query active jobs by label selector and
			// surface their statuses keyed by (scope, region, version).
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"jobs": []any{}})
		}
	}
}
