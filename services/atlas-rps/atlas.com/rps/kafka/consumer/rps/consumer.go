// Package rps wires the atlas-rps command consumer: it consumes
// COMMAND_TOPIC_RPS and routes each Command's Type to the matching
// game.Processor …AndEmit method.
package rps

import (
	"atlas-rps/configuration"
	"atlas-rps/game"
	consumer2 "atlas-rps/kafka/consumer"
	rpsMsg "atlas-rps/kafka/message/rps"
	rpsSaga "atlas-rps/saga"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

// InitConsumers registers the COMMAND_TOPIC_RPS consumer.
func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("rps_command")(rpsMsg.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

// InitHandlers registers one handler per Command Type on COMMAND_TOPIC_RPS.
func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(rpsMsg.EnvCommandTopic)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleBeginCommand))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleSelectCommand))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleContinueCommand))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRetryCommand))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCollectCommand))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleQuitCommand))); err != nil {
			return err
		}
		return nil
	}
}

// LadderProviderFor builds a game.LadderProvider backed by the configuration
// service for the tenant carried on ctx. This is the real, non-shell wiring:
// game cannot import configuration directly (configuration imports game for
// the Ladder/Rung types), so the provider closure is constructed here, at a
// layer that is permitted to import both. Exported so other composition
// roots that also need a real (non-shell) game.Processor - e.g. the REST
// bootstrap in main.go - can reuse this wiring instead of duplicating it.
func LadderProviderFor(l logrus.FieldLogger, ctx context.Context) game.LadderProvider {
	return func() (game.Ladder, error) {
		t := tenant.MustFromContext(ctx)
		return configuration.NewProcessor(l, ctx).GetLadder(t.Id())
	}
}

// SagaSubmitterFor builds a game.SagaSubmitter backed by the local
// "atlas-rps/saga" package for the tenant/context of a single command or
// request. Mirrors LadderProviderFor: game cannot import the local saga
// package itself (see game.SagaSubmitter's doc), so this composition-root
// layer - which is free to import both "atlas-rps/game" and
// "atlas-rps/saga" - builds the closure that does. Exported so other
// composition roots (e.g. the REST bootstrap in main.go) can reuse this
// wiring instead of duplicating it.
func SagaSubmitterFor(l logrus.FieldLogger, ctx context.Context) game.SagaSubmitter {
	return func(s sharedsaga.Saga) error {
		return rpsSaga.NewProcessor(l, ctx).Create(s)
	}
}

// newProcessor builds the real RPS game processor for a single command,
// wired with the server-authoritative DefaultThrowSource, a
// configuration-backed LadderProvider, and a saga-orchestrator-backed
// SagaSubmitter. It is held as a package-level var (rather than called
// directly) so handler-level tests can swap in a stub ladder provider
// without standing up a real configuration REST server — mirroring the seam
// pattern used by mount/consumer.go's tamingMobInfoBroadcaster.
var newProcessor = func(l logrus.FieldLogger, ctx context.Context) game.Processor {
	return game.NewProcessorWithLadder(l, ctx, game.DefaultThrowSource, LadderProviderFor(l, ctx), SagaSubmitterFor(l, ctx))
}

func handleBeginCommand(l logrus.FieldLogger, ctx context.Context, c rpsMsg.Command[rpsMsg.BeginCommandBody]) {
	if c.Type != rpsMsg.CommandTypeBegin {
		return
	}
	if _, err := newProcessor(l, ctx).BeginAndEmit(c.CharacterId); err != nil {
		l.WithError(err).Errorf("Unable to process BEGIN command for character [%d].", c.CharacterId)
	}
}

func handleSelectCommand(l logrus.FieldLogger, ctx context.Context, c rpsMsg.Command[rpsMsg.SelectCommandBody]) {
	if c.Type != rpsMsg.CommandTypeSelect {
		return
	}
	if _, err := newProcessor(l, ctx).SelectAndEmit(c.CharacterId, game.Throw(c.Body.Throw)); err != nil {
		l.WithError(err).Errorf("Unable to process SELECT command for character [%d].", c.CharacterId)
	}
}

func handleContinueCommand(l logrus.FieldLogger, ctx context.Context, c rpsMsg.Command[rpsMsg.ContinueCommandBody]) {
	if c.Type != rpsMsg.CommandTypeContinue {
		return
	}
	if _, err := newProcessor(l, ctx).ContinueAndEmit(c.CharacterId); err != nil {
		l.WithError(err).Errorf("Unable to process CONTINUE command for character [%d].", c.CharacterId)
	}
}

func handleRetryCommand(l logrus.FieldLogger, ctx context.Context, c rpsMsg.Command[rpsMsg.RetryCommandBody]) {
	if c.Type != rpsMsg.CommandTypeRetry {
		return
	}
	if _, err := newProcessor(l, ctx).RetryAndEmit(c.CharacterId); err != nil {
		l.WithError(err).Errorf("Unable to process RETRY command for character [%d].", c.CharacterId)
	}
}

func handleCollectCommand(l logrus.FieldLogger, ctx context.Context, c rpsMsg.Command[rpsMsg.CollectCommandBody]) {
	if c.Type != rpsMsg.CommandTypeCollect {
		return
	}
	if _, err := newProcessor(l, ctx).CollectAndEmit(c.CharacterId); err != nil {
		l.WithError(err).Errorf("Unable to process COLLECT command for character [%d].", c.CharacterId)
	}
}

func handleQuitCommand(l logrus.FieldLogger, ctx context.Context, c rpsMsg.Command[rpsMsg.QuitCommandBody]) {
	if c.Type != rpsMsg.CommandTypeQuit {
		return
	}
	if _, err := newProcessor(l, ctx).QuitAndEmit(c.CharacterId); err != nil {
		l.WithError(err).Errorf("Unable to process QUIT command for character [%d].", c.CharacterId)
	}
}
