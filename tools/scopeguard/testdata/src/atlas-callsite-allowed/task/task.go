// Package task exercises the call-site allowlist: this WithoutTenantFilter
// call site is present in the test-only allowlist TestAnalyzerAllowlisted
// installs, so no diagnostic is expected here even though the same call
// shape is a violation in atlas-callsite/scheduler.
package task

import (
	"context"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
)

func Run(ctx context.Context) context.Context {
	return database.WithoutTenantFilter(ctx)
}
