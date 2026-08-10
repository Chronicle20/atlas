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
