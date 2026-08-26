package consumer

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// startGroupEngine owns the consumer-group lifecycle: join → consume
// generations → on failure, back off and rejoin. Only a cancelled parent
// context means shutdown.
func (c *Consumer) startGroupEngine(l logrus.FieldLogger, ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	l.Infof("Creating topic consumer (engine consumergroup).")

	backoff := newFetchBackoff()
	for {
		if ctx.Err() != nil {
			l.Infof("Parent context canceled; shutting down topic consumer.")
			return
		}

		// Never join a group for a topic that does not exist (task-267). This
		// runs inside the per-consumer goroutine launched by AddConsumer, and
		// AFTER the wg.Add(1) above — hence awaitTopic's mandatory ctx select.
		if !c.awaitTopic(l, ctx) {
			return
		}

		grp, err := c.gp(c.groupConfig())
		if err == nil {
			err = c.runGenerations(l, ctx, grp)
			if cerr := grp.Close(); cerr != nil {
				l.WithError(cerr).Debugf("Error closing consumer group.")
			}
		}

		if ctx.Err() != nil || errors.Is(err, context.Canceled) {
			l.Infof("Topic consumer stopped.")
			return
		}

		c.recordError(err)
		l.WithError(err).Errorf("Consumer group for topic [%s] exited; rejoining after backoff.", c.topic)
		wait := backoff.next()
		select {
		case <-ctx.Done():
			l.Infof("Topic consumer stopped during backoff.")
			return
		case <-time.After(wait):
			c.recordBackoff(wait)
		}
	}
}

// awaitTopic blocks until this consumer's topic definitely has at least one
// partition, or the lookup is indeterminate, or ctx is cancelled. It returns
// false ONLY for cancellation.
//
// This is the cold-start half of task-267. kafka-go's leader forgives a
// missing topic when computing assignments (consumergroup.go:1010-1049) and
// hands the member an EMPTY assignment; its partition watcher is supposed to
// rebalance when the topic appears, but the watcher's startup read returns on
// any error including UnknownTopicOrPartition (consumergroup.go:512-518) — the
// very error its own ticker branch tolerates. So a member that joins for a
// missing topic is permanently deaf. The remedy is to not be a member: a
// non-member cannot go deaf, and cannot destabilise the group for the members
// that ARE working.
//
// FR-2.5: only ErrTopicNotFound — a DEFINITE negative — parks here. Any other
// error is indeterminate and joins immediately, exactly as today, because a
// broker outage must never hold a consumer out of its group.
func (c *Consumer) awaitTopic(l logrus.FieldLogger, ctx context.Context) bool {
	if c.pcp == nil {
		return true
	}

	backoff := newFetchBackoff()
	waiting := false
	var lastWarn time.Time

	for {
		if ctx.Err() != nil {
			return false
		}

		lctx, cancel := context.WithTimeout(ctx, topicMetadataTimeout)
		count, err := c.pcp(lctx, c.brokers, c.topic)
		cancel()

		switch {
		case err != nil && !errors.Is(err, ErrTopicNotFound):
			l.WithError(err).Debugf("Partition-count lookup for topic [%s] was indeterminate; joining group [%s] anyway.", c.topic, c.groupId)
			return true
		case err == nil && count >= 1:
			if waiting {
				l.Infof("Topic [%s] now has %d partition(s); joining group [%s].", c.topic, count, c.groupId)
			}
			return true
		}

		now := time.Now()
		if !waiting || now.Sub(lastWarn) >= topicMissingWarnInterval {
			l.Warnf("Topic [%s] does not exist or has no partitions; consumer will not join group [%s] until it appears.", c.topic, c.groupId)
			lastWarn = now
			c.recordTopicMissing()
		} else {
			l.Debugf("Topic [%s] is still absent; re-polling before joining group [%s].", c.topic, c.groupId)
		}
		waiting = true

		wait := backoff.next()
		select {
		case <-ctx.Done():
			l.Infof("Topic consumer stopped while waiting for topic [%s] to appear.", c.topic)
			return false
		case <-time.After(wait):
			c.recordBackoff(wait)
		}
	}
}

// runGenerations consumes generations until the group closes or ctx is done.
//
// Next blocks until the current generation ends (kafka-go consumergroup.go:701
// receives on the unbuffered cg.next, which is fed only after <-gen.done in
// nextGeneration), so the zero-assignment branch below PARKS — it does not
// spin.
func (c *Consumer) runGenerations(l logrus.FieldLogger, ctx context.Context, grp Group) error {
	for {
		gen, err := grp.Next(ctx)
		if err != nil {
			return err
		}

		parts := gen.Assignments()[c.topic]
		ids := partitionIDs(parts)
		c.onAssignment(gen.ID(), ids)

		if len(parts) == 0 {
			// HEALTHY-IDLE (FR-2.1/2.2/2.3/2.5). Every atlas-main topic has a
			// single partition while services run replicas: 2, so exactly one
			// member of every group lands here permanently. It holds nothing,
			// so it must not tick, must not warn, and must not recreate — the
			// recreate is what rejoins the group and rebalances every OTHER
			// topic in it. Heartbeats are generation-scoped and started
			// unconditionally by kafka-go (consumergroup.go:840), so doing
			// nothing here keeps this member alive and eligible for the next
			// assignment. Debug, not Warn: this state is expected (FR-4.3).
			l.Debugf("Consumer for topic [%s] (group [%s]) holds no partition assignment in generation %d; healthy-idle.",
				c.topic, c.groupId, gen.ID())
			continue
		}

		l.Infof("Consumer for topic [%s] (group [%s]) assigned partitions %v in generation %d.",
			c.topic, c.groupId, ids, gen.ID())

		for _, pa := range parts {
			assignment := pa
			gen.Start(func(gctx context.Context) {
				c.runPartition(l, ctx, gctx, gen, assignment)
			})
		}
	}
}

// partitionIDs extracts a sorted partition-id list. Always non-nil so
// Snapshot.AssignedPartitions serialises as [] rather than null.
func partitionIDs(pas []kafka.PartitionAssignment) []int {
	out := make([]int, 0, len(pas))
	for _, pa := range pas {
		out = append(out, pa.ID)
	}
	sort.Ints(out)
	return out
}
