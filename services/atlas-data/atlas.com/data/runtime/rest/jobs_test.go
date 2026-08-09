package rest

import (
	"atlas-data/data/workers"
	"atlas-data/ingestrun"
	"context"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	redis "github.com/Chronicle20/atlas/libs/atlas-redis"
)

// testTemplate returns a minimal JobTemplateSpec used by tests in lieu of the
// production ConfigMap-loaded template.
func testTemplate() *batchv1.JobTemplateSpec {
	backoff := int32(0)
	ttl := int32(3600)
	return &batchv1.JobTemplateSpec{
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoff,
			TTLSecondsAfterFinished: &ttl,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{{
						Name:  "ingest",
						Image: "atlas-data:test",
					}},
				},
			},
		},
	}
}

func TestJobCreatorCreate(t *testing.T) {
	cs := fake.NewSimpleClientset()
	jc := &JobCreator{K8s: cs, Namespace: "test-ns", Template: testTemplate()}
	name, err := jc.Create(context.Background(), "tenants/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "GMS", 83, 1, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "trace-1")
	if err != nil {
		t.Fatal(err)
	}
	got, err := cs.BatchV1().Jobs("test-ns").Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Labels[labelIngest] != "true" {
		t.Fatalf("missing ingest label, got %v", got.Labels)
	}
	if got.Labels["region"] != "GMS" {
		t.Fatalf("region label = %s", got.Labels["region"])
	}
	if got.Labels["version"] != "83.1" {
		t.Fatalf("version label = %s", got.Labels["version"])
	}
	if got.Labels["tenant"] != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("tenant label = %s", got.Labels["tenant"])
	}
	if c := got.Spec.Template.Spec.Containers; len(c) != 1 {
		t.Fatalf("expected 1 container, got %d", len(c))
	}
	want := map[string]string{
		"MODE":          "ingest",
		"SCOPE":         "tenants/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		"REGION":        "GMS",
		"MAJOR_VERSION": "83",
		"MINOR_VERSION": "1",
		"TENANT_ID":     "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		"TRACEPARENT":   "trace-1",
	}
	have := map[string]string{}
	for _, e := range got.Spec.Template.Spec.Containers[0].Env {
		have[e.Name] = e.Value
	}
	for k, v := range want {
		if have[k] != v {
			t.Fatalf("env %s = %q, want %q", k, have[k], v)
		}
	}
}

func TestRenderJobSharedScopeOmitsTenantLabel(t *testing.T) {
	job := renderJob(testTemplate(), "ns", "shared", "GMS", 83, 1, "", "", "", "")
	if _, ok := job.Labels["tenant"]; ok {
		t.Fatalf("did not expect tenant label for shared scope")
	}
	if job.Labels["scope"] != "shared" {
		t.Fatalf("scope label = %s", job.Labels["scope"])
	}
	for _, e := range job.Spec.Template.Spec.Containers[0].Env {
		if e.Name == "TRACEPARENT" {
			t.Fatalf("did not expect TRACEPARENT env when traceparent is empty")
		}
	}
}

// The Redis key suffix uses the RAW scope; the Kubernetes label uses the
// sanitized form. Reconstructing the suffix from labels must undo that
// sanitisation or the Watchdog reads a key nobody writes (design F-2).
func TestIngestJobKeySuffixFromLabelsRoundTrips(t *testing.T) {
	cases := []struct {
		name     string
		scope    string
		tenantId string
	}{
		{"shared", "shared", ""},
		{"tenant", "tenants/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			job := renderJob(testTemplate(), "ns", tc.scope, "GMS", 83, 1, tc.tenantId, "", "", "run-1")
			want := ingestrun.KeySuffix(tc.scope, "GMS", 83, 1)
			if got := ingestJobKeySuffixFromLabels(job); got != want {
				t.Fatalf("got %q, want %q", got, want)
			}
		})
	}
}

func TestIngestJobKeySuffixFromLabelsRejectsIncompleteLabels(t *testing.T) {
	// Non-shared scope with no tenant label cannot be reconstructed; the
	// contract is to skip (return "") rather than guess.
	j := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{
		"scope": "tenants-aaaaaaaa", "region": "GMS", "version": "83.1",
	}}}
	if got := ingestJobKeySuffixFromLabels(j); got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}

