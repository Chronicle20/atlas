package asset

import (
	"context"

	asset2 "atlas-saga-orchestrator/kafka/message/asset"

	"github.com/sirupsen/logrus"
)

// HandleAssetCreatedEventForTest re-exports handleAssetCreatedEvent for
// cross-package integration tests. Test-only — only compiled during `go test`.
func HandleAssetCreatedEventForTest(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.CreatedStatusEventBody]) {
	handleAssetCreatedEvent(l, ctx, e)
}
