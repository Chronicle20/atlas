package buffs

import (
	"context"

	"github.com/sirupsen/logrus"
)

// GetActiveBuffs retrieves all active buffs for a character from atlas-buffs
func GetActiveBuffs(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) ([]RestModel, error) {
	return func(ctx context.Context) func(characterId uint32) ([]RestModel, error) {
		return func(characterId uint32) ([]RestModel, error) {
			return requestBuffs(characterId)(l, ctx)
		}
	}
}
