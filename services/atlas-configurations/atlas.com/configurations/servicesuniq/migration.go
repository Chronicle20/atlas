// Package servicesuniq dedupes any (type, environment) group that holds
// more than one services row, then adds a unique index on
// services (type, environment) (task-243 D3, §4.3 layer 2).
//
// It must run after services.Migration and after environmentcol.Migration:
// the latter backfills environment on every pre-existing row, so by the
// time this migration runs every row carries a non-empty value and the
// grouping below is meaningful.
package servicesuniq

import (
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	outboxlib "github.com/Chronicle20/atlas/libs/atlas-outbox"
)

// EnvServiceStatusTopic mirrors services.EnvServiceStatusTopic; it is
// duplicated as a literal here (rather than imported) to avoid an import
// cycle between this package and services.
const EnvServiceStatusTopic topic.Token = "EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS"

// atlasServiceNS is the UUIDv5 namespace a sparse environment's SERVICE_ID
// is derived from. It appears here and in exactly one other place,
// tools/derive-service-id.sh, which carries the reciprocal reference. Never
// regenerate it: changing it re-keys every sparse environment's
// service-config row. Reproducible rather than arbitrary, so the value can
// be re-derived if this line is ever lost:
//
//	uuid5(NAMESPACE_DNS, "service-config.atlas.chronicle20")
var atlasServiceNS = uuid.MustParse("c8f90111-a0cf-513e-95e6-c54609e5dec0")

// DuplicateGroup names a (type, environment) pair that holds more than one
// services row.
type DuplicateGroup struct {
	Type        string
	Environment string
	Count       int
}

// Preflight is the read-only Layer 3 check: it lists every duplicate group
// without touching any data.
func Preflight(db *gorm.DB) ([]DuplicateGroup, error) {
	var groups []DuplicateGroup
	err := db.Raw(
		"SELECT type, environment, COUNT(*) AS count FROM services GROUP BY type, environment HAVING COUNT(*) > 1",
	).Scan(&groups).Error
	if err != nil {
		return nil, fmt.Errorf("preflight duplicate services groups: %w", err)
	}
	return groups, nil
}

// candidateRow is a services row id, plus the newest service_history
// created_at recorded against it (zero value when there is no history).
type candidateRow struct {
	Id            uuid.UUID
	NewestHistory time.Time
	HasHistory    bool
}

// Migration runs the dedupe (Layer 2) and, once every group resolves
// unambiguously, creates the unique index that enforces the invariant going
// forward.
func Migration(db *gorm.DB) error {
	groups, err := Preflight(db)
	if err != nil {
		return err
	}

	for _, g := range groups {
		rows, err := candidateRowsFor(db, g.Type, g.Environment)
		if err != nil {
			return err
		}

		_, losers, err := resolveGroup(g, rows)
		if err != nil {
			return err
		}

		if len(losers) == 0 {
			continue
		}

		topic := os.Getenv(string(EnvServiceStatusTopic))
		err = database.ExecuteTransaction(db, func(tx *gorm.DB) error {
			for _, loser := range losers {
				if err := tx.Exec("DELETE FROM services WHERE id = ?", loser).Error; err != nil {
					return fmt.Errorf("delete duplicate service %s: %w", loser, err)
				}
				if topic == "" {
					continue
				}
				if err := outboxlib.Enqueue(tx, outboxlib.Message{
					Topic: topic,
					Key:   []byte("service:" + loser.String()),
					Value: nil,
				}); err != nil {
					return fmt.Errorf("enqueue tombstone for %s: %w", loser, err)
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_services_type_env ON services (type, environment)").Error; err != nil {
		return fmt.Errorf("create idx_services_type_env: %w", err)
	}
	return nil
}

// candidateRowsFor loads every services row id in the named group, plus the
// newest service_history.created_at recorded for that service_id (if any).
func candidateRowsFor(db *gorm.DB, serviceType, environment string) ([]candidateRow, error) {
	var ids []uuid.UUID
	if err := db.Raw(
		"SELECT id FROM services WHERE type = ? AND environment = ?", serviceType, environment,
	).Scan(&ids).Error; err != nil {
		return nil, fmt.Errorf("load candidate rows for %s/%s: %w", serviceType, environment, err)
	}

	rows := make([]candidateRow, 0, len(ids))
	for _, id := range ids {
		row := candidateRow{Id: id}

		newest, hasHistory, err := newestHistoryFor(db, id)
		if err != nil {
			return nil, fmt.Errorf("load newest history for %s: %w", id, err)
		}
		if hasHistory {
			row.HasHistory = true
			row.NewestHistory = newest
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// newestHistoryFor returns the newest service_history.created_at recorded
// for id, and whether any history row exists. Scanned as interface{}
// rather than *time.Time because SQLite's MAX() aggregate can report the
// underlying driver value as a string rather than a native time.Time.
func newestHistoryFor(db *gorm.DB, id uuid.UUID) (time.Time, bool, error) {
	var raw interface{}
	if err := db.Raw(
		"SELECT MAX(created_at) FROM service_history WHERE service_id = ?", id,
	).Row().Scan(&raw); err != nil {
		return time.Time{}, false, err
	}

	var s string
	switch v := raw.(type) {
	case nil:
		return time.Time{}, false, nil
	case time.Time:
		return v, true, nil
	case string:
		s = v
	case []byte:
		s = string(v)
	default:
		return time.Time{}, false, fmt.Errorf("unexpected created_at scan type %T", raw)
	}

	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05.999999999-07:00", "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true, nil
		}
	}
	return time.Time{}, false, fmt.Errorf("unparseable created_at value %q", s)
}

// resolveGroup applies the keeper rule: the row whose id equals
// uuid5(atlasServiceNS, type+"/"+environment) is kept and every other row in
// the group is a loser. If no row matches the derived id, the group is
// unresolvable and resolveGroup returns an error naming every candidate
// rather than falling back to a heuristic that could delete a row the
// system depends on (e.g. a canonical pinned id whose history happens to be
// older than an interloper's). An operator must resolve such a group by
// hand; see docs/runbooks/sparse-environments.md §"Pre-flight".
func resolveGroup(g DuplicateGroup, rows []candidateRow) (keeper uuid.UUID, losers []uuid.UUID, err error) {
	derived := uuid.NewSHA1(atlasServiceNS, []byte(g.Type+"/"+g.Environment))

	for _, r := range rows {
		if r.Id == derived {
			keeper = r.Id
			break
		}
	}

	if keeper == uuid.Nil {
		ids := make([]string, 0, len(rows))
		for _, r := range rows {
			ids = append(ids, r.Id.String())
		}
		return uuid.Nil, nil, fmt.Errorf(
			"servicesuniq: cannot resolve duplicate group type=%s environment=%s unambiguously, candidates=%v",
			g.Type, g.Environment, ids,
		)
	}

	for _, r := range rows {
		if r.Id != keeper {
			losers = append(losers, r.Id)
		}
	}
	return keeper, losers, nil
}
