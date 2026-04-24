//go:build test

package character

import (
	"context"

	character2 "atlas-saga-orchestrator/kafka/message/character"

	"github.com/sirupsen/logrus"
)

// HandleCharacterStatChangedEventForTest re-exports handleCharacterStatChangedEvent
// for cross-package integration tests. Compiled only with -tags=test.
func HandleCharacterStatChangedEventForTest(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventStatChangedBody]) {
	handleCharacterStatChangedEvent(l, ctx, e)
}

// HandleCharacterJobChangedEventForTest re-exports handleCharacterJobChangedEvent
// for cross-package integration tests. Compiled only with -tags=test.
func HandleCharacterJobChangedEventForTest(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.JobChangedStatusEventBody]) {
	handleCharacterJobChangedEvent(l, ctx, e)
}
