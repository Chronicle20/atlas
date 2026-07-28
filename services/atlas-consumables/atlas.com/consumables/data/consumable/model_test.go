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
