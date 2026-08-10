package consumer

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
)

// errFakeGroupClosed stands in for kafka.ErrGroupClosed.
var errFakeGroupClosed = errors.New("fake group closed")

type committedOffset struct {
	Topic     string
	Partition int
	Offset    int64
}

// fakeGeneration is a scriptable Generation. Start runs fn on a routine.Go
// with a generation-scoped context, exactly like kafka-go: the first fn to
// return ends the generation.
type fakeGeneration struct {
	id          int32
	assignments map[string][]kafka.PartitionAssignment

	mu        sync.Mutex
	committed []committedOffset
	commitErr error
	closed    bool
	done      chan struct{}
	wg        sync.WaitGroup
}

func newFakeGeneration(id int32, assignments map[string][]kafka.PartitionAssignment) *fakeGeneration {
	return &fakeGeneration{
		id:          id,
		assignments: assignments,
		done:        make(chan struct{}),
	}
}

func (g *fakeGeneration) ID() int32 { return g.id }

func (g *fakeGeneration) Assignments() map[string][]kafka.PartitionAssignment {
	return g.assignments
}

func (g *fakeGeneration) Start(fn func(ctx context.Context)) {
	l, _ := test.NewNullLogger()
	g.wg.Add(1)
	routine.Go(l, context.Background(), func(_ context.Context) {
		defer g.wg.Done()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		stop := context.AfterFunc(genDoneContext(g), cancel)
		defer stop()
		fn(ctx)
		g.end()
	})
}

// genDoneContext adapts the generation's done channel to a context, mirroring
// kafka-go's genCtx (consumergroup.go:276-301).
func genDoneContext(g *fakeGeneration) context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	l, _ := test.NewNullLogger()
	routine.Go(l, context.Background(), func(_ context.Context) {
		<-g.done
		cancel()
	})
	return ctx
}

// end closes the generation, mirroring kafka-go's "first Start fn to return
// ends the generation" rule.
func (g *fakeGeneration) end() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.closed {
		g.closed = true
		close(g.done)
	}
}

func (g *fakeGeneration) ended() bool {
	select {
	case <-g.done:
		return true
	default:
		return false
	}
}

// wait blocks until every Start'd function has exited.
func (g *fakeGeneration) wait() { g.wg.Wait() }

func (g *fakeGeneration) setCommitErr(err error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.commitErr = err
}

func (g *fakeGeneration) CommitOffsets(offsets map[string]map[int]int64) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.commitErr != nil {
		return g.commitErr
	}
	for topic, parts := range offsets {
		for partition, offset := range parts {
			g.committed = append(g.committed, committedOffset{Topic: topic, Partition: partition, Offset: offset})
		}
	}
	return nil
}

func (g *fakeGeneration) commits() []committedOffset {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := make([]committedOffset, len(g.committed))
	copy(out, g.committed)
	return out
}

// fakeGroup hands out scripted generations in order, then blocks on ctx (a
// real ConsumerGroup.Next parks until the next rebalance).
type fakeGroup struct {
	mu     sync.Mutex
	gens   []*fakeGeneration
	next   int
	nextN  int
	closed bool
}

func newFakeGroup(gens ...*fakeGeneration) *fakeGroup {
	return &fakeGroup{gens: gens}
}

func (g *fakeGroup) Next(ctx context.Context) (Generation, error) {
	g.mu.Lock()
	g.nextN++
	if g.next > 0 {
		prev := g.gens[g.next-1]
		g.mu.Unlock()
		// Mirror kafka-go: Next never returns a new generation until the
		// previous one has ended.
		select {
		case <-prev.done:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		g.mu.Lock()
	}
	if g.next >= len(g.gens) {
		g.mu.Unlock()
		<-ctx.Done()
		return nil, ctx.Err()
	}
	gen := g.gens[g.next]
	g.next++
	g.mu.Unlock()
	return gen, nil
}

// nextCalls reports how many times Next has been entered. The FR-2.3 test
// uses it to prove a wedged partition reader rebuilds WITHOUT ending the
// generation.
func (g *fakeGroup) nextCalls() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.nextN
}

func (g *fakeGroup) Close() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.closed = true
	return nil
}

var (
	_ Group      = (*fakeGroup)(nil)
	_ Generation = (*fakeGeneration)(nil)
)

// silentLogger returns a logger whose entries the caller can inspect.
func silentLogger() (*logrus.Logger, *test.Hook) {
	return test.NewNullLogger()
}

// waitFor polls cond until it holds or 3s elapse.
func waitFor(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal(msg)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// hasLogContaining reports whether any captured entry's message contains sub.
func hasLogContaining(hook *test.Hook, sub string) bool {
	for _, e := range hook.AllEntries() {
		if strings.Contains(e.Message, sub) {
			return true
		}
	}
	return false
}
