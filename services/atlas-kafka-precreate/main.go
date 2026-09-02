// Command atlas-kafka-precreate is the sync-wave-0 Kubernetes Job that
// pre-creates every Kafka topic an Atlas environment's Job/Deployment specs
// declare, applies cleanup.policy=compact to the config-projection topics,
// and seeds committed offsets for any override consumer group so a first
// (or re-)sync never replays the full retention window. See README.md for
// the environment variables, the five phases, and the exit-code contract.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	kafka "github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"atlas.com/kafka-precreate/internal/discover"
	"atlas.com/kafka-precreate/internal/groups"
	"atlas.com/kafka-precreate/internal/kafkaops"
	"atlas.com/kafka-precreate/internal/manifest"
	"atlas.com/kafka-precreate/internal/topics"
)

func main() {
	if err := run(); err != nil {
		logrus.WithError(err).Error("kafka precreate failed")
		os.Exit(1)
	}
}

func run() error {
	logrus.SetFormatter(&logrus.JSONFormatter{})

	bootstrap := os.Getenv("BOOTSTRAP_SERVERS")
	if bootstrap == "" {
		return fmt.Errorf("BOOTSTRAP_SERVERS not set in atlas-env")
	}

	// Comfortably inside the Job's activeDeadlineSeconds: 300, so the
	// tool's own deadline fires first and produces a named error rather
	// than an opaque pod kill (NFR-4).
	ctx, cancel := context.WithTimeout(context.Background(), 240*time.Second)
	defer cancel()

	addr := kafka.TCP(strings.Split(bootstrap, ",")...)
	client := &kafka.Client{
		Addr:    addr,
		Timeout: 60 * time.Second,
		// The default MetadataTTL is 6s and the transport serves Metadata
		// from that cache, so a shorter TTL is what lets topics.Settle
		// observe freshly created topics quickly (design §3 Phase C).
		Transport: &kafka.Transport{MetadataTTL: 1 * time.Second},
	}

	// Phase A: discover. The topic set is the code-derived manifest mounted
	// from the atlas-kafka-topics ConfigMap; names still resolve through
	// atlas-env because they carry the per-environment suffix the manifest
	// deliberately does not encode (task-276 FR-4.3).
	path := os.Getenv("KAFKA_TOPIC_MANIFEST_PATH")
	if path == "" {
		path = "/etc/atlas/topics/topics.yaml"
	}
	m, err := manifest.Load(path)
	if err != nil {
		return fmt.Errorf("loading topic manifest from %s: %w", path, err)
	}
	t, err := manifest.Resolve(m, os.LookupEnv)
	if err != nil {
		return err
	}
	logrus.WithFields(logrus.Fields{
		"phase":    "discover",
		"manifest": path,
		"tokens":   len(m.Topics),
		"compact":  len(t.Compact),
	}).Info("topic manifest loaded")
	groupIDs := discover.Groups(os.Getenv("KAFKA_CONSUMER_GROUP"))
	union := t.Union()

	// Phase B: create and alter.
	ensureResult, err := topics.Ensure(ctx, client, addr, t)
	if err != nil {
		return err
	}
	logrus.WithFields(logrus.Fields{
		"phase":    "create",
		"plain":    len(t.Plain),
		"compact":  len(t.Compact),
		"created":  ensureResult.Created,
		"existing": ensureResult.Existing,
	}).Info("topics ensured")
	if len(t.Compact) > 0 {
		logrus.WithFields(logrus.Fields{
			"phase":   "alter",
			"topics":  len(t.Compact),
			"policy":  "compact",
			"configs": topics.CompactConfigNames(),
		}).Info("compacted topic configuration applied")
	}

	if len(groupIDs) == 0 {
		logrus.WithFields(logrus.Fields{
			"phase":   "seed",
			"skipped": true,
			"reason":  "KAFKA_CONSUMER_GROUP unset — main, NG6",
		}).Info("seed skipped")
		return nil
	}

	// Phase C: settle metadata and read end offsets.
	parts, err := topics.Settle(ctx, client, addr, union, topics.SettleConfig{})
	if err != nil {
		return err
	}
	offsets, err := topics.EndOffsets(ctx, client, addr, parts, kafkaops.DefaultRetryConfig())
	if err != nil {
		return err
	}

	// Phase D: seed.
	res, err := groups.Seed(ctx, client, addr, groupIDs, parts, offsets, kafkaops.DefaultRetryConfig())
	if err != nil {
		return err
	}
	for _, group := range groupIDs {
		if res.WasSkipped(group) {
			logrus.WithFields(logrus.Fields{
				"phase":   "seed",
				"group":   group,
				"outcome": "skipped",
				"state":   res.States[group],
			}).Info("group offsets skipped")
			continue
		}
		seededPartitions := 0
		for _, ids := range parts {
			seededPartitions += len(ids)
		}
		logrus.WithFields(logrus.Fields{
			"phase":      "seed",
			"group":      group,
			"outcome":    "seeded",
			"partitions": seededPartitions,
		}).Info("group offsets seeded")
	}
	logrus.WithFields(logrus.Fields{
		"phase":   "seed",
		"seeded":  len(res.Seeded),
		"skipped": len(res.Skipped),
	}).Info("seed summary")
	if len(res.Seeded) == 0 && len(res.Skipped) > 0 {
		logrus.Infof("all %d override consumer groups were already active — nothing seeded this run (re-sync no-op)", len(res.Skipped))
	}

	// Phase E: verify.
	reports, err := groups.Verify(ctx, client, addr, groupIDs, parts, res, kafkaops.DefaultRetryConfig())
	if err != nil {
		return err
	}
	for _, report := range reports {
		if len(report.Missing) == 0 {
			logrus.WithFields(logrus.Fields{
				"phase":   "verify",
				"group":   report.Group,
				"outcome": "ok",
				"topics":  report.Total,
			}).Info("group offsets verified")
			continue
		}
		fields := logrus.Fields{
			"group":   report.Group,
			"missing": len(report.Missing),
			"of":      report.Total,
			"topics":  truncatedTopicList(report.Missing),
		}
		if len(report.Missing) > 10 {
			fields["more"] = len(report.Missing) - 10
		}
		logrus.WithFields(fields).Warn("group has topics with no committed offset")
	}
	logrus.WithFields(logrus.Fields{"phase": "verify"}).Info("verify ok")

	return nil
}

// truncatedTopicList joins up to the first 10 names in names with ", ".
// FR-4.4 bounds the log line's length independently of how many topics a
// group is actually missing.
func truncatedTopicList(names []string) string {
	if len(names) > 10 {
		names = names[:10]
	}
	return strings.Join(names, ", ")
}
