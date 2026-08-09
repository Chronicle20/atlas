package coupon

import (
	"encoding/json"
	"testing"
)

func TestRewardRoundTripsThroughJSON(t *testing.T) {
	in := Rewards{
		NewCurrencyReward(2, 10000),
		NewCashItemReward(50200000, 1),
	}
	v, err := in.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	var out Rewards
	if err := out.Scan(v); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("len = %d, want 2", len(out))
	}
	if out[0].Type() != RewardTypeCurrency || out[0].Currency() != 2 || out[0].Amount() != 10000 {
		t.Errorf("currency reward = %+v", out[0])
	}
	if out[1].Type() != RewardTypeCashItem || out[1].SerialNumber() != 50200000 || out[1].Quantity() != 1 {
		t.Errorf("cash item reward = %+v", out[1])
	}
}

func TestRewardsScanNil(t *testing.T) {
	var out Rewards
	if err := out.Scan(nil); err != nil {
		t.Fatalf("Scan(nil): %v", err)
	}
	if len(out) != 0 {
		t.Errorf("len = %d, want 0", len(out))
	}
}

func TestRewardJSONShapeMatchesTheRESTContract(t *testing.T) {
	b, err := json.Marshal(NewCurrencyReward(1, 5))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"type":"CURRENCY","currency":1,"amount":5}` {
		t.Errorf("currency JSON = %s", b)
	}
	b, err = json.Marshal(NewCashItemReward(50200000, 2))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"type":"CASH_ITEM","serialNumber":50200000,"quantity":2}` {
		t.Errorf("cash item JSON = %s", b)
	}
}

func TestRewardValidate(t *testing.T) {
	for _, c := range []struct {
		name    string
		reward  Reward
		wantErr bool
	}{
		{"valid currency", NewCurrencyReward(1, 100), false},
		{"valid cash item", NewCashItemReward(50200000, 1), false},
		{"zero currency amount", NewCurrencyReward(1, 0), true},
		{"zero serial number", NewCashItemReward(0, 1), true},
		{"zero quantity", NewCashItemReward(50200000, 0), true},
		{"unknown type", Reward{rewardType: "MESO", amount: 1}, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			if err := c.reward.Validate(); (err != nil) != c.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, c.wantErr)
			}
		})
	}
}
