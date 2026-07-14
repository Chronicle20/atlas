package frederick

import (
	"testing"

	"atlas-merchant/kafka/message/asset"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestItemBuilder(t *testing.T) {
	id := uuid.New()
	m, err := NewItemBuilder().
		SetId(id).
		SetCharacterId(1000).
		SetItemId(2000000).
		SetItemType(2).
		SetQuantity(50).
		SetItemSnapshot(asset.AssetData{Quantity: 50}).
		Build()
	require.NoError(t, err)
	assert.Equal(t, id, m.Id())
	assert.Equal(t, uint32(1000), m.CharacterId())
	assert.Equal(t, uint32(2000000), m.ItemId())
	assert.Equal(t, byte(2), m.ItemType())
	assert.Equal(t, uint16(50), m.Quantity())
	assert.Equal(t, uint32(50), m.ItemSnapshot().Quantity)
}

func TestItemBuilder_Validation(t *testing.T) {
	_, err := NewItemBuilder().SetCharacterId(1).SetItemId(2000000).Build()
	assert.Error(t, err, "missing id")
	_, err = NewItemBuilder().SetId(uuid.New()).SetItemId(2000000).Build()
	assert.Error(t, err, "missing characterId")
	_, err = NewItemBuilder().SetId(uuid.New()).SetCharacterId(1).Build()
	assert.Error(t, err, "missing itemId")
}

func TestMesoBuilder(t *testing.T) {
	id := uuid.New()
	m, err := NewMesoBuilder().SetId(id).SetCharacterId(1000).SetAmount(123456).Build()
	require.NoError(t, err)
	assert.Equal(t, id, m.Id())
	assert.Equal(t, uint32(1000), m.CharacterId())
	assert.Equal(t, uint32(123456), m.Amount())

	_, err = NewMesoBuilder().SetCharacterId(1).Build()
	assert.Error(t, err, "missing id")
}
