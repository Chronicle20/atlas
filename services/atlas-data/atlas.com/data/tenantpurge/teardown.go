package tenantpurge

import (
	"context"
	"errors"
	"fmt"
	"time"

	minio "atlas-data/storage/minio"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"gorm.io/gorm"
)

// NamespaceLister is the subset of the Kubernetes API client used by
// PurgeAllIfNamespaceTerminating. Defined as an interface so tests can
// substitute a fake without spinning up envtest. The signature matches
// `(kubernetes.Interface).CoreV1().Namespaces()`.
type NamespaceLister interface {
	NamespaceDeletionTimestamp(ctx context.Context, name string) (*time.Time, error)
}

// kubeClientNamespaceLister wraps a real kubernetes.Interface.
type kubeClientNamespaceLister struct {
	client kubernetes.Interface
}

// NewKubeNamespaceLister returns a NamespaceLister backed by the
// supplied kubernetes client.
func NewKubeNamespaceLister(client kubernetes.Interface) NamespaceLister {
	return &kubeClientNamespaceLister{client: client}
}

func (k *kubeClientNamespaceLister) NamespaceDeletionTimestamp(ctx context.Context, name string) (*time.Time, error) {
	ns, err := k.client.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if ns.DeletionTimestamp == nil {
		return nil, nil
	}
	t := ns.DeletionTimestamp.Time
	return &t, nil
}

// TenantEnumerator returns the list of tenant UUIDs (as strings) that
// atlas-data has stored data for. Defined as an interface so tests can
// inject a fixed list. The production implementation reads
// `SELECT DISTINCT tenant_id FROM tenant_baselines`.
type TenantEnumerator interface {
	EnumerateTenants(ctx context.Context) ([]string, error)
}

// dbTenantEnumerator queries tenant_baselines.
type dbTenantEnumerator struct {
	db *gorm.DB
}

// NewDBTenantEnumerator returns an enumerator that reads from
// `tenant_baselines`.
func NewDBTenantEnumerator(db *gorm.DB) TenantEnumerator {
	return &dbTenantEnumerator{db: db}
}

func (e *dbTenantEnumerator) EnumerateTenants(ctx context.Context) ([]string, error) {
	var ids []string
	if err := e.db.WithContext(ctx).Raw(
		"SELECT DISTINCT tenant_id FROM tenant_baselines",
	).Scan(&ids).Error; err != nil {
		return nil, fmt.Errorf("enumerate tenant_baselines: %w", err)
	}
	return ids, nil
}

// PurgeAllIfNamespaceTerminating is the graceful-shutdown handler. It
// checks whether atlas-data's own namespace has `deletionTimestamp`
// set — only then does it enumerate atlas-data's tenants and purge
// each. This distinguishes env teardown (namespace being deleted) from
// routine pod restarts (rolling update, manual delete, OOM-restart).
//
// On a routine restart, namespace deletion is NOT in progress and this
// function returns early — preserving per-tenant data across image
// bumps and pod replacements.
//
// On env teardown, Argo CD sets deletionTimestamp on the namespace as
// part of the Application's resources-finalizer drain. By the time
// SIGTERM reaches atlas-data, the namespace's deletionTimestamp is
// observable.
//
// The function is best-effort: failures are logged at warn level but
// don't propagate (we're already mid-shutdown, can't usefully retry).
// The orphan-sweep backstop (`sweep-orphans.sh --minio`) catches any
// tenant data that escapes this hook (e.g., atlas-data OOM-killed
// before SIGTERM, atlas-data restarting before namespace deletionTimestamp
// was visible).
//
// Issue #596.
func PurgeAllIfNamespaceTerminating(
	ctx context.Context,
	l logrus.FieldLogger,
	db *gorm.DB,
	mc *minio.Client,
	nsLister NamespaceLister,
	tenantEnum TenantEnumerator,
	namespace string,
) {
	if nsLister == nil || namespace == "" {
		l.Debug("teardown: no namespace lister or namespace; skipping")
		return
	}

	delTs, err := nsLister.NamespaceDeletionTimestamp(ctx, namespace)
	if err != nil {
		l.WithError(err).WithField("namespace", namespace).
			Warn("teardown: fetch own namespace failed; skipping per-tenant purge")
		return
	}
	if delTs == nil {
		// Routine restart, not env teardown.
		l.WithField("namespace", namespace).
			Debug("teardown: namespace not being deleted; skipping per-tenant purge")
		return
	}

	l.WithField("namespace", namespace).WithField("deletionTimestamp", *delTs).
		Info("teardown: namespace being deleted; purging per-tenant data")

	tenantIDs, err := tenantEnum.EnumerateTenants(ctx)
	if err != nil {
		l.WithError(err).Warn("teardown: enumerate tenants failed; skipping per-tenant purge")
		return
	}
	for _, idStr := range tenantIDs {
		id, perr := uuid.Parse(idStr)
		if perr != nil {
			l.WithError(perr).WithField("tenant_id", idStr).
				Warn("teardown: invalid tenant id; skipping")
			continue
		}
		if err := Purge(ctx, l, db, mc, id); err != nil {
			if errors.Is(err, ErrCanonicalRefused) {
				l.WithField("tenant_id", idStr).
					Info("teardown: skip canonical tenant")
				continue
			}
			l.WithError(err).WithField("tenant_id", idStr).
				Warn("teardown: purge tenant failed")
			continue
		}
		l.WithField("tenant_id", idStr).Info("teardown: tenant purged")
	}
}
