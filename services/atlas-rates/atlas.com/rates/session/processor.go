package session

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

// GetSessionsSince retrieves all sessions for a character since the given time
func GetSessionsSince(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, since time.Time) ([]SessionRestModel, error) {
	return func(ctx context.Context) func(characterId uint32, since time.Time) ([]SessionRestModel, error) {
		return func(characterId uint32, since time.Time) ([]SessionRestModel, error) {
			return RequestSessionsSince(characterId, since.Unix())(l, ctx)
		}
	}
}

// ComputePlaytimeSince computes total playtime for a character since the given time

// ComputePlaytimeInRange computes total playtime within a specific time range
func ComputePlaytimeInRange(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, start, end time.Time) (time.Duration, error) {
	return func(ctx context.Context) func(characterId uint32, start, end time.Time) (time.Duration, error) {
		return func(characterId uint32, start, end time.Time) (time.Duration, error) {
			sessions, err := GetSessionsSince(l)(ctx)(characterId, start)
			if err != nil {
				return 0, err
			}

			var total time.Duration
			for _, session := range sessions {
				total += session.OverlapsWith(start, end)
			}

			return total, nil
		}
	}
}
