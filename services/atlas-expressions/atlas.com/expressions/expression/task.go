package expression

import (
	"atlas-expressions/kafka/message/expression"
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const RevertTaskName = "expression_revert_task"

type RevertTask struct {
	l          logrus.FieldLogger
	interval   time.Duration
	envContext func(context.Context) context.Context
}

// NewRevertTask builds the expired-expression revert sweep. envContext
// originates this pod's own environment identity (env.Self()) onto each
// expired expression's per-tenant context before the revert event is
// produced -- expression is outside env-domain-guard's permitted atlas-env
// import list, so the caller (main.go) threads this in as a plain function
// value rather than the package importing atlas-env itself. Without it,
// decide() sees an empty ENVIRONMENT header and fails open per FR-1.8:
// every live deployment, not just this pod's, would revert the expression.
func NewRevertTask(l logrus.FieldLogger, interval time.Duration, envContext func(context.Context) context.Context) *RevertTask {
	l.Infof("Initializing expression revert task to run every %dms", interval.Milliseconds())
	return &RevertTask{l, interval, envContext}
}

func (e *RevertTask) Run(ctx context.Context) {
	sctx, span := otel.GetTracerProvider().Tracer("atlas-expressions").Start(ctx, RevertTaskName)
	defer span.End()

	processExpired(e.l, sctx, GetRegistry().popExpired(sctx), revertExpression, e.envContext)
}

// revertExpression emits the revert event for one expired expression. It is
// injected into processExpired so the pure sweep logic can be tested with a
// spy in place of the real Kafka producer call.
func revertExpression(l logrus.FieldLogger, ctx context.Context, exp Model) error {
	transactionId := uuid.New() // Generate a new transaction ID for each expired expression
	// Revert always restores the neutral face: expression 0 with no duration
	// and no item option (FR-3.7). The registry Model deliberately does not
	// persist the original duration/byItemOption, so there is nothing to
	// replay here.
	return producer.ProviderImpl(l)(ctx)(expression.EnvExpressionEvent)(expressionEventProvider(transactionId, exp.CharacterId(), exp.Field(), 0, 0, false))
}

// processExpired originates this pod's own environment identity onto each
// expired expression's per-tenant context before calling revert -- expiry
// bookkeeping is per-character lifecycle state driven by real gameplay, so
// an empty ENVIRONMENT header would make decide() fail open per FR-1.8 and
// every live deployment, not just this pod's, would revert the expression.
// A nil envContext is a caller bug; tests exercise this directly since
// NewRevertTask's own tests can't observe the resulting context.
func processExpired(l logrus.FieldLogger, ctx context.Context, expired []Model, revert func(l logrus.FieldLogger, ctx context.Context, exp Model) error, envContext func(context.Context) context.Context) {
	for _, exp := range expired {
		tctx := envContext(tenant.WithContext(ctx, exp.Tenant()))
		_ = revert(l, tctx, exp)
	}
}

func (e *RevertTask) SleepTime() time.Duration {
	return e.interval
}
