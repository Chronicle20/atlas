package consumer

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
)

// This file holds the LEGACY consumer engine: one *kafka.Reader per Consumer
// with GroupID set, so kafka-go owns partition assignment, offset commits and
// rebalance handling internally. It is selected by
// KAFKA_CONSUMER_ENGINE=reader and is retained for one release as the
// rollback path for task-209 (see engine.go). Do not add behaviour here —
// the supported engine is engine_group.go.

// errFetchWedged is returned from runFetchLoop when FetchMessage has hit
// its deadline maxConsecutiveTimeouts times in a row without a successful
// fetch in between. The outer start loop treats it identically to any
// other recreate-eligible error: close reader, backoff, rebuild.
var errFetchWedged = errors.New("consumer fetch wedged: exceeded consecutive timeouts")

// startReaderEngine owns the full reader lifecycle: create reader → run fetch
// loop → close reader → backoff → repeat, until the parent context is
// canceled. Only a canceled parent ctx means shutdown; every other error
// (including io.EOF) flows through the backoff + recreate path.
func (c *Consumer) startReaderEngine(l logrus.FieldLogger, ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	l.Infof("Creating topic consumer.")

	backoff := newFetchBackoff()
	for attempt := 0; ; attempt++ {
		if ctx.Err() != nil {
			l.Infof("Parent context canceled; shutting down topic consumer.")
			return
		}

		reader := c.rp(c.readerConfig)
		c.onReaderCreated(legacyPartition, attempt)
		if attempt == 0 {
			l.Infof("Start consuming topic.")
		} else {
			l.Infof("Recreated reader for topic (attempt %d).", attempt)
		}

		err := c.runFetchLoop(l, ctx, reader)
		if cerr := reader.Close(); cerr != nil {
			l.WithError(cerr).Debugf("Error closing reader during recreate.")
		}

		if ctx.Err() != nil || errors.Is(err, context.Canceled) {
			l.Infof("Topic consumer stopped.")
			return
		}

		c.recordError(err)
		l.WithError(err).Errorf("Fetcher exited; recreating reader after backoff.")
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

// runFetchLoop dispatches to the serial or parallel fetch loop depending on
// c.maxInFlight. Default (maxInFlight == 1) uses the serial path, which is
// bit-exact with the original implementation.
func (c *Consumer) runFetchLoop(l logrus.FieldLogger, ctx context.Context, reader KafkaReader) error {
	if c.maxInFlight <= 1 {
		return c.runFetchLoopSerial(l, ctx, reader)
	}
	return c.runFetchLoopParallel(l, ctx, reader)
}

// runFetchLoopSerial is the original single-goroutine fetch loop. It blocks
// until the reader errors or ctx is canceled.
//
// Each iteration runs FetchMessage under a per-call deadline
// (c.fetchTimeout) that acts as a liveness tick. An expiration on a reader
// that is still making fetch attempts (per Stats() deltas) is an idle
// tick — healthy, never a recreate. Only ticks with zero reader progress
// count toward c.maxConsecutiveTimeouts; at the threshold the loop returns
// errFetchWedged so the outer start loop closes and recreates the reader.
// A successful fetch resets the counter via recordFetch.
func (c *Consumer) runFetchLoopSerial(l logrus.FieldLogger, ctx context.Context, reader KafkaReader) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		fetchCtx, cancelFetch := context.WithTimeout(ctx, c.fetchTimeout)
		fetchStart := time.Now()
		msg, err := reader.FetchMessage(fetchCtx)
		cancelFetch()
		c.recordFetchDuration(time.Since(fetchStart))

		if err != nil {
			if ctx.Err() != nil || errors.Is(err, context.Canceled) {
				return err
			}
			if errors.Is(err, context.DeadlineExceeded) {
				if werr := c.handleFetchDeadline(l, reader, legacyPartition); werr != nil {
					return werr
				}
				continue
			}
			return err
		}

		c.recordFetch(legacyPartition)
		l.Debugf("Message received %s.", string(msg.Value))
		handlerStart := time.Now()
		ok := c.processMessage(l, ctx, msg)
		c.recordHandlerDuration(time.Since(handlerStart))
		if ok {
			if cerr := reader.CommitMessages(ctx, msg); cerr != nil {
				l.WithError(cerr).Warnf("Could not commit message offset, it may be redelivered.")
			}
		}
	}
}

// runFetchLoopParallel is an opt-in parallel fetch loop that uses a
// prefix-commit cursor to commit offsets in order even when handlers complete
// out of order. Up to c.maxInFlight handlers run concurrently; the in-flight
// queue is capped at 4*c.maxInFlight to bound memory growth when the head is
// stuck on a failing message.
//
// Commit semantics: only the highest contiguously-completed offset is
// committed. A failed handler (processMessage returning false) blocks the
// cursor — subsequent messages are not committed until the failed message is
// redelivered and succeeds (matching at-least-once semantics).
func (c *Consumer) runFetchLoopParallel(l logrus.FieldLogger, ctx context.Context, reader KafkaReader) error {
	type pending struct {
		msg  kafka.Message
		done atomic.Bool
		ok   atomic.Bool
	}

	maxQueue := 4 * c.maxInFlight
	sem := make(chan struct{}, c.maxInFlight)

	var qmu sync.Mutex // guards queue slice header
	var queue []*pending

	advanceCommit := func() {
		qmu.Lock()
		i := 0
		for i < len(queue) && queue[i].done.Load() && queue[i].ok.Load() {
			i++
		}
		if i == 0 {
			qmu.Unlock()
			return
		}
		commitMsg := queue[i-1].msg
		queue = queue[i:]
		qmu.Unlock()
		if cerr := reader.CommitMessages(ctx, commitMsg); cerr != nil {
			l.WithError(cerr).Warn("Could not commit message offset; may be redelivered.")
		}
	}

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Back-pressure: stop fetching when the queue is full (head stuck on a
		// failure). Wait one fetchTimeout tick then retry; advanceCommit may
		// have moved the cursor by then.
		qmu.Lock()
		full := len(queue) >= maxQueue
		qmu.Unlock()
		if full {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.fetchTimeout):
			}
			advanceCommit()
			continue
		}

		fetchCtx, cancelFetch := context.WithTimeout(ctx, c.fetchTimeout)
		fetchStart := time.Now()
		msg, err := reader.FetchMessage(fetchCtx)
		cancelFetch()
		c.recordFetchDuration(time.Since(fetchStart))

		if err != nil {
			if ctx.Err() != nil || errors.Is(err, context.Canceled) {
				return err
			}
			if errors.Is(err, context.DeadlineExceeded) {
				if werr := c.handleFetchDeadline(l, reader, legacyPartition); werr != nil {
					return werr
				}
				// In-flight goroutines may have completed; try to advance.
				advanceCommit()
				continue
			}
			return err
		}

		c.recordFetch(legacyPartition)
		l.Debugf("Message received %s.", string(msg.Value))

		pm := &pending{msg: msg}
		qmu.Lock()
		queue = append(queue, pm)
		qmu.Unlock()

		sem <- struct{}{}
		p := pm
		routine.Go(l, ctx, func(_ context.Context) {
			defer func() { <-sem }()
			handlerStart := time.Now()
			ok := c.processMessage(l, ctx, p.msg)
			c.recordHandlerDuration(time.Since(handlerStart))
			p.ok.Store(ok)
			p.done.Store(true)
			advanceCommit()
		})
	}
}
