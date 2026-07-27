//go:build test

package saga

import (
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// SetEmitConversationRewardNoticeForTest swaps the underlying emit function
// and returns the previous one for restoration. Compiled only with
// -tags=test — production code cannot reach this seam.
func SetEmitConversationRewardNoticeForTest(fn func(logrus.FieldLogger, context.Context, uint32, string, uint32, uint32) error) func(logrus.FieldLogger, context.Context, uint32, string, uint32, uint32) error {
	prev := emitConversationRewardNoticeFn
	emitConversationRewardNoticeFn = fn
	return prev
}

// SetEmitSagaFailedForTest swaps the underlying Failed-emission function and
// returns the previous one for restoration. Compiled only with -tags=test.
func SetEmitSagaFailedForTest(fn func(logrus.FieldLogger, context.Context, uuid.UUID, string, uint32, uint32, string, string, string) error) func(logrus.FieldLogger, context.Context, uuid.UUID, string, uint32, uint32, string, string, string) error {
	prev := emitSagaFailedByIdsFn
	emitSagaFailedByIdsFn = fn
	return prev
}

// SetEmitMtsSagaFailedForTest swaps the underlying MTS Failed-emission function
// (which carries characterId + MtsKind) and returns the previous one for
// restoration. Compiled only with -tags=test.
func SetEmitMtsSagaFailedForTest(fn func(logrus.FieldLogger, context.Context, uuid.UUID, string, uint32, string, string, string, string) error) func(logrus.FieldLogger, context.Context, uuid.UUID, string, uint32, string, string, string, string) error {
	prev := emitMtsSagaFailedFn
	emitMtsSagaFailedFn = fn
	return prev
}
