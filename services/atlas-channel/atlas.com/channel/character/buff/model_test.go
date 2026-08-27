package buff

import (
	"atlas-channel/character/buff/stat"
	"testing"
	"time"
)

func TestNoExpiryMirrorNeverExpires(t *testing.T) {
	b := NewBuff(5211006, 1, 0, []stat.Model{stat.NewStat("HOMING_BEACON", 1000001)}, time.Now(), time.Time{}, true)
	if b.Expired() {
		t.Fatal("no-expiry mirror buff must not report expired")
	}
	if !b.NoExpiry() {
		t.Fatal("NoExpiry() must be true")
	}
}

func TestFiniteMirrorStillExpires(t *testing.T) {
	b := NewBuff(2001001, 5, 1000, []stat.Model{stat.NewStat("SPEED", 20)}, time.Now().Add(-2*time.Second), time.Now().Add(-time.Second), false)
	if !b.Expired() {
		t.Fatal("past-expiry finite mirror buff must report expired")
	}
}

func TestIsZombified(t *testing.T) {
	tests := []struct {
		name   string
		buffs  []Model
		expect bool
	}{
		{
			name: "unexpired undead",
			buffs: []Model{
				NewBuff(9001001, 1, 0, []stat.Model{stat.NewStat("UNDEAD", 1)}, time.Now(), time.Now().Add(time.Minute), false),
			},
			expect: true,
		},
		{
			name: "expired undead",
			buffs: []Model{
				NewBuff(9001001, 1, 0, []stat.Model{stat.NewStat("UNDEAD", 1)}, time.Now(), time.Now().Add(-time.Second), false),
			},
			expect: false,
		},
		{
			name: "no-expiry undead",
			buffs: []Model{
				NewBuff(9001001, 1, 0, []stat.Model{stat.NewStat("UNDEAD", 1)}, time.Now(), time.Time{}, true),
			},
			expect: true,
		},
		{
			name: "unexpired non-undead",
			buffs: []Model{
				NewBuff(2001001, 5, 1000, []stat.Model{stat.NewStat("SPEED", 20)}, time.Now(), time.Now().Add(time.Minute), false),
			},
			expect: false,
		},
		{
			name: "undead not first change",
			buffs: []Model{
				NewBuff(9001001, 1, 0, []stat.Model{stat.NewStat("SPEED", 20), stat.NewStat("UNDEAD", 1)}, time.Now(), time.Now().Add(time.Minute), false),
			},
			expect: true,
		},
		{
			name:   "empty slice",
			buffs:  nil,
			expect: false,
		},
		{
			name: "expired undead alongside unexpired speed",
			buffs: []Model{
				NewBuff(9001001, 1, 0, []stat.Model{stat.NewStat("UNDEAD", 1)}, time.Now(), time.Now().Add(-time.Second), false),
				NewBuff(2001001, 5, 1000, []stat.Model{stat.NewStat("SPEED", 20)}, time.Now(), time.Now().Add(time.Minute), false),
			},
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsZombified(tt.buffs); got != tt.expect {
				t.Fatalf("IsZombified() = %v, want %v", got, tt.expect)
			}
		})
	}
}
