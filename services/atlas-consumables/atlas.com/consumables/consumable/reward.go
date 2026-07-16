package consumable

import (
	consumable3 "atlas-consumables/data/consumable"
	"crypto/rand"
	"errors"
	"math/big"
	"strings"
	"time"
)

// rollReward performs one clean prob-weighted pick over the reward table using a
// CSPRNG (design task-131 §2.4 — deliberate deviation from Cosmic's order-biased
// iterate-and-maybe-nothing algorithm). Zero-prob entries are skipped naturally.
// Errors when the summed weight is zero (defense in depth; callers validate first).
func rollReward(rewards []consumable3.RewardModel) (consumable3.RewardModel, error) {
	var total uint32
	for _, r := range rewards {
		total += r.Prob()
	}
	if total == 0 {
		return consumable3.RewardModel{}, errors.New("reward table has zero total probability")
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(total)))
	if err != nil {
		return consumable3.RewardModel{}, err
	}
	roll := uint32(n.Int64())

	var cumulative uint32
	for _, r := range rewards {
		cumulative += r.Prob()
		if roll < cumulative {
			return r, nil
		}
	}
	// Unreachable given total>0, but return the last entry defensively.
	return rewards[len(rewards)-1], nil
}

// validateRewardTable is the pre-reserve guard used by RequestItemReward. It
// rejects an item that has no reward table or whose entries sum to zero
// probability (nothing could ever be rolled).
func validateRewardTable(rewards []consumable3.RewardModel) error {
	if len(rewards) == 0 {
		return errors.New("item has no reward table")
	}
	var total uint32
	for _, r := range rewards {
		total += r.Prob()
	}
	if total == 0 {
		return errors.New("reward table has zero total probability")
	}
	return nil
}

// grantQuantity clamps a reward entry's count up to 1 (design §5.4): a count of
// zero still grants a single item.
func grantQuantity(count uint32) uint32 {
	if count == 0 {
		return 1
	}
	return count
}

// rewardExpiration converts a reward entry's period (MINUTES; design §2.3) to an
// absolute expiration timestamp. period <= 0 (default -1) means no expiration.
func rewardExpiration(period int32, now time.Time) time.Time {
	if period <= 0 {
		return time.Time{}
	}
	return now.Add(time.Duration(period) * time.Minute)
}

// substituteWorldMsg fills the reward worldMsg template's /name and /item tokens.
// Applied here, once, in one place (design §4.2 — Cosmic's replaceAll was a no-op).
func substituteWorldMsg(template, characterName, itemName string) string {
	s := strings.ReplaceAll(template, "/name", characterName)
	s = strings.ReplaceAll(s, "/item", itemName)
	return s
}
