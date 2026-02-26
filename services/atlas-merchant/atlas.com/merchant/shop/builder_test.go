package shop

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShopBuilder_ValidCharacterShop(t *testing.T) {
	id := uuid.New()
	m, err := NewBuilder().
		SetId(id).
		SetCharacterId(12345).
		SetShopType(CharacterShop).
		SetState(Draft).
		SetTitle("My Shop").
		SetMapId(910000000).
		SetX(100).
		SetY(200).
		SetPermitItemId(5140000).
		SetCreatedAt(time.Now()).
		Build()

	require.NoError(t, err)
	assert.Equal(t, id, m.Id())
	assert.Equal(t, uint32(12345), m.CharacterId())
	assert.Equal(t, CharacterShop, m.ShopType())
	assert.Equal(t, Draft, m.State())
	assert.Equal(t, "My Shop", m.Title())
	assert.Equal(t, uint32(910000000), m.MapId())
	assert.Nil(t, m.ExpiresAt())
}

func TestShopBuilder_ValidHiredMerchant(t *testing.T) {
	id := uuid.New()
	expires := time.Now().Add(24 * time.Hour)
	m, err := NewBuilder().
		SetId(id).
		SetCharacterId(12345).
		SetShopType(HiredMerchant).
		SetState(Draft).
		SetTitle("Hired Merchant").
		SetMapId(910000000).
		SetX(50).
		SetY(50).
		SetPermitItemId(5030000).
		SetCreatedAt(time.Now()).
		SetExpiresAt(&expires).
		Build()

	require.NoError(t, err)
	assert.Equal(t, HiredMerchant, m.ShopType())
	assert.NotNil(t, m.ExpiresAt())
}

func TestShopBuilder_MissingId(t *testing.T) {
	_, err := NewBuilder().
		SetCharacterId(12345).
		SetShopType(CharacterShop).
		SetState(Draft).
		Build()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "id is required")
}

func TestShopBuilder_MissingCharacterId(t *testing.T) {
	_, err := NewBuilder().
		SetId(uuid.New()).
		SetShopType(CharacterShop).
		SetState(Draft).
		Build()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "characterId is required")
}

func TestShopBuilder_MissingShopType(t *testing.T) {
	_, err := NewBuilder().
		SetId(uuid.New()).
		SetCharacterId(12345).
		SetState(Draft).
		Build()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "shopType is required")
}

func TestShopBuilder_MissingState(t *testing.T) {
	_, err := NewBuilder().
		SetId(uuid.New()).
		SetCharacterId(12345).
		SetShopType(CharacterShop).
		Build()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "state is required")
}

func TestShopBuilder_Clone(t *testing.T) {
	original, err := NewBuilder().
		SetId(uuid.New()).
		SetCharacterId(12345).
		SetShopType(CharacterShop).
		SetState(Draft).
		SetTitle("My Shop").
		Build()
	require.NoError(t, err)

	cloned, err := Clone(original).
		SetState(Open).
		Build()
	require.NoError(t, err)

	assert.Equal(t, Draft, original.State())
	assert.Equal(t, Open, cloned.State())
	assert.Equal(t, original.Id(), cloned.Id())
}

func TestShopBuilder_StateEnums(t *testing.T) {
	assert.Equal(t, State(1), Draft)
	assert.Equal(t, State(2), Open)
	assert.Equal(t, State(3), Maintenance)
	assert.Equal(t, State(4), Closed)
}

func TestShopBuilder_ShopTypeEnums(t *testing.T) {
	assert.Equal(t, ShopType(1), CharacterShop)
	assert.Equal(t, ShopType(2), HiredMerchant)
}

func TestShopBuilder_CloseReasonEnums(t *testing.T) {
	assert.Equal(t, CloseReason(0), CloseReasonNone)
	assert.Equal(t, CloseReason(1), CloseReasonSoldOut)
	assert.Equal(t, CloseReason(2), CloseReasonManualClose)
	assert.Equal(t, CloseReason(3), CloseReasonDisconnect)
	assert.Equal(t, CloseReason(4), CloseReasonExpired)
	assert.Equal(t, CloseReason(5), CloseReasonServerRestart)
	assert.Equal(t, CloseReason(6), CloseReasonEmpty)
}
