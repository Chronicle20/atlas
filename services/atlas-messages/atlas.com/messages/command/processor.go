package command

import (
	"atlas-messages/character"
	"context"
	"github.com/sirupsen/logrus"
)

type Producer func(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, character character.Model, m string) (Executor, bool)

type Executor func(l logrus.FieldLogger) func(ctx context.Context) error
