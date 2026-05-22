package rest

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// RecoverActiveJobs lists Jobs labeled as atlas-data ingest Jobs and returns
// the names of those that are still "active" (i.e. not yet succeeded or
// failed). Called once at startup so operators can see which ingest runs
// survived a REST-pod restart.
func RecoverActiveJobs(ctx context.Context, cs kubernetes.Interface, namespace string) ([]string, error) {
	if cs == nil {
		return nil, fmt.Errorf("kubernetes client unavailable")
	}
	list, err := cs.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelIngest + "=true",
	})
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	active := make([]string, 0, len(list.Items))
	for _, j := range list.Items {
		if j.Status.Succeeded > 0 {
			continue
		}
		// A Job is considered failed if it has a Failed condition; we treat
		// anything not yet succeeded as "still active" for operator visibility.
		failed := false
		for _, c := range j.Status.Conditions {
			if c.Type == "Failed" && c.Status == "True" {
				failed = true
				break
			}
		}
		if failed {
			continue
		}
		active = append(active, j.Name)
	}
	return active, nil
}
