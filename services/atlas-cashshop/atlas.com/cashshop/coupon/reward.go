package coupon

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

type RewardType string

const (
	RewardTypeCurrency RewardType = "CURRENCY"
	RewardTypeCashItem RewardType = "CASH_ITEM"
)

var ErrInvalidReward = errors.New("invalid reward")

// Reward is a discriminated value. Only the fields belonging to its Type are
// meaningful; the others are zero and are omitted from JSON.
//
// Currency ids reuse the existing wallet.Model.Balance convention (1 = credit
// / NX, 2 = Maple Points, anything else = prepaid) rather than introducing a
// second enum for the same thing (DOM-21).
//
// Mesos and regular-inventory items are explicit non-goals (PRD §2); adding
// either means adding a RewardType here AND a granter in granter.go, and — for
// a reward owned by another service — is the point at which the local
// redemption transaction has to become a saga (design §2.1).
type Reward struct {
	rewardType   RewardType
	currency     uint32
	amount       uint32
	serialNumber uint32
	quantity     uint32
}

func NewCurrencyReward(currency uint32, amount uint32) Reward {
	return Reward{rewardType: RewardTypeCurrency, currency: currency, amount: amount}
}

func NewCashItemReward(serialNumber uint32, quantity uint32) Reward {
	return Reward{rewardType: RewardTypeCashItem, serialNumber: serialNumber, quantity: quantity}
}

func (r Reward) Type() RewardType     { return r.rewardType }
func (r Reward) Currency() uint32     { return r.currency }
func (r Reward) Amount() uint32       { return r.amount }
func (r Reward) SerialNumber() uint32 { return r.serialNumber }
func (r Reward) Quantity() uint32     { return r.quantity }

func (r Reward) Validate() error {
	switch r.rewardType {
	case RewardTypeCurrency:
		if r.amount == 0 {
			return fmt.Errorf("%w: currency reward amount must be positive", ErrInvalidReward)
		}
		return nil
	case RewardTypeCashItem:
		if r.serialNumber == 0 {
			return fmt.Errorf("%w: cash item reward needs a serial number", ErrInvalidReward)
		}
		if r.quantity == 0 {
			return fmt.Errorf("%w: cash item reward quantity must be positive", ErrInvalidReward)
		}
		return nil
	default:
		return fmt.Errorf("%w: unknown reward type %q", ErrInvalidReward, r.rewardType)
	}
}

// rewardJSON is the on-disk and on-the-wire shape. It is deliberately the same
// document in the jsonb column and in the REST attribute, so an admin editing
// a bundle sees exactly what is stored.
type rewardJSON struct {
	Type         RewardType `json:"type"`
	Currency     uint32     `json:"currency,omitempty"`
	Amount       uint32     `json:"amount,omitempty"`
	SerialNumber uint32     `json:"serialNumber,omitempty"`
	Quantity     uint32     `json:"quantity,omitempty"`
}

func (r Reward) MarshalJSON() ([]byte, error) {
	return json.Marshal(rewardJSON{
		Type:         r.rewardType,
		Currency:     r.currency,
		Amount:       r.amount,
		SerialNumber: r.serialNumber,
		Quantity:     r.quantity,
	})
}

func (r *Reward) UnmarshalJSON(b []byte) error {
	var j rewardJSON
	if err := json.Unmarshal(b, &j); err != nil {
		return err
	}
	r.rewardType = j.Type
	r.currency = j.Currency
	r.amount = j.Amount
	r.serialNumber = j.SerialNumber
	r.quantity = j.Quantity
	return nil
}

// Rewards is the whole bundle, persisted as one jsonb document.
type Rewards []Reward

func (rs Rewards) Value() (driver.Value, error) {
	if rs == nil {
		return []byte("[]"), nil
	}
	return json.Marshal([]Reward(rs))
}

func (rs *Rewards) Scan(src interface{}) error {
	if src == nil {
		*rs = nil
		return nil
	}
	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("coupon: cannot scan %T into Rewards", src)
	}
	if len(b) == 0 {
		*rs = nil
		return nil
	}
	return json.Unmarshal(b, (*[]Reward)(rs))
}

func (rs Rewards) Validate() error {
	if len(rs) == 0 {
		return fmt.Errorf("%w: a coupon must grant at least one reward", ErrInvalidReward)
	}
	for i, r := range rs {
		if err := r.Validate(); err != nil {
			return fmt.Errorf("reward %d: %w", i, err)
		}
	}
	return nil
}

// CashItemCount is the number of locker slots this bundle needs.
func (rs Rewards) CashItemCount() int {
	n := 0
	for _, r := range rs {
		if r.Type() == RewardTypeCashItem {
			n++
		}
	}
	return n
}
