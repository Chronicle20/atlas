package rest

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	clientrest "k8s.io/client-go/rest"
	"sigs.k8s.io/yaml"
)

// labelIngest marks Jobs that this service manages and that RecoverActiveJobs
// + the Watchdog use to scope their list queries.
const labelIngest = "atlas-data-ingest"

// jobTemplateConfigMapName is the canonical name of the ConfigMap that holds
// the JobTemplateSpec atlas-data renders into ingest Jobs.
const jobTemplateConfigMapName = "atlas-data-ingest-job-template"

// jobTemplateConfigMapKey is the key inside the ConfigMap holding the YAML
// definition of a batchv1.Job whose spec is copied into rendered Jobs.
const jobTemplateConfigMapKey = "job.yaml"

// JobCreator constructs k8s Jobs that run atlas-data in MODE=ingest.
//
// Template is a JobTemplateSpec used as the base for every Job and must be
// loaded from the atlas-data-ingest-job-template ConfigMap. Redis is used to
// publish a heartbeat key per (scope, region, version) so the Watchdog can
// notice stuck Jobs.
type JobCreator struct {
	K8s       kubernetes.Interface
	Namespace string
	Template  *batchv1.JobTemplateSpec
	Redis     *goredis.Client
	// ControllerImage is the container image the running atlas-data pod uses.
	// Rendered Jobs inherit it so MODE=ingest binaries match the code that
	// rendered them. Empty string falls back to the template's image (intended
	// for tests and single-image clusters; in PR/prod the template ships
	// `:latest` which is too stale to use).
	ControllerImage string
}

// NewJobCreatorInCluster builds a JobCreator using the pod's in-cluster
// ServiceAccount and loads the Job template from the
// atlas-data-ingest-job-template ConfigMap. Returns an error if the in-cluster
// config is unavailable (e.g. running outside Kubernetes) or the ConfigMap is
// missing/invalid.
func NewJobCreatorInCluster() (*JobCreator, error) {
	return NewJobCreatorInClusterWithRedis(nil)
}

// NewJobCreatorInClusterWithRedis is like NewJobCreatorInCluster but also
// attaches a Redis client used to publish per-Job heartbeat keys.
func NewJobCreatorInClusterWithRedis(rdb *goredis.Client) (*JobCreator, error) {
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
		if nsBytes, rerr := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); rerr == nil {
			ns = strings.TrimSpace(string(nsBytes))
		}
	}
	if ns == "" {
		ns = "default"
	}
	tmpl, err := loadTemplateFromConfigMap(context.Background(), cs, ns, jobTemplateConfigMapName)
	if err != nil {
		return nil, fmt.Errorf("load job template ConfigMap: %w", err)
	}
	img, ierr := discoverControllerImage(context.Background(), cs, ns)
	if ierr != nil {
		// Non-fatal: log path elsewhere. Empty ControllerImage leaves the
		// template's image untouched (sufficient for tests).
		_ = ierr
	}
	return &JobCreator{
		K8s:             cs,
		Namespace:       ns,
		Template:        tmpl,
		Redis:           rdb,
		ControllerImage: img,
	}, nil
}

// discoverControllerImage looks up the controller pod by its HOSTNAME (which
// k8s sets to the pod name by default) and returns the image of the container
// whose name matches the atlas-data primary container. This lets rendered Jobs
// inherit the same image tag the controller is running, side-stepping the
// kustomize-can't-patch-ConfigMap-strings problem that would otherwise pin
// Jobs to the template's `:latest`.
func discoverControllerImage(ctx context.Context, cs kubernetes.Interface, namespace string) (string, error) {
	host := os.Getenv("HOSTNAME")
	if host == "" {
		return "", fmt.Errorf("HOSTNAME unset; cannot self-identify pod")
	}
	pod, err := cs.CoreV1().Pods(namespace).Get(ctx, host, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get self pod %s/%s: %w", namespace, host, err)
	}
	for _, c := range pod.Spec.Containers {
		if strings.TrimSpace(c.Image) != "" {
			return c.Image, nil
		}
	}
	return "", fmt.Errorf("self pod %s/%s has no containers with image", namespace, host)
}

