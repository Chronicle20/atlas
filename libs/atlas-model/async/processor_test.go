package async

import (
	"context"
	"errors"
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
		ctx := context.WithValue(context.Background(), "key", "value")
		results, err := AwaitSlice(model.SliceMap(AsyncTestTransformer)(model.FixedProvider(items))(), SetContext(ctx))()
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
	}()
	wg.Wait()
}

func AsyncTestTransformer(m uint32) (Provider[uint32], error) {
	return func(ctx context.Context, rchan chan uint32, echan chan error) {
		time.Sleep(time.Duration(50) * time.Millisecond)

		if ctx.Value("key") != "value" {
			echan <- errors.New("invalid context")
		}

		rchan <- m
	}, nil
}

func TestAsyncSliceErrorHandling(t *testing.T) {
	// Test that errors are properly returned without double-read issues
	items := []uint32{1, 2, 3}
	expectedError := errors.New("test error")

	ctx := context.Background()
	results, err := AwaitSlice(model.SliceMap(func(m uint32) (Provider[uint32], error) {
		return func(ctx context.Context, rchan chan uint32, echan chan error) {
			if m == 2 {
				echan <- expectedError
				return
			}
			rchan <- m
		}, nil
	})(model.FixedProvider(items))(), SetContext(ctx))()

	if err == nil {
		t.Fatal("Expected error but got none")
	}
	if err.Error() != expectedError.Error() {
		t.Fatalf("Expected error %q but got %q", expectedError.Error(), err.Error())
	}
	if results != nil {
		t.Fatal("Expected nil results on error")
	}
}
