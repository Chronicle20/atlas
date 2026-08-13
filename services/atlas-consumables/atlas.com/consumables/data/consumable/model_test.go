package consumable

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMorphsGetter(t *testing.T) {
	rm := RestModel{Morphs: map[uint32]uint32{100: 30, 101: 70}}
	m, err := Extract(rm)
	assert.NoError(t, err)
	assert.Equal(t, map[uint32]uint32{100: 30, 101: 70}, m.Morphs())
}

func TestMorphsGetter_Empty(t *testing.T) {
	m, err := Extract(RestModel{})
	assert.NoError(t, err)
	assert.Empty(t, m.Morphs())
}

// TestCatchFieldAccessors covers the bridle fields the catch flow reads. They
// were already parsed and carried over REST; only the getters were missing.
func TestCatchFieldAccessors(t *testing.T) {
	rm := RestModel{
		Id:            2270008,
		Create:        2022323,
		MonsterId:     9500336,
		MonsterHP:     0,
		BridleMsgType: 4,
		BridleProp:    50,
		BridlePropChg: 1.2,
		UseDelay:      3000,
		DelayMsg:      "You cannot use the Fishing Net yet.",
	}
	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if m.Create() != 2022323 || m.MonsterId() != 9500336 || m.MonsterHp() != 0 ||
		m.BridleMsgType() != 4 || m.BridleProp() != 50 || m.BridlePropChg() != 1.2 ||
		m.UseDelay() != 3000 || m.DelayMsg() != "You cannot use the Fishing Net yet." {
		t.Fatalf("accessors returned %+v", m)
	}
}
