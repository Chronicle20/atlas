package async

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"sync"
	"testing"
	"time"
)

func TestAsyncSlice(t *testing.T) {
	items := []uint32{1, 2, 3, 4, 5}

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		results, err := AwaitSlice(model.SliceMap(AsyncTestTransformer)(model.FixedProvider(items))())()
		if err != nil {
			t.Fatal(err)
		}
		for _, result := range results {
			found := false
			for _, item := range items {
				if item == result {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("Invalid item.")
			}
		}
		t.Logf("Here 2")
	}()
	wg.Wait()
	t.Logf("Here 1")
}

func AsyncTestTransformer(m uint32) (Provider[uint32], error) {
	return func(ctx context.Context, rchan chan uint32, echan chan error) {
		time.Sleep(time.Duration(50) * time.Millisecond)
		rchan <- m
	}, nil
}
