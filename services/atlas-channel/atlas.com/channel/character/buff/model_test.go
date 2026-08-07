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
