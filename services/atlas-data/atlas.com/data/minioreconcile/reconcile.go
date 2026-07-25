package minioreconcile

import (
	"atlas-data/canonical"
	"context"
	"errors"
	"time"

	"github.com/sirupsen/logrus"
)

// ErrEmptyKeepList is returned when the request carries no tenant ids. An empty
// keep-list must never be interpreted as "delete everything".
var ErrEmptyKeepList = errors.New("keep-list is empty; refusing to reconcile")

// PrefixInfo is the aggregate for one tenants/<uuid>/ prefix.
type PrefixInfo struct {
	Count  int
	Bytes  int64
	Newest time.Time
}

// Store is the narrow MinIO surface Reconcile needs. Implemented by minioStore
// (store.go) in production and by a fake in tests.
type Store interface {
	ListTenantIDs(ctx context.Context, bucket string) ([]string, error)
	PrefixInfo(ctx context.Context, bucket, prefix string) (PrefixInfo, error)
	RemovePrefix(ctx context.Context, bucket, prefix string) error
	Buckets() []string
}

// Request is the reconcile input.
type Request struct {
	KeepTenantIDs []string
	MinAgeHours   int
	DryRun        bool
}

// ReportRow is one prefix's outcome. Action ∈ {"deleted","would-delete","kept-too-new"}.
type ReportRow struct {
	Bucket   string
	TenantID string
	Action   string
	Count    int
	Bytes    int64
	Newest   time.Time
}

// Report aggregates the sweep result. Totals count only eligible prefixes
// (deleted or would-delete), not kept-too-new.
type Report struct {
	DryRun        bool
	MinAgeHours   int
	TotalPrefixes int
	TotalBytes    int64
	Rows          []ReportRow
}

// Reconcile removes tenants/<uuid>/ prefixes not in req.KeepTenantIDs whose
// newest object is at least req.MinAgeHours old, across every bucket the store
// reports. It refuses an empty keep-list and never touches the canonical
// sentinel. `now` is injected for deterministic age tests.
func Reconcile(ctx context.Context, l logrus.FieldLogger, store Store, req Request, now time.Time) (Report, error) {
	if len(req.KeepTenantIDs) == 0 {
		return Report{}, ErrEmptyKeepList
	}
	keep := make(map[string]struct{}, len(req.KeepTenantIDs))
	for _, id := range req.KeepTenantIDs {
		keep[id] = struct{}{}
	}
	minAge := time.Duration(req.MinAgeHours) * time.Hour
	rep := Report{DryRun: req.DryRun, MinAgeHours: req.MinAgeHours}

	for _, bucket := range store.Buckets() {
		ids, err := store.ListTenantIDs(ctx, bucket)
		if err != nil {
			return rep, err
		}
		for _, id := range ids {
			if _, ok := keep[id]; ok {
				continue
			}
			if id == canonical.TenantUUID {
				continue // canonical sentinel is never per-tenant data; never delete
			}
			prefix := "tenants/" + id + "/"
			info, err := store.PrefixInfo(ctx, bucket, prefix)
			if err != nil {
				return rep, err
			}
			if info.Count == 0 {
				continue
			}
			if now.Sub(info.Newest) < minAge {
				rep.Rows = append(rep.Rows, ReportRow{Bucket: bucket, TenantID: id, Action: "kept-too-new", Count: info.Count, Bytes: info.Bytes, Newest: info.Newest})
				continue
			}
			action := "would-delete"
			if !req.DryRun {
				if err := store.RemovePrefix(ctx, bucket, prefix); err != nil {
					l.WithError(err).Warnf("reconcile: failed to remove %s/%s", bucket, prefix)
					return rep, err
				}
				action = "deleted"
			}
			l.Infof("reconcile: %s %s/%s (%d objects, %d bytes, newest %s)", action, bucket, prefix, info.Count, info.Bytes, info.Newest.UTC().Format(time.RFC3339))
			rep.Rows = append(rep.Rows, ReportRow{Bucket: bucket, TenantID: id, Action: action, Count: info.Count, Bytes: info.Bytes, Newest: info.Newest})
			rep.TotalPrefixes++
			rep.TotalBytes += info.Bytes
		}
	}
	return rep, nil
}
