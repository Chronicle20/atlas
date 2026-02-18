package command

import (
	"atlas-messages/character"
	"context"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/sirupsen/logrus"
)

type Producer func(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model, character character.Model, m string) (Executor, bool)

type Executor func(l logrus.FieldLogger) func(ctx context.Context) error
