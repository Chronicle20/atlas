package rest

import (
	"context"
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestRecoverActiveJobs(t *testing.T) {
	cs := fake.NewSimpleClientset(
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "active",
				Namespace: "ns",
				Labels:    map[string]string{labelIngest: "true"},
			},
			Status: batchv1.JobStatus{Active: 1},
		},
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "done",
				Namespace: "ns",
				Labels:    map[string]string{labelIngest: "true"},
			},
			Status: batchv1.JobStatus{Succeeded: 1},
		},
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "failed",
				Namespace: "ns",
				Labels:    map[string]string{labelIngest: "true"},
			},
			Status: batchv1.JobStatus{
				Conditions: []batchv1.JobCondition{{Type: batchv1.JobFailed, Status: "True"}},
			},
		},
	)
	names, err := RecoverActiveJobs(context.Background(), cs, "ns")
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 1 || names[0] != "active" {
		t.Fatalf("expected [active], got %v", names)
	}
}
