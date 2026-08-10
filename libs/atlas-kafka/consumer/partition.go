package consumer

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
)

// drainTimeout bounds how long a partition loop waits for in-flight handlers
// when its generation ends. It MUST stay well below kafka-go's
// defaultRebalanceTimeout of 30s (consumergroup.go:47) — a slow drain holds
// up the whole group's rebalance, which is exactly the stall this task
// removes. 5s matches kafka-go's defaultTimeout (consumergroup.go:63).
// Handlers still running at the deadline are abandoned uncommitted and are
// redelivered in the next generation: at-least-once, erring toward
// duplication rather than loss (risks.md R1).
const drainTimeout = 5 * time.Second

// runPartition owns one assigned partition for the lifetime of one
// generation: create a positioned reader → fetch/handle/commit → on any
// recoverable error rebuild THAT READER ONLY → repeat.
//
// It must not return until the generation or the process is shutting down.
// kafka-go ends the generation as soon as any function passed to
// Generation.Start returns (consumergroup.go:387-405), so returning early
// would rejoin the group and rebalance every topic in it — the failure mode
// this task exists to remove (design §1.1).
func (c *Consumer) runPartition(l logrus.FieldLogger, ctx context.Context, gctx context.Context, gen Generation, pa kafka.PartitionAssignment) {
	pl := l.WithField("partition", pa.ID)

	// pctx is cancelled by EITHER the process context or the generation
	// context. context.AfterFunc gives us that without a goroutine of our own.
	pctx, cancel := context.WithCancel(ctx)
	defer cancel()
	stop := context.AfterFunc(gctx, cancel)
	defer stop()

	st := c.partitionStateFor(pa.ID)
	cur := newCursor()
	var inflight sync.WaitGroup

	commit := func(offset int64) error {
		return gen.CommitOffsets(map[string]map[int]int64{
			c.topic: {pa.ID: offset},
		})
	}

	offset := pa.Offset
	for attempt := 0; ; attempt++ {
		if pctx.Err() != nil {
			break
		}

		rd := c.prp(c.partitionReaderConfig(pa.ID), offset)
		c.onReaderCreated(pa.ID, attempt)
		if attempt == 0 {
			pl.Infof("Start consuming topic [%s] partition %d from offset %d.", c.topic, pa.ID, offset)
		} else {
			pl.Infof("Rebuilt partition reader for topic [%s] partition %d from offset %d (attempt %d).", c.topic, pa.ID, offset, attempt)
		}

		err := c.runPartitionFetchLoop(pl, pctx, pa.ID, rd, cur, &inflight, commit)
		if cerr := rd.Close(); cerr != nil {
			pl.WithError(cerr).Debugf("Error closing partition reader during rebuild.")
		}

		if pctx.Err() != nil {
			break
		}

		c.recordError(err)
		pl.WithError(err).Warnf("Partition reader for topic [%s] partition %d exited; rebuilding after backoff.", c.topic, pa.ID)

		// Let in-flight handlers finish and commit what they can before
		// choosing a resume offset, so the rebuilt reader re-reads only
		// genuinely uncommitted messages.
		c.quiesce(pl, &inflight, cur, commit)
		offset = cur.resumeOffset(pa.Offset)
		cur.reset()

		wait := st.backoff.next()
		select {
		case <-pctx.Done():
		case <-time.After(wait):
			c.recordBackoff(wait)
		}
	}

	c.quiesce(pl, &inflight, cur, commit)
	pl.Debugf("Partition loop for topic [%s] partition %d stopped.", c.topic, pa.ID)
}

// runPartitionFetchLoop is the task-136 fetch loop rebased onto a single
// partition: the per-call fetchTimeout is still a liveness tick, an expired
// deadline on a reader that is still making fetch attempts is still a healthy
// idle tick, and maxConsecutiveTimeouts no-progress ticks still return
// errFetchWedged. The only substitutions are that offsets commit through the
// generation-scoped cursor rather than reader.CommitMessages, and that the
// loop's cancel signal is pctx (process OR generation).
func (c *Consumer) runPartitionFetchLoop(l logrus.FieldLogger, pctx context.Context, partition int, rd KafkaReader, cur *cursor, wg *sync.WaitGroup, commit func(offset int64) error) error {
	maxQueue := 4 * c.maxInFlight
	var sem chan struct{}
	if c.maxInFlight > 1 {
		sem = make(chan struct{}, c.maxInFlight)
	}

	for {
		if pctx.Err() != nil {
			return pctx.Err()
		}

		// Back-pressure: stop fetching when the queue is full (the head is
		// stuck on a failure). Wait one tick, then try to advance.
		if c.maxInFlight > 1 && cur.len() >= maxQueue {
			select {
			case <-pctx.Done():
				return pctx.Err()
			case <-time.After(c.fetchTimeout):
			}
			c.tryAdvance(l, cur, commit)
			continue
		}

		fetchCtx, cancelFetch := context.WithTimeout(pctx, c.fetchTimeout)
		fetchStart := time.Now()
		msg, err := rd.FetchMessage(fetchCtx)
		cancelFetch()
		c.recordFetchDuration(time.Since(fetchStart))

		if err != nil {
			if pctx.Err() != nil || errors.Is(err, context.Canceled) {
				return err
			}
			if errors.Is(err, context.DeadlineExceeded) {
				if werr := c.handleFetchDeadline(l, rd, partition); werr != nil {
					return werr
				}
				c.tryAdvance(l, cur, commit)
				continue
			}
			return err
		}

		c.recordFetch(partition)
		l.Debugf("Message received %s.", string(msg.Value))

		in := cur.track(msg.Offset)

		if c.maxInFlight <= 1 {
			handlerStart := time.Now()
			ok := c.processMessage(l, pctx, msg)
			c.recordHandlerDuration(time.Since(handlerStart))
			in.mark(ok)
			c.tryAdvance(l, cur, commit)
			continue
		}

		sem <- struct{}{}
		m := msg
		tracked := in
		wg.Add(1)
		routine.Go(l, pctx, func(hctx context.Context) {
			defer wg.Done()
			defer func() { <-sem }()
			handlerStart := time.Now()
			ok := c.processMessage(l, hctx, m)
			c.recordHandlerDuration(time.Since(handlerStart))
			tracked.mark(ok)
			c.tryAdvance(l, cur, commit)
		})
	}
}

// tryAdvance advances the commit cursor, logging a commit failure at Warn.
// The cursor retains its high-water mark on failure, so the next call retries
// the same offset (FR-1.3) — the message is never silently skipped.
func (c *Consumer) tryAdvance(l logrus.FieldLogger, cur *cursor, commit func(offset int64) error) {
	if err := cur.advance(commit); err != nil {
		l.WithError(err).Warnf("Could not commit message offset for topic [%s]; it may be redelivered.", c.topic)
	}
}

// quiesce waits up to drainTimeout for in-flight handlers, then makes a final
// commit attempt (FR-1.6). Handlers still running at the deadline are
// abandoned uncommitted and redelivered.
func (c *Consumer) quiesce(l logrus.FieldLogger, wg *sync.WaitGroup, cur *cursor, commit func(offset int64) error) {
	done := make(chan struct{})
	routine.Go(l, context.Background(), func(_ context.Context) {
		wg.Wait()
		close(done)
	})

	select {
	case <-done:
	case <-time.After(drainTimeout):
		l.Warnf("Drain timeout (%v) elapsed on topic [%s]; abandoning in-flight handlers uncommitted for redelivery.", drainTimeout, c.topic)
	}

	c.tryAdvance(l, cur, commit)
}
