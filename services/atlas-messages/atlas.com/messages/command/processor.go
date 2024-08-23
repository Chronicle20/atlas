package command

import (
	"atlas-messages/character"
	"atlas-messages/tenant"
	"context"
	"github.com/sirupsen/logrus"
)

type Producer func(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model, worldId byte, channelId byte, character character.Model, m string) (Executor, bool)

type Executor func(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) error
