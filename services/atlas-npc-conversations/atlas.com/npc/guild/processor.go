package guild

import (
	"context"
	"github.com/sirupsen/logrus"
)

func IsLeader(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) bool {
	return func(ctx context.Context) func(characterId uint32) bool {
		return func(characterId uint32) bool {
			//TODO
			return false
		}
	}
}

func HasGuild(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) bool {
	return func(ctx context.Context) func(characterId uint32) bool {
		return func(characterId uint32) bool {
			//TODO
			return false
		}
	}
}
