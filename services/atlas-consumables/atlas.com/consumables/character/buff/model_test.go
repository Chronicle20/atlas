package buff

import (
	"atlas-consumables/character/buff/stat"
	"testing"
	"time"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
)

func TestIsZombified(t *testing.T) {
	tests := []struct {
		name  string
		buffs []Model
		want  bool
	}{
		{
			name: "unexpired undead",
			buffs: []Model{
				NewBuff(1, 1, 0, []stat.Model{{Type: charconst.TemporaryStatTypeUndead, Amount: 1}}, time.Now(), time.Now().Add(time.Minute), false),
			},
			want: true,
		},
		{
			name: "expired undead",
			buffs: []Model{
				NewBuff(1, 1, 0, []stat.Model{{Type: charconst.TemporaryStatTypeUndead, Amount: 1}}, time.Now(), time.Now().Add(-time.Second), false),
			},
			want: false,
		},
		{
			name: "no-expiry undead",
			buffs: []Model{
				NewBuff(1, 1, 0, []stat.Model{{Type: charconst.TemporaryStatTypeUndead, Amount: 1}}, time.Now(), time.Time{}, true),
			},
			want: true,
		},
		{
			name: "unexpired non-undead",
			buffs: []Model{
				NewBuff(1, 1, 0, []stat.Model{{Type: charconst.TemporaryStatTypeSpeed, Amount: 20}}, time.Now(), time.Now().Add(time.Minute), false),
			},
			want: false,
		},
		{
			name: "undead not first change",
			buffs: []Model{
				NewBuff(1, 1, 0, []stat.Model{
					{Type: charconst.TemporaryStatTypeSpeed, Amount: 20},
					{Type: charconst.TemporaryStatTypeUndead, Amount: 1},
				}, time.Now(), time.Now().Add(time.Minute), false),
			},
			want: true,
		},
		{
			name:  "empty slice",
			buffs: nil,
			want:  false,
		},
		{
			name: "expired undead alongside unexpired speed",
			buffs: []Model{
				NewBuff(1, 1, 0, []stat.Model{{Type: charconst.TemporaryStatTypeUndead, Amount: 1}}, time.Now(), time.Now().Add(-time.Second), false),
				NewBuff(2, 1, 0, []stat.Model{{Type: charconst.TemporaryStatTypeSpeed, Amount: 20}}, time.Now(), time.Now().Add(time.Minute), false),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsZombified(tt.buffs); got != tt.want {
				t.Fatalf("IsZombified() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExpiredHonoursNoExpiry(t *testing.T) {
	noExpiry := NewBuff(1, 1, 0, nil, time.Now(), time.Time{}, true)
	if noExpiry.Expired() {
		t.Fatal("no-expiry buff must not report expired")
	}

	pastExpiry := NewBuff(1, 1, 0, nil, time.Now(), time.Now().Add(-time.Second), false)
	if !pastExpiry.Expired() {
		t.Fatal("past-expiry buff must report expired")
	}
}

func TestIsPotionLocked(t *testing.T) {
	tests := []struct {
		name  string
		buffs []Model
		want  bool
	}{
		{
			name: "unexpired stop portion",
			buffs: []Model{
				NewBuff(1, 1, 0, []stat.Model{{Type: charconst.TemporaryStatTypeStopPortion, Amount: 1}}, time.Now(), time.Now().Add(time.Minute), false),
			},
			want: true,
		},
		{
			name: "expired stop portion",
			buffs: []Model{
				NewBuff(1, 1, 0, []stat.Model{{Type: charconst.TemporaryStatTypeStopPortion, Amount: 1}}, time.Now(), time.Now().Add(-time.Second), false),
			},
			want: false,
		},
		{
			name: "no-expiry stop portion",
			buffs: []Model{
				NewBuff(1, 1, 0, []stat.Model{{Type: charconst.TemporaryStatTypeStopPortion, Amount: 1}}, time.Now(), time.Time{}, true),
			},
			want: true,
		},
		{
			name: "unexpired non-stop-portion",
			buffs: []Model{
				NewBuff(1, 1, 0, []stat.Model{{Type: charconst.TemporaryStatTypeSpeed, Amount: 20}}, time.Now(), time.Now().Add(time.Minute), false),
			},
			want: false,
		},
		{
			name: "stop portion not first change",
			buffs: []Model{
				NewBuff(1, 1, 0, []stat.Model{
					{Type: charconst.TemporaryStatTypeSpeed, Amount: 20},
					{Type: charconst.TemporaryStatTypeStopPortion, Amount: 1},
				}, time.Now(), time.Now().Add(time.Minute), false),
			},
			want: true,
		},
		{
			name:  "empty slice",
			buffs: nil,
			want:  false,
		},
		{
			name: "expired stop portion alongside unexpired speed",
			buffs: []Model{
				NewBuff(1, 1, 0, []stat.Model{{Type: charconst.TemporaryStatTypeStopPortion, Amount: 1}}, time.Now(), time.Now().Add(-time.Second), false),
				NewBuff(2, 1, 0, []stat.Model{{Type: charconst.TemporaryStatTypeSpeed, Amount: 20}}, time.Now(), time.Now().Add(time.Minute), false),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPotionLocked(tt.buffs); got != tt.want {
				t.Fatalf("IsPotionLocked() = %v, want %v", got, tt.want)
			}
		})
	}
}
