package saga

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractCharacterId_WarpToPortalPayload(t *testing.T) {
	step := NewStep[any]("s1", Pending, WarpToPortal, WarpToPortalPayload{CharacterId: 4242})
	assert.Equal(t, uint32(4242), ExtractCharacterId(step))
}

// FR-1.4: without this case the character-id guard cannot constrain a
// WarpToSavedLocation step — it would read 0 and treat the step as
// unconstrained, silently defeating the guard for that action.
func TestExtractCharacterId_WarpToSavedLocationPayload(t *testing.T) {
	step := NewStep[any]("s1", Pending, WarpToSavedLocation, WarpToSavedLocationPayload{CharacterId: 777, LocationType: "FREE_MARKET"})
	assert.Equal(t, uint32(777), ExtractCharacterId(step))
}

func TestExtractCharacterId_OpenNpcShop(t *testing.T) {
	step := NewStep[any]("s1", Pending, OpenNpcShop, OpenNpcShopPayload{CharacterId: 4242, NpcTemplateId: 9090000})
	assert.Equal(t, uint32(4242), ExtractCharacterId(step))
}

func TestExtractCharacterId_UnknownPayloadIsZero(t *testing.T) {
	step := NewStep[any]("s1", Pending, WarpToPortal, struct{ Foo string }{Foo: "bar"})
	assert.Equal(t, uint32(0), ExtractCharacterId(step),
		"unknown payloads must read 0 so the guard leaves them unconstrained")
}
