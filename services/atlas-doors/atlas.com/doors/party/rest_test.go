package party

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExtractPreservesMemberOrder verifies that Extract keeps members in the
// order they appear in the RestModel (join order, leader seeded at index 0).
// No re-sorting must be applied.
func TestExtractPreservesMemberOrder(t *testing.T) {
	rm := RestModel{
		Id:       1,
		LeaderId: 100,
		Members: []MemberRestModel{
			{Id: 100}, // member[0] — leader (join-order index 0)
			{Id: 200}, // member[1] — second to join
		},
	}

	m, err := Extract(rm)
	require.NoError(t, err)

	assert.Equal(t, uint32(1), m.Id())
	assert.Equal(t, character.Id(100), m.LeaderId())

	members := m.Members()
	require.Len(t, members, 2)
	assert.Equal(t, character.Id(100), members[0], "member[0] must be the first in join order (leader)")
	assert.Equal(t, character.Id(200), members[1], "member[1] must be the second in join order")
}
