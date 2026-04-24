//go:build test

package saga

import (
	"context"

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