// loadTemplateFromConfigMap reads the configured ConfigMap and unmarshals its
// "job.yaml" key as a batchv1.Job, returning the Job's Spec wrapped in a
// JobTemplateSpec. Returns an error if the ConfigMap is missing, lacks the
// expected key, or contains an invalid Job document.
func loadTemplateFromConfigMap(ctx context.Context, cs kubernetes.Interface, namespace, name string) (*batchv1.JobTemplateSpec, error) {
	if cs == nil {
		return nil, fmt.Errorf("kubernetes client unavailable")
	}
	cm, err := cs.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("get ConfigMap %s/%s: %w", namespace, name, err)
	}
	raw, ok := cm.Data[jobTemplateConfigMapKey]
	if !ok || strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("ConfigMap %s/%s missing key %q", namespace, name, jobTemplateConfigMapKey)
	}
	var job batchv1.Job
	if err := yaml.Unmarshal([]byte(raw), &job); err != nil {
		return nil, fmt.Errorf("parse %s: %w", jobTemplateConfigMapKey, err)
	}
	if len(job.Spec.Template.Spec.Containers) == 0 {
		return nil, fmt.Errorf("ConfigMap %s/%s: %s has no containers", namespace, name, jobTemplateConfigMapKey)
	}
	for _, c := range job.Spec.Template.Spec.Containers {
		if strings.TrimSpace(c.Image) == "" {
			return nil, fmt.Errorf("ConfigMap %s/%s: container %q missing image", namespace, name, c.Name)
		}
	}
	return &batchv1.JobTemplateSpec{Spec: job.Spec}, nil
}

// Create renders and submits a Kubernetes Job for the given ingest run.
// Returns the generated Job name.
//
// scope must be either "shared" or "tenants/<tenantId>".
func (j *JobCreator) Create(ctx context.Context, scope, region string, major, minor uint16, tenantId, traceparent string) (string, error) {
	if j == nil || j.K8s == nil {
		return "", fmt.Errorf("job creator unavailable")
	}
	if j.Template == nil {
		return "", fmt.Errorf("job template unavailable")
	}
	job := renderJob(j.Template, j.Namespace, scope, region, major, minor, tenantId, traceparent, j.ControllerImage)
	created, err := j.K8s.BatchV1().Jobs(j.Namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("create job: %w", err)
	}
	if j.Redis != nil {
		key := redisJobKey(scope, region, major, minor)
		_ = j.Redis.Set(ctx, key, created.Name, time.Hour).Err()
		_ = j.Redis.Set(ctx, key+":updatedAt", time.Now().UTC().Format(time.RFC3339), time.Hour).Err()
	}
	return created.Name, nil
}

// redisJobKey produces the Redis key used to publish per-Job heartbeat data.
func redisJobKey(scope, region string, major, minor uint16) string {
	return fmt.Sprintf("atlas-data:ingest:%s:%s:%d.%d", scope, region, major, minor)
}

// redisJobKeyFromLabels reconstructs the per-Job Redis key from a Job's
// labels. Returns the empty string if any required label is missing (so the
// caller can fall back to the Job's creationTimestamp).
func redisJobKeyFromLabels(j *batchv1.Job) string {
	scope, region, version := j.Labels["scope"], j.Labels["region"], j.Labels["version"]
	if scope == "" || region == "" || version == "" {
		return ""
	}
	return fmt.Sprintf("atlas-data:ingest:%s:%s:%s", scope, region, version)
}

// renderJob produces a *batchv1.Job derived from template, scoped/labeled and
// with the ingest-specific env vars injected into every container.
func renderJob(template *batchv1.JobTemplateSpec, namespace, scope, region string, major, minor uint16, tenantId, traceparent, controllerImage string) *batchv1.Job {
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
	// Inherit DB_NAME from the running atlas-data pod so ingest Jobs hit the
	// same database. The Job template hardcodes DB_NAME="atlas-data" as a
	// sensible default for single-env clusters, but PR overlays patch the
	// Deployment env to suffix it per-env (e.g. atlas-data-bbb1). Kustomize
	// can't reach into the ConfigMap-embedded Job template to apply that same
	// patch, so we propagate the live value here. k8s env-list semantics are
	// last-wins, so appending overrides the template's default.
	if v := os.Getenv("DB_NAME"); v != "" {
		envs = append(envs, corev1.EnvVar{Name: "DB_NAME", Value: v})
	}

	for i := range spec.Template.Spec.Containers {
		spec.Template.Spec.Containers[i].Env = append(spec.Template.Spec.Containers[i].Env, envs...)
		// Override the template's hardcoded `:latest` image with the controller
		// pod's own image so the ingest binary matches the code that rendered
		// it. Kustomize image substitution can't reach into the ConfigMap-
		// embedded Job template, so this Go-side override is the only way to
		// keep tag-pinned PR environments coherent.
		if controllerImage != "" {
			spec.Template.Spec.Containers[i].Image = controllerImage
		}
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
func jobName(scope, region string, major, minor uint16) string {
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
