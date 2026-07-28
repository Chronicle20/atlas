package model

import (
	"errors"
	"testing"
)

func TestErrDecoratorSuccessEnriches(t *testing.T) {
	d := ErrDecorator(
		func(m int) (int, error) { return m + 1, nil },
		func(m int, err error) { t.Fatalf("onErr must not be called on success") },
	)
	if got := d(41); got != 42 {
		t.Fatalf("expected 42, got %d", got)
	}
}

func TestErrDecoratorFailureDegradesLoudly(t *testing.T) {
	boom := errors.New("boom")
	var gotM int
	var gotErr error
	d := ErrDecorator(
		func(m int) (int, error) { return 0, boom },
		func(m int, err error) { gotM, gotErr = m, err },
	)
	if got := d(41); got != 41 {
		t.Fatalf("expected un-enriched 41, got %d", got)
	}
	if gotM != 41 || !errors.Is(gotErr, boom) {
		t.Fatalf("onErr not invoked with original model and cause: m=%d err=%v", gotM, gotErr)
	}
}
