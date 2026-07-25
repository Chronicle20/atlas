package command

import (
	"atlas-messages/character"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

type Producer func(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model, character character.Model, m string) (Executor, bool)

type Executor func(l logrus.FieldLogger) func(ctx context.Context) error
