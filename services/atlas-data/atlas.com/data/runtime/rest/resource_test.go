package rest

import (
	"atlas-data/ingestrun"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"

	"github.com/google/uuid"
)

type fakeServerInfo struct{}

func (fakeServerInfo) GetBaseURL() string { return "" }
func (fakeServerInfo) GetPrefix() string  { return "" }

const testTenantId = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

func tenantCtx(t *testing.T) context.Context {
	t.Helper()
	tm, err := tenant.Create(uuid.MustParse(testTenantId), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	return tenant.WithContext(context.Background(), tm)
}

func newRegs(t *testing.T) (*IngestRegistries, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	return NewIngestRegistries(rdb), mr
}

// ingestRunAttributes is the decoded JSON:API attributes object.
type ingestRunAttributes struct {
	RunId           string  `json:"runId"`
	JobName         string  `json:"jobName"`
	Scope           string  `json:"scope"`
	Region          string  `json:"region"`
	Version         string  `json:"version"`
	Tenant          string  `json:"tenant"`
	Phase           string  `json:"phase"`
	StartedAt       *string `json:"startedAt"`
	FinishedAt      *string `json:"finishedAt"`
	Reason          *string `json:"reason"`
	WorkersTotal    int     `json:"workersTotal"`
	WorkersComplete int     `json:"workersComplete"`
	Workers         []struct {
		Name       string  `json:"name"`
		State      string  `json:"state"`
		StartedAt  *string `json:"startedAt"`
		FinishedAt *string `json:"finishedAt"`
		Error      *string `json:"error"`
	} `json:"workers"`
}

func doStatus(t *testing.T, jc *JobCreator, regs *IngestRegistries, query string, operator bool) *httptest.ResponseRecorder {
	t.Helper()
	ctx := tenantCtx(t)
	d := server.NewHandlerDependency(logrus.New(), ctx)
	c := server.NewHandlerContext(fakeServerInfo{})
	h := processStatus(jc, regs)(&d, &c)

	req := httptest.NewRequest(http.MethodGet, "/api/data/process"+query, nil).WithContext(ctx)
	if operator {
		req.Header.Set("X-Atlas-Operator", "1")
	}
	rr := httptest.NewRecorder()
	h(rr, req)
	return rr
}

func decodeRun(t *testing.T, rr *httptest.ResponseRecorder) (string, ingestRunAttributes) {
	t.Helper()
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var doc struct {
		Data struct {
			Type       string              `json:"type"`
			Id         string              `json:"id"`
			Attributes ingestRunAttributes `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&doc); err != nil {
		t.Fatalf("decode: %v; body=%s", err, rr.Body.String())
	}
	if doc.Data.Type != "ingestRun" {
		t.Fatalf("resource type = %q, want ingestRun", doc.Data.Type)
	}
	return doc.Data.Id, doc.Data.Attributes
}

func TestProcessStatusNoRecordReportsNone(t *testing.T) {
	regs, _ := newRegs(t)
	rr := doStatus(t, nil, regs, "", false)
	id, attrs := decodeRun(t, rr)
	if attrs.Phase != string(ingestrun.PhaseNone) {
		t.Fatalf("phase = %s, want none", attrs.Phase)
	}
	if attrs.WorkersTotal != 0 || len(attrs.Workers) != 0 {
		t.Fatalf("none record has workers: %+v", attrs)
	}
	want := "tenants/" + testTenantId + ":GMS:83.1"
	if id != want {
		t.Fatalf("id = %q, want %q", id, want)
	}
}

func TestProcessStatusReturnsOnlyTheCallersScope(t *testing.T) {
	ctx := context.Background()
	regs, _ := newRegs(t)

	mine := ingestrun.KeySuffix("tenants/"+testTenantId, "GMS", 83, 1)
	other := ingestrun.KeySuffix("tenants/bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "GMS", 83, 1)
	put := func(suffix, runId string) {
		rec := ingestrun.NewRecord(runId, "job-"+runId, "x", "GMS", "83.1", "", time.Now().UTC(), []string{"STRING"})
		rec = rec.WithPhase(ingestrun.PhaseSucceeded, time.Now().UTC(), "")
		if err := regs.Run.PutWithTTL(ctx, suffix+ingestrun.RunKeySuffix, rec, ingestrun.RecordTTL); err != nil {
			t.Fatal(err)
		}
	}
	put(mine, "run-mine")
	put(other, "run-other")

	_, attrs := decodeRun(t, doStatus(t, nil, regs, "", false))
	if attrs.RunId != "run-mine" {
		t.Fatalf("runId = %q: another tenant's run is visible", attrs.RunId)
	}
}

func TestProcessStatusSharedRequiresOperator(t *testing.T) {
	regs, _ := newRegs(t)
	if rr := doStatus(t, nil, regs, "?scope=shared", false); rr.Code != http.StatusForbidden {
		t.Fatalf("status %d, want 403", rr.Code)
	}
	rr := doStatus(t, nil, regs, "?scope=shared", true)
	id, _ := decodeRun(t, rr)
	if id != "shared:GMS:83.1" {
		t.Fatalf("id = %q, want shared:GMS:83.1", id)
	}
}

func TestProcessStatusBogusScopeIs400(t *testing.T) {
	regs, _ := newRegs(t)
	if rr := doStatus(t, nil, regs, "?scope=bogus", false); rr.Code != http.StatusBadRequest {
		t.Fatalf("status %d, want 400", rr.Code)
	}
}

// A terminal record is returned as stored, with no Kubernetes call at all —
// this is what makes it readable after the Job is garbage-collected.
func TestProcessStatusTerminalRecordServedWithoutK8s(t *testing.T) {
	ctx := context.Background()
	regs, _ := newRegs(t)
	suffix := ingestrun.KeySuffix("tenants/"+testTenantId, "GMS", 83, 1)

	rec := ingestrun.NewRecord("run-1", "job-1", "tenants/"+testTenantId, "GMS", "83.1", testTenantId,
		time.Now().UTC().Add(-time.Hour), []string{"STRING", "MAP"})
	rec = rec.WithWorkerTerminal("STRING", ingestrun.WorkerSucceeded, time.Now().UTC(), "")
	rec = rec.WithWorkerTerminal("MAP", ingestrun.WorkerSkipped, time.Now().UTC(), "")
	rec = rec.WithPhase(ingestrun.PhaseSucceeded, time.Now().UTC(), "")
	if err := regs.Run.PutWithTTL(ctx, suffix+ingestrun.RunKeySuffix, rec, ingestrun.RecordTTL); err != nil {
		t.Fatal(err)
	}

	_, attrs := decodeRun(t, doStatus(t, nil, regs, "", false))
	if attrs.Phase != string(ingestrun.PhaseSucceeded) {
		t.Fatalf("phase = %s, want succeeded", attrs.Phase)
	}
	if attrs.WorkersTotal != 2 || attrs.WorkersComplete != 2 {
		t.Fatalf("workers %d/%d, want 2/2", attrs.WorkersComplete, attrs.WorkersTotal)
	}
	if attrs.FinishedAt == nil {
		t.Fatal("terminal record has no finishedAt")
	}
}

func TestProcessStatusStuckRecordSurfacesReason(t *testing.T) {
	ctx := context.Background()
	regs, _ := newRegs(t)
	suffix := ingestrun.KeySuffix("tenants/"+testTenantId, "GMS", 83, 1)

	rec := ingestrun.NewRecord("run-1", "job-1", "tenants/"+testTenantId, "GMS", "83.1", testTenantId,
		time.Now().UTC(), []string{"STRING", "MAP"})
	rec = rec.WithWorkerRunning("MAP", time.Now().UTC())
	rec = rec.WithPhase(ingestrun.PhaseStuck, time.Now().UTC(), "watchdog deleted the ingest Job after 7200s without a heartbeat")
	if err := regs.Run.PutWithTTL(ctx, suffix+ingestrun.RunKeySuffix, rec, ingestrun.RecordTTL); err != nil {
		t.Fatal(err)
	}

	_, attrs := decodeRun(t, doStatus(t, nil, regs, "", false))
	if attrs.Phase != string(ingestrun.PhaseStuck) {
		t.Fatalf("phase = %s, want stuck", attrs.Phase)
	}
	if attrs.Reason == nil || *attrs.Reason == "" {
		t.Fatal("stuck record has no reason")
	}
	if attrs.Workers[1].State != string(ingestrun.WorkerRunning) {
		t.Fatalf("MAP = %s, want running (preserved under a terminal phase)", attrs.Workers[1].State)
	}
}

func seedRunning(t *testing.T, regs *IngestRegistries) string {
	t.Helper()
	suffix := ingestrun.KeySuffix("tenants/"+testTenantId, "GMS", 83, 1)
	rec := ingestrun.NewRecord("run-1", "job-1", "tenants/"+testTenantId, "GMS", "83.1", testTenantId,
		time.Now().UTC(), []string{"STRING", "MAP"})
	if err := regs.Run.PutWithTTL(context.Background(), suffix+ingestrun.RunKeySuffix, rec, ingestrun.RecordTTL); err != nil {
		t.Fatal(err)
	}
	return suffix
}

func TestProcessStatusRunningWithFreshHeartbeatStaysRunning(t *testing.T) {
	regs, _ := newRegs(t)
	suffix := seedRunning(t, regs)
	if err := regs.Job.PutWithTTL(context.Background(), suffix+ingestrun.HeartbeatKeySuffix,
		time.Now().UTC().Format(time.RFC3339), time.Hour); err != nil {
		t.Fatal(err)
	}
	// No JobCreator at all: a fresh heartbeat alone is enough.
	_, attrs := decodeRun(t, doStatus(t, nil, regs, "", false))
	if attrs.Phase != string(ingestrun.PhaseRunning) {
		t.Fatalf("phase = %s, want running", attrs.Phase)
	}
}

func TestProcessStatusRunningWithStaleHeartbeatAndNoJobIsUnknown(t *testing.T) {
	regs, _ := newRegs(t)
	suffix := seedRunning(t, regs)
	stale := time.Now().UTC().Add(-time.Duration(DefaultWatchdogTimeoutSecs+60) * time.Second)
	if err := regs.Job.PutWithTTL(context.Background(), suffix+ingestrun.HeartbeatKeySuffix,
		stale.Format(time.RFC3339), time.Hour); err != nil {
		t.Fatal(err)
	}
	jc := &JobCreator{K8s: fake.NewSimpleClientset(), Namespace: "ns"}
	_, attrs := decodeRun(t, doStatus(t, jc, regs, "", false))
	if attrs.Phase != string(ingestrun.PhaseUnknown) {
		t.Fatalf("phase = %s, want unknown", attrs.Phase)
	}
}

func TestProcessStatusRunningWithNoK8sClientIsUnknown(t *testing.T) {
	regs, _ := newRegs(t)
	_ = seedRunning(t, regs)
	// No heartbeat, no JobCreator: nothing corroborates the running record.
	_, attrs := decodeRun(t, doStatus(t, nil, regs, "", false))
	if attrs.Phase != string(ingestrun.PhaseUnknown) {
		t.Fatalf("phase = %s, want unknown", attrs.Phase)
	}
}

// The label selector must narrow to this triple server-side: a live Job for a
// DIFFERENT version must not keep this run looking alive.
func TestProcessStatusJobListIsNarrowedToTheTriple(t *testing.T) {
	regs, _ := newRegs(t)
	_ = seedRunning(t, regs)

	otherVersion := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{
		Name: "j-other", Namespace: "ns",
		Labels: map[string]string{
			labelIngest: "true",
			"scope":     sanitizeLabel("tenants/" + testTenantId),
			"region":    "GMS", "version": "87.1", "tenant": testTenantId,
		},
	}, Status: batchv1.JobStatus{Active: 1}}
	cs := fake.NewSimpleClientset(otherVersion)
	jc := &JobCreator{K8s: cs, Namespace: "ns"}

	_, attrs := decodeRun(t, doStatus(t, jc, regs, "", false))
	if attrs.Phase != string(ingestrun.PhaseUnknown) {
		t.Fatalf("phase = %s, want unknown (the live Job is a different version)", attrs.Phase)
	}
}

func TestProcessStatusRunningWithLiveJobStaysRunning(t *testing.T) {
	regs, _ := newRegs(t)
	_ = seedRunning(t, regs)

	live := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{
		Name: "j-live", Namespace: "ns",
		Labels: map[string]string{
			labelIngest: "true",
			"scope":     sanitizeLabel("tenants/" + testTenantId),
			"region":    "GMS", "version": "83.1", "tenant": testTenantId,
		},
	}, Status: batchv1.JobStatus{Active: 1}}
	cs := fake.NewSimpleClientset(live)
	jc := &JobCreator{K8s: cs, Namespace: "ns"}

	_, attrs := decodeRun(t, doStatus(t, jc, regs, "", false))
	if attrs.Phase != string(ingestrun.PhaseRunning) {
		t.Fatalf("phase = %s, want running", attrs.Phase)
	}
}

// A failed Job list is not evidence of absence — corroborateRunning must keep
// reporting the stored `running` phase rather than downgrading to `unknown`
// (the deliberate §4.4-over-§7 deviation: resource.go:158-161).
func TestProcessStatusRunningWithJobListErrorStaysRunning(t *testing.T) {
	regs, _ := newRegs(t)
	_ = seedRunning(t, regs)

	cs := fake.NewSimpleClientset()
	cs.PrependReactor("list", "jobs", func(action ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, fmt.Errorf("simulated apiserver failure")
	})
	jc := &JobCreator{K8s: cs, Namespace: "ns"}

	_, attrs := decodeRun(t, doStatus(t, jc, regs, "", false))
	if attrs.Phase != string(ingestrun.PhaseRunning) {
		t.Fatalf("phase = %s, want running (a failed List must not be treated as evidence the Job is gone)", attrs.Phase)
	}
}

func TestProcessStatusWithNoRegistryReportsNone(t *testing.T) {
	rr := doStatus(t, nil, nil, "", false)
	_, attrs := decodeRun(t, rr)
	if attrs.Phase != string(ingestrun.PhaseNone) {
		t.Fatalf("phase = %s, want none", attrs.Phase)
	}
}
