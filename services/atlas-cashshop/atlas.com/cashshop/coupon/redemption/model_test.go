package redemption

import (
	"atlas-cashshop/coupon"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsUniqueViolation(t *testing.T) {
	// 23505 is Postgres' unique_violation. The redemption insert relies on it
	// to resolve the same-account race into exactly one COUPON_ALREADY_USED,
	// so misclassifying it would turn a lost race into UNKNOWN_ERROR.
	for _, c := range []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"unrelated", errors.New("boom"), false},
		{"unique violation", &pgconn.PgError{Code: "23505"}, true},
		{"wrapped unique violation", errors.Join(errors.New("insert failed"), &pgconn.PgError{Code: "23505"}), true},
		{"foreign key violation", &pgconn.PgError{Code: "23503"}, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := IsUniqueViolation(c.err); got != c.want {
				t.Errorf("IsUniqueViolation(%v) = %v, want %v", c.err, got, c.want)
			}
		})
	}
}

func validRewards() coupon.Rewards {
	return coupon.Rewards{coupon.NewCurrencyReward(1, 100)}
}

func TestBuilderRejectsAnInvalidRedemption(t *testing.T) {
	if _, err := NewBuilder(uuid.Nil, 123, 456).SetRewardsGranted(validRewards()).Build(); err == nil {
		t.Error("want an error for a zero couponId")
	}
	if _, err := NewBuilder(uuid.New(), 0, 456).SetRewardsGranted(validRewards()).Build(); err == nil {
		t.Error("want an error for a zero accountId")
	}
	if _, err := NewBuilder(uuid.New(), 123, 456).Build(); err == nil {
		t.Error("want an error for an empty rewardsGranted snapshot")
	}
}

func TestBuilderBuildsAValidRedemption(t *testing.T) {
	id := uuid.New()
	couponId := uuid.New()
	transactionId := uuid.New()
	rewards := validRewards()
	redeemedAt := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)

	m, err := NewBuilder(couponId, 123, 456).
		SetId(id).
		SetTransactionId(transactionId).
		SetRewardsGranted(rewards).
		SetRedeemedAt(redeemedAt).
		Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if m.Id() != id {
		t.Errorf("Id() = %v, want %v", m.Id(), id)
	}
	if m.CouponId() != couponId {
		t.Errorf("CouponId() = %v, want %v", m.CouponId(), couponId)
	}
	if m.AccountId() != 123 {
		t.Errorf("AccountId() = %v, want 123", m.AccountId())
	}
	if m.CharacterId() != 456 {
		t.Errorf("CharacterId() = %v, want 456", m.CharacterId())
	}
	if m.TransactionId() != transactionId {
		t.Errorf("TransactionId() = %v, want %v", m.TransactionId(), transactionId)
	}
	if len(m.RewardsGranted()) != len(rewards) {
		t.Errorf("RewardsGranted() = %v, want %v", m.RewardsGranted(), rewards)
	}
	if !m.RedeemedAt().Equal(redeemedAt) {
		t.Errorf("RedeemedAt() = %v, want %v", m.RedeemedAt(), redeemedAt)
	}
}
