package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestProcessStatusListsActiveJobs(t *testing.T) {
	cs := fake.NewSimpleClientset(
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "j1",
				Namespace: "ns",
				Labels: map[string]string{
					labelIngest: "true",
					"scope":     "tenants-t1", "region": "GMS", "version": "83.1",
					"tenant": "t1",
				},
			},
			Status: batchv1.JobStatus{Active: 1},
		},
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "j2-not-ingest",
				Namespace: "ns",
				Labels:    map[string]string{"unrelated": "true"},
			},
			Status: batchv1.JobStatus{Active: 1},
		},
	)
	jc := &JobCreator{K8s: cs, Namespace: "ns"}
	d := server.NewHandlerDependency(logrus.New(), context.Background())
	c := server.NewHandlerContext(nil)
	h := processStatus(jc)(&d, &c)

	rr := httptest.NewRecorder()
	h(rr, httptest.NewRequest(http.MethodGet, "/api/data/process", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var body struct {
		Jobs []processStatusJob `json:"jobs"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Jobs) != 1 {
		t.Fatalf("got %d jobs, want 1 (ingest-labeled only): %+v", len(body.Jobs), body.Jobs)
	}
	j := body.Jobs[0]
	if j.Name != "j1" || j.Region != "GMS" || j.Version != "83.1" || j.Tenant != "t1" || j.Active != 1 {
		t.Fatalf("unexpected job entry: %+v", j)
	}
}

func TestProcessStatus503WhenK8sUnavailable(t *testing.T) {
	d := server.NewHandlerDependency(logrus.New(), context.Background())
	c := server.NewHandlerContext(nil)
	h := processStatus(nil)(&d, &c)
	rr := httptest.NewRecorder()
	h(rr, httptest.NewRequest(http.MethodGet, "/api/data/process", nil))
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("got %d, want 503", rr.Code)
	}
}
