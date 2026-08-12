package buff

import (
	buff2 "atlas-channel/kafka/message/buff"
	"testing"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
)

func TestEnergyChargeChange(t *testing.T) {
	t.Run("finds the energy change", func(t *testing.T) {
		c, ok := energyChargeChange([]buff2.StatChange{
			{Type: "COMBO", Amount: 3},
			{Type: string(charconst.TemporaryStatTypeEnergyCharge), Amount: 4998},
		})
		if !ok || c.Amount != 4998 {
			t.Fatalf("got (%+v,%v)", c, ok)
		}
	})
	t.Run("absent", func(t *testing.T) {
		if _, ok := energyChargeChange([]buff2.StatChange{{Type: "COMBO", Amount: 3}}); ok {
			t.Fatal("must not match an unrelated stat")
		}
	})
	t.Run("empty", func(t *testing.T) {
		if _, ok := energyChargeChange(nil); ok {
			t.Fatal("must not match an empty change set")
		}
	})
}

// The promotion fires on EXACTLY the value that tops the accumulation cap.
// atlas-buffs emits STAT_UPDATED only on a real change and clamps at the cap,
// so exactly one event in the bar's life carries 10000.
func TestEnergyChargeShouldPromote(t *testing.T) {
	for _, tc := range []struct {
		amount int32
		want   bool
	}{
		{0, false},
		{102, false},
		{9999, false},
		{10000, true},
		{15000, false},
	} {
		if got := energyChargeShouldPromote(tc.amount); got != tc.want {
			t.Fatalf("amount=%d: got %v want %v", tc.amount, got, tc.want)
		}
	}
}
