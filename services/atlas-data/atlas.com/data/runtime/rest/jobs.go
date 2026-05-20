package rest

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	clientrest "k8s.io/client-go/rest"
)

// labelIngest marks Jobs that this service manages and that RecoverActiveJobs
// + the Watchdog use to scope their list queries.
const labelIngest = "atlas-data-ingest"

// JobCreator constructs k8s Jobs that run atlas-data in MODE=ingest.
//
// Template is a JobTemplateSpec used as the base for every Job. In production
// it should be loaded from a ConfigMap (see Task 14); for now it is a minimal
// hardcoded template if no ConfigMap is wired in.
type JobCreator struct {
	K8s       kubernetes.Interface
	Namespace string
	Template  *batchv1.JobTemplateSpec
}

// NewJobCreatorInCluster builds a JobCreator using the pod's in-cluster
// ServiceAccount. Returns an error if the in-cluster config is unavailable
// (e.g. running outside Kubernetes).
func NewJobCreatorInCluster() (*JobCreator, error) {
	cfg, err := clientrest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("in-cluster config: %w", err)
	}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("kubernetes clientset: %w", err)
	}
	ns := os.Getenv("POD_NAMESPACE")
	if ns == "" {
		ns = "default"
	}
	return &JobCreator{
		K8s:       cs,
		Namespace: ns,
		Template:  defaultTemplate(),
	}, nil
}

// defaultTemplate returns a minimal JobTemplateSpec. The container image is
// taken from INGEST_IMAGE; in production the entire template should come from
// the ingest-job-template ConfigMap (Task 14 follow-up).
func defaultTemplate() *batchv1.JobTemplateSpec {
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
						Image: os.Getenv("INGEST_IMAGE"),
					}},
				},
			},
		},
	}
}

// Create renders and submits a Kubernetes Job for the given ingest run.
// Returns the generated Job name.
//
// scope must be either "shared" or "tenants/<tenantId>".
func (j *JobCreator) Create(ctx context.Context, scope, region string, major, minor int, tenantId, traceparent string) (string, error) {
	if j == nil || j.K8s == nil {
		return "", fmt.Errorf("job creator unavailable")
	}
	job := renderJob(j.Template, j.Namespace, scope, region, major, minor, tenantId, traceparent)
	created, err := j.K8s.BatchV1().Jobs(j.Namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("create job: %w", err)
	}
	return created.Name, nil
}

// renderJob produces a *batchv1.Job derived from template, scoped/labeled and
// with the ingest-specific env vars injected into every container.
func renderJob(template *batchv1.JobTemplateSpec, namespace, scope, region string, major, minor int, tenantId, traceparent string) *batchv1.Job {
	var spec batchv1.JobSpec
	if template != nil {
		spec = *template.Spec.DeepCopy()
	} else {
		backoff := int32(0)
		spec = batchv1.JobSpec{
			BackoffLimit: &backoff,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers:    []corev1.Container{{Name: "ingest"}},
				},
			},
		}
	}

	scopeLabel := sanitizeLabel(scope)
	versionLabel := fmt.Sprintf("%d.%d", major, minor)
	name := jobName(scope, region, major, minor)

	envs := []corev1.EnvVar{
		{Name: "MODE", Value: "ingest"},
		{Name: "SCOPE", Value: scope},
		{Name: "REGION", Value: region},
		{Name: "MAJOR_VERSION", Value: fmt.Sprintf("%d", major)},
		{Name: "MINOR_VERSION", Value: fmt.Sprintf("%d", minor)},
		{Name: "TENANT_ID", Value: tenantId},
	}
	if traceparent != "" {
		envs = append(envs, corev1.EnvVar{Name: "TRACEPARENT", Value: traceparent})
	}

	for i := range spec.Template.Spec.Containers {
		spec.Template.Spec.Containers[i].Env = append(spec.Template.Spec.Containers[i].Env, envs...)
	}

	labels := map[string]string{
		labelIngest: "true",
		"scope":     scopeLabel,
		"region":    region,
		"version":   versionLabel,
	}
	if tenantId != "" {
		labels["tenant"] = tenantId
	}

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: spec,
	}
}

// jobName produces a deterministic-ish but unique Job name suffixed with a
// short random token. k8s names must match DNS-1123 (lowercase alphanumeric
// and dashes).
func jobName(scope, region string, major, minor int) string {
	scopeSeg := "shared"
	if strings.HasPrefix(scope, "tenants/") {
		id := strings.TrimPrefix(scope, "tenants/")
		// take first 8 chars of the tenant id for the name segment
		if len(id) > 8 {
			id = id[:8]
		}
		scopeSeg = "t-" + id
	}
	suffix := randSuffix()
	name := fmt.Sprintf("ingest-%s-%s-%d-%d-%s", scopeSeg, strings.ToLower(region), major, minor, suffix)
	return sanitizeLabel(name)
}

var nameRand = rand.New(rand.NewSource(time.Now().UnixNano()))

func randSuffix() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = charset[nameRand.Intn(len(charset))]
	}
	return string(b)
}

// sanitizeLabel coerces an arbitrary string into a DNS-1123-compatible value:
// lowercase, alnum + '-', max length 63. The "shared" / "tenants/<id>"
// distinction would otherwise produce '/' which is invalid in labels.
func sanitizeLabel(in string) string {
	if in == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range strings.ToLower(in) {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	out := b.String()
	out = strings.Trim(out, "-.")
	if len(out) > 63 {
		out = out[:63]
		out = strings.TrimRight(out, "-.")
	}
	return out
}
