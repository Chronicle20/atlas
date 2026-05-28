package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"atlas-data/rest"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InitResource installs POST/GET /data/process. When jc is nil (k8s unavailable)
// the create handler responds 503.
func InitResource(jc *JobCreator) func(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(si jsonapi.ServerInformation) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			r := router.PathPrefix("/data").Subrouter()
			r.HandleFunc("/process", rest.RegisterHandler(l)(si)("process_create", processCreate(jc))).Methods(http.MethodPost)
			r.HandleFunc("/process", rest.RegisterHandler(l)(si)("process_status", processStatus(jc))).Methods(http.MethodGet)
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
				t.MajorVersion(),
				t.MinorVersion(),
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

// processStatusJob is the JSON shape returned per ingest Job from
// processStatus.
type processStatusJob struct {
	Name      string `json:"name"`
	Scope     string `json:"scope"`
	Region    string `json:"region"`
	Version   string `json:"version"`
	Tenant    string `json:"tenant,omitempty"`
	Active    int32  `json:"active"`
	Succeeded int32  `json:"succeeded"`
	Failed    int32  `json:"failed"`
	StartTime string `json:"startTime,omitempty"`
}

func processStatus(jc *JobCreator) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if jc == nil || jc.K8s == nil {
				http.Error(w, "k8s unavailable", http.StatusServiceUnavailable)
				return
			}
			list, err := jc.K8s.BatchV1().Jobs(jc.Namespace).List(r.Context(), metav1.ListOptions{
				LabelSelector: labelIngest + "=true",
			})
			if err != nil {
				http.Error(w, fmt.Sprintf("list jobs: %v", err), http.StatusInternalServerError)
				return
			}
			out := make([]processStatusJob, 0, len(list.Items))
			for _, j := range list.Items {
				e := processStatusJob{
					Name:      j.Name,
					Scope:     j.Labels["scope"],
					Region:    j.Labels["region"],
					Version:   j.Labels["version"],
					Tenant:    j.Labels["tenant"],
					Active:    j.Status.Active,
					Succeeded: j.Status.Succeeded,
					Failed:    j.Status.Failed,
				}
				if j.Status.StartTime != nil {
					e.StartTime = j.Status.StartTime.UTC().Format(time.RFC3339)
				}
				out = append(out, e)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"jobs": out})
		}
	}
}
