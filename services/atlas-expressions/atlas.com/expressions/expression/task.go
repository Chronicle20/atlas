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

func (e *RevertTask) Run() {
	sctx, span := otel.GetTracerProvider().Tracer("atlas-expressions").Start(context.Background(), RevertTaskName)
	defer span.End()

	for _, exp := range GetRegistry().popExpired(sctx) {
		tctx := e.envContext(tenant.WithContext(sctx, exp.Tenant()))
		transactionId := uuid.New() // Generate a new transaction ID for each expired expression
		_ = producer.ProviderImpl(e.l)(tctx)(expression.EnvExpressionEvent)(expressionEventProvider(transactionId, exp.CharacterId(), exp.Field(), 0))
	}
}

func (e *RevertTask) SleepTime() time.Duration {
	return e.interval
}
