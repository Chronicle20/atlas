package rest

import (
	"atlas-data/ingestrun"
	"atlas-data/rest"
	"atlas-data/wzinput"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
	redis "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// InitResource installs POST/GET /data/process.
//
// jc and regs are independently nil-able: no jc makes the create handler
// respond 503 and disables the status handler's live-Job cross-check; no regs
// degrades the status handler to phase "none" (PRD FR-4.5).
func InitResource(jc *JobCreator, regs *IngestRegistries) func(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(si jsonapi.ServerInformation) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			r := router.PathPrefix("/data").Subrouter()
			r.HandleFunc("/process", rest.RegisterHandler(l)(si)("process_create", processCreate(jc))).Methods(http.MethodPost)
			r.HandleFunc("/process", rest.RegisterHandler(l)(si)("process_status", processStatus(jc, regs))).Methods(http.MethodGet)
		}
	}
}

// writeScopeError maps a wzinput.ResolveScope failure onto its HTTP status.
// Shared by both verbs so the operator gate on scope=shared cannot drift.
func writeScopeError(w http.ResponseWriter, err error) {
	if errors.Is(err, wzinput.ErrSharedRequiresOperator) {
		http.Error(w, "operator required", http.StatusForbidden)
		return
	}
	http.Error(w, "invalid scope", http.StatusBadRequest)
}

func processCreate(jc *JobCreator) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if jc == nil {
				http.Error(w, "k8s unavailable", http.StatusServiceUnavailable)
				return
			}
			t := tenant.MustFromContext(d.Context())
			sc, serr := wzinput.ResolveScope(r, t)
			if serr != nil {
				writeScopeError(w, serr)
				return
			}
			name, err := jc.Create(
				r.Context(),
				sc.Key,
				t.Region(),
				t.MajorVersion(),
				t.MinorVersion(),
				t.Id().String(),
				r.Header.Get("traceparent"),
			)
			if err != nil {
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jobName": name,
				"scope":   sc.Key,
				"version": fmt.Sprintf("%d.%d", t.MajorVersion(), t.MinorVersion()),
			})
		}
	}
}

// processStatus returns the ingest run for the caller's resolved scope.
//
// Redis is the system of record; Kubernetes is a corroborating second opinion
// used only to demote a stale `running` to `unknown`. That direction is
// deliberate: the Job object is the thing that disappears
// (ttlSecondsAfterFinished, Watchdog deletion), so it cannot be the source of
// truth for a feature whose whole point is surviving its disappearance.
func processStatus(jc *JobCreator, regs *IngestRegistries) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			t := tenant.MustFromContext(d.Context())
			sc, serr := wzinput.ResolveScope(r, t)
			if serr != nil {
				writeScopeError(w, serr)
				return
			}
			suffix := ingestrun.KeySuffix(sc.Key, t.Region(), t.MajorVersion(), t.MinorVersion())
			id := suffix

			var rec ingestrun.Record
			phase := ingestrun.PhaseNone
			if regs != nil && regs.Run != nil {
				stored, err := regs.Run.Get(r.Context(), env.Self(), suffix+ingestrun.RunKeySuffix)
				switch {
				case err == nil:
					rec, phase = stored, stored.Phase
				case errors.Is(err, redis.ErrNotFound):
					// "No ingest has been run" is a valid, actionable answer.
				default:
					d.Logger().WithError(err).Errorf("Unable to read ingest run record %s.", suffix)
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
			}

			if phase == ingestrun.PhaseRunning {
				phase = corroborateRunning(r.Context(), jc, regs, suffix, sc.Key, t.Region(), t.MajorVersion(), t.MinorVersion())
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			res := toIngestRunRestModel(rec, phase, id)
			server.MarshalResponse[IngestRunRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	}
}

// corroborateRunning decides whether a stored `running` phase is still
// believable. It never rewrites the stored record: `unknown` is computed at
// read time, so a record that later proves alive (a slow Job list, a restarted
// pod) recovers on the next poll with no repair path.
func corroborateRunning(ctx context.Context, jc *JobCreator, regs *IngestRegistries, suffix, scope, region string, major, minor uint16) ingestrun.Phase {
	// A heartbeat inside the Watchdog's staleness window means the pod is
	// alive; believe it regardless of what Kubernetes says. This covers the
	// window between Job creation and pod scheduling, and any Job-list hiccup.
	if regs != nil && regs.Job != nil {
		if ts, err := regs.Job.Get(ctx, env.Self(), suffix+ingestrun.HeartbeatKeySuffix); err == nil && ts != "" {
			if hb, perr := time.Parse(time.RFC3339, ts); perr == nil {
				if time.Since(hb) < time.Duration(DefaultWatchdogTimeoutSecs)*time.Second {
					return ingestrun.PhaseRunning
				}
			}
		}
	}
	if jc == nil || jc.K8s == nil {
		return ingestrun.PhaseUnknown
	}
	// Narrowed server-side to this triple rather than listing the namespace and
	// filtering client-side. Note the selector uses the SANITIZED scope,
	// matching renderJob's label; the raw scope only ever appears in Redis keys.
	selector := fmt.Sprintf("%s=true,scope=%s,region=%s,version=%d.%d",
		labelIngest, sanitizeLabel(scope), region, major, minor)
	list, err := jc.K8s.BatchV1().Jobs(jc.Namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		// A failed list is not evidence of absence — keep the stored phase.
		return ingestrun.PhaseRunning
	}
	for i := range list.Items {
		j := &list.Items[i]
		if j.Status.Succeeded == 0 && !jobFailed(j) {
			return ingestrun.PhaseRunning
		}
	}
	return ingestrun.PhaseUnknown
}