func TestRenderJobInjectsRunId(t *testing.T) {
	job := renderJob(testTemplate(), "ns", "shared", "GMS", 83, 1, "", "", "", "run-abc")
	var found string
	for _, e := range job.Spec.Template.Spec.Containers[0].Env {
		if e.Name == "INGEST_RUN_ID" {
			found = e.Value
		}
	}
	if found != "run-abc" {
		t.Fatalf("INGEST_RUN_ID = %q, want run-abc", found)
	}
}

func TestJobCreatorCreateInitialisesRunRecord(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	regs := NewIngestRegistries(rdb)

	cs := fake.NewSimpleClientset()
	jc := &JobCreator{
		K8s: cs, Namespace: "ns", Template: testTemplate(),
		Registry: regs.Job, RunRegistry: regs.Run,
	}
	scope := "tenants/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	name, err := jc.Create(context.Background(), scope, "GMS", 83, 1, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "")
	if err != nil {
		t.Fatal(err)
	}

	suffix := ingestrun.KeySuffix(scope, "GMS", 83, 1)
	rec, err := regs.Run.Get(context.Background(), suffix+ingestrun.RunKeySuffix)
	if err != nil {
		t.Fatalf("run record not written: %v", err)
	}
	if rec.Phase != ingestrun.PhaseRunning {
		t.Fatalf("phase = %s, want running", rec.Phase)
	}
	if rec.JobName != name {
		t.Fatalf("jobName = %q, want %q", rec.JobName, name)
	}
	if rec.RunId == "" {
		t.Fatal("run record has no runId")
	}
	if rec.Scope != scope || rec.Region != "GMS" || rec.Version != "83.1" {
		t.Fatalf("record identity wrong: %+v", rec)
	}
	if len(rec.Workers) != len(workers.RegisteredNames()) {
		t.Fatalf("roster size = %d, want %d", len(rec.Workers), len(workers.RegisteredNames()))
	}
	for _, w := range rec.Workers {
		if w.State != ingestrun.WorkerPending {
			t.Fatalf("worker %s = %s, want pending", w.Name, w.State)
		}
	}
	if rec.StartedAt.IsZero() {
		t.Fatal("record has no startedAt")
	}

	// The record's runId must be the one the pod will read from its env.
	created, err := cs.BatchV1().Jobs("ns").Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	var envRunId string
	for _, e := range created.Spec.Template.Spec.Containers[0].Env {
		if e.Name == "INGEST_RUN_ID" {
			envRunId = e.Value
		}
	}
	if envRunId != rec.RunId {
		t.Fatalf("job env runId %q != record runId %q", envRunId, rec.RunId)
	}

	// The run record must expire; PutWithTTL(RecordTTL) is what sets that up.
	if ttl := mr.TTL(strings.Join([]string{redis.KeyPrefix(), ingestrun.Namespace, suffix + ingestrun.RunKeySuffix}, ":")); ttl <= 0 {
		t.Fatalf("run record TTL = %v, want > 0", ttl)
	}
}

// The ingest pod and the REST pod must namespace Redis identically or every
// progress/heartbeat write lands under a key nobody reads. ATLAS_ENV is what
// libs/atlas-redis derives that namespace from, and it reaches the REST pod
// through a Deployment env patch that Kustomize cannot apply to the
// ConfigMap-embedded Job template — so, exactly like DB_NAME, it has to be
// propagated here. PR-1266 evidence: the ingest pod wrote
// `atlas:data-ingest:shared:GMS:48.1:run` while the REST pod read
// `cc04:atlas:data-ingest:shared:GMS:48.1:run`, leaving every worker pending
// in the UI for the whole run and freezing the Watchdog heartbeat at its
// Job-creation value.
func TestRenderJobPropagatesAtlasEnv(t *testing.T) {
	t.Setenv("ATLAS_ENV", "cc04")
	job := renderJob(testTemplate(), "ns", "shared", "GMS", 83, 1, "", "", "", "run-abc")
	var found string
	for _, e := range job.Spec.Template.Spec.Containers[0].Env {
		if e.Name == "ATLAS_ENV" {
			found = e.Value
		}
	}
	if found != "cc04" {
		t.Fatalf("ATLAS_ENV = %q, want cc04", found)
	}
}

func TestRenderJobOmitsAtlasEnvWhenUnset(t *testing.T) {
	t.Setenv("ATLAS_ENV", "")
	job := renderJob(testTemplate(), "ns", "shared", "GMS", 83, 1, "", "", "", "run-abc")
	for _, e := range job.Spec.Template.Spec.Containers[0].Env {
		if e.Name == "ATLAS_ENV" {
			t.Fatalf("did not expect ATLAS_ENV env when unset (main env uses the bare prefix)")
		}
	}
}
