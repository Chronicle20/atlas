package producer

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus/hooks/test"
)

// fakeWriter is a Writer implementation used by manager tests. Tracks how many
// times Close was called and supports an injected close error.
type fakeWriter struct {
	topicName string
	closeErr  error
	closes    int32
}

func (f *fakeWriter) Topic() string { return f.topicName }
func (f *fakeWriter) WriteMessages(_ context.Context, _ ...kafka.Message) error {
	return nil
}
func (f *fakeWriter) Close() error {
	atomic.AddInt32(&f.closes, 1)
	return f.closeErr
}

func TestManager_LazyCreate(t *testing.T) {
	ResetInstance()
	var built int32
	factory := func(topicName string) Writer {
		atomic.AddInt32(&built, 1)
		return &fakeWriter{topicName: topicName}
	}
	m := GetManager(ConfigWriterFactory(factory))
	l, _ := test.NewNullLogger()

	w1, err := m.Writer(l, "MY_TOPIC")
	if err != nil {
		t.Fatalf("first Writer call returned error: %v", err)
	}
	w2, err := m.Writer(l, "MY_TOPIC")
	if err != nil {
		t.Fatalf("second Writer call returned error: %v", err)
	}
	if w1 != w2 {
		t.Fatalf("expected same Writer instance on repeat lookup; got distinct pointers")
	}
	if got := atomic.LoadInt32(&built); got != 1 {
		t.Fatalf("factory should be called exactly once; got %d", got)
	}
}

// Suppress unused-import warning until later tests reference these.
var _ = sync.Once{}
var _ = errors.New

func TestManager_ConcurrentFirstTouch(t *testing.T) {
	ResetInstance()
	var built int32
	factory := func(topicName string) Writer {
		atomic.AddInt32(&built, 1)
		return &fakeWriter{topicName: topicName}
	}
	m := GetManager(ConfigWriterFactory(factory))
	l, _ := test.NewNullLogger()

	const goroutines = 64
	results := make([]Writer, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	start := make(chan struct{})
	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			<-start
			w, err := m.Writer(l, "RACE_TOPIC")
			if err != nil {
				t.Errorf("goroutine %d: %v", i, err)
				return
			}
			results[i] = w
		}()
	}
	close(start)
	wg.Wait()

	if got := atomic.LoadInt32(&built); got != 1 {
		t.Fatalf("factory should be called exactly once across %d racers; got %d", goroutines, got)
	}
	for i := 1; i < goroutines; i++ {
		if results[i] != results[0] {
			t.Fatalf("goroutine %d returned a different Writer than goroutine 0", i)
		}
	}
}

func TestManager_IdempotentClose(t *testing.T) {
	ResetInstance()
	fw := &fakeWriter{topicName: "T"}
	factory := func(topicName string) Writer { return fw }
	m := GetManager(ConfigWriterFactory(factory))
	l, _ := test.NewNullLogger()

	if _, err := m.Writer(l, "ANY_TOPIC"); err != nil {
		t.Fatalf("Writer: %v", err)
	}
	if err := m.Close(l); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := m.Close(l); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	if got := atomic.LoadInt32(&fw.closes); got != 1 {
		t.Fatalf("underlying Writer.Close should be called exactly once; got %d", got)
	}
}

func TestManager_CloseErrorsDoNotShortCircuit(t *testing.T) {
	ResetInstance()
	writers := map[string]*fakeWriter{
		"A": {topicName: "A"},
		"B": {topicName: "B", closeErr: errors.New("boom")},
		"C": {topicName: "C"},
	}
	factory := func(topicName string) Writer { return writers[topicName] }
	m := GetManager(ConfigWriterFactory(factory))
	l, _ := test.NewNullLogger()

	for _, k := range []string{"A", "B", "C"} {
		if _, err := m.Writer(l, k); err != nil {
			t.Fatalf("Writer(%s): %v", k, err)
		}
	}
	if err := m.Close(l); err != nil {
		t.Fatalf("Close: %v", err)
	}
	for k, w := range writers {
		if got := atomic.LoadInt32(&w.closes); got != 1 {
			t.Fatalf("writer %s closed %d times; want 1", k, got)
		}
	}
}
