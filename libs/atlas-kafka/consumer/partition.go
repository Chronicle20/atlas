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
//
// A package-level var (not a const) so tests can drive the drain-timeout-
// exceeded branch of quiesce deterministically without waiting 5s; production
// code never reassigns it, so the default stays exactly 5s.
var drainTimeout = 5 * time.Second

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

	commit := func(offset int64) error {
		return gen.CommitOffsets(map[string]map[int]int64{
			c.topic: {pa.ID: offset},
		})
	}

	// wg tracks the in-flight handler(s) of the CURRENT attempt only. It is
	// reassigned to a fresh WaitGroup on every reader rebuild
	// (runPartitionFetchLoop allocates its own) so a straggler left behind by
	// a timed-out quiesce can never observe a later attempt's wg.Add — the
	// stdlib's documented-unsafe "Add concurrent with Wait" pattern, which
	// panics (task-209 review finding 2).
	wg := &sync.WaitGroup{}

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

		var err error
		wg, err = c.runPartitionFetchLoop(pl, pctx, pa.ID, rd, cur, commit)
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
		c.quiesce(pl, wg, cur, commit)
		offset = cur.resumeOffset(pa.Offset)
		cur.reset()

		wait := st.backoff.next()
		select {
		case <-pctx.Done():
		case <-time.After(wait):
			c.recordBackoff(wait)
		}
	}

	c.quiesce(pl, wg, cur, commit)
	pl.Debugf("Partition loop for topic [%s] partition %d stopped.", c.topic, pa.ID)
}

// runPartitionFetchLoop is the task-136 fetch loop rebased onto a single
// partition: the per-call fetchTimeout is still a liveness tick, an expired
// deadline on a reader that is still making fetch attempts is still a healthy
// idle tick, and maxConsecutiveTimeouts no-progress ticks still return
// errFetchWedged. The only substitutions are that offsets commit through the
// generation-scoped cursor rather than reader.CommitMessages, and that the
// loop's cancel signal is pctx (process OR generation).
//
// It owns a fresh *sync.WaitGroup for the lifetime of THIS attempt only and
// returns it to the caller, who quiesces against it. A partition-lifetime
// WaitGroup would let a straggler abandoned by one attempt's quiesce timeout
// still be outstanding when a later attempt's wg.Add fires — the stdlib's
// documented-unsafe "Add concurrent with Wait" race, which panics
// (task-209 review finding 2).
//
// Both the serial (maxInFlight<=1) and parallel branches run the handler
// through wg/routine.Go, so quiesce has something to bound in either
// configuration (task-209 review finding 1: the serial path used to call
// processMessage synchronously, inline in this loop, which meant a hung
// handler blocked this function from ever returning — quiesce was
// unreachable and the generation's Start function never returned). The
// serial branch stays serial: the loop does not fetch the next message until
// the current handler's completion is observed (or pctx is cancelled), so at
// most one handler is ever in flight; only the *blocking structure* changed,
// making a stuck handler abandonable at the drain deadline instead of
// wedging the loop forever.
func (c *Consumer) runPartitionFetchLoop(l logrus.FieldLogger, pctx context.Context, partition int, rd KafkaReader, cur *cursor, commit func(offset int64) error) (*sync.WaitGroup, error) {
	wg := &sync.WaitGroup{}
	maxQueue := 4 * c.maxInFlight
	var sem chan struct{}
	if c.maxInFlight > 1 {
		sem = make(chan struct{}, c.maxInFlight)
	}

	for {
		if pctx.Err() != nil {
			return wg, pctx.Err()
		}

		// Back-pressure: stop fetching when the queue is full (the head is
		// stuck on a failure). Wait one tick, then try to advance.
		if c.maxInFlight > 1 && cur.len() >= maxQueue {
			select {
			case <-pctx.Done():
				return wg, pctx.Err()
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
				return wg, err
			}
			if errors.Is(err, context.DeadlineExceeded) {
				if werr := c.handleFetchDeadline(l, rd, partition); werr != nil {
					return wg, werr
				}
				c.tryAdvance(l, cur, commit)
				continue
			}
			return wg, err
		}

		c.recordFetch(partition)
		l.Debugf("Message received %s.", string(msg.Value))

		in := cur.track(msg.Offset)
		m := msg

		if c.maxInFlight <= 1 {
			// Bounded-serial: run the handler on its own goroutine (tracked
			// by wg, so quiesce can abandon it at the drain deadline), but
			// don't advance to the next fetch until it finishes or pctx is
			// cancelled — exactly one handler in flight at a time, same as
			// the synchronous call this replaces.
			handlerDone := make(chan struct{})
			wg.Add(1)
			routine.Go(l, pctx, func(hctx context.Context) {
				defer wg.Done()
				defer close(handlerDone)
				handlerStart := time.Now()
				ok := c.processMessage(l, hctx, m)
				c.recordHandlerDuration(time.Since(handlerStart))
				in.mark(ok)
				c.tryAdvance(l, cur, commit)
			})
			select {
			case <-handlerDone:
			case <-pctx.Done():
				return wg, pctx.Err()
			}
			continue
		}

		sem <- struct{}{}
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
