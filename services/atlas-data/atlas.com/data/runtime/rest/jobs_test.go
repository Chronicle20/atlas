package rest

import (
	"context"
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
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
	job := renderJob(testTemplate(), "ns", "shared", "GMS", 83, 1, "", "")
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
