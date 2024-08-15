package command

import (
	"atlas-messages/character"
	"atlas-messages/tenant"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

type Producer func(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model, worldId byte, channelId byte, character character.Model, m string) (Executor, bool)

type Executor func(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) error
