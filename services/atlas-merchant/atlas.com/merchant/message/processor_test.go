package message

import (
	"context"
	"testing"

	database "github.com/Chronicle20/atlas-database"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dbName := uuid.New().String()
	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	l, _ := test.NewNullLogger()
	database.RegisterTenantCallbacks(l, db)

	require.NoError(t, Migration(db))
	return db
}

func setupTestContext(t *testing.T) (context.Context, tenant.Model) {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tenant.WithContext(context.Background(), ten), ten
}

func TestSendAndGetMessages(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	shopId := uuid.New()

	require.NoError(t, p.SendMessage(shopId, 1000, "Hello"))
	require.NoError(t, p.SendMessage(shopId, 2000, "World"))

	messages, err := p.GetMessages(shopId)
	require.NoError(t, err)
	assert.Len(t, messages, 2)
	assert.Equal(t, "Hello", messages[0].Content())
	assert.Equal(t, uint32(1000), messages[0].CharacterId())
	assert.Equal(t, "World", messages[1].Content())
	assert.Equal(t, uint32(2000), messages[1].CharacterId())
}

func TestGetMessages_EmptyShop(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	messages, err := p.GetMessages(uuid.New())
	require.NoError(t, err)
	assert.Empty(t, messages)
}

func TestMultipleShopIsolation(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	shop1 := uuid.New()
	shop2 := uuid.New()

	require.NoError(t, p.SendMessage(shop1, 1000, "Shop 1 message"))
	require.NoError(t, p.SendMessage(shop2, 2000, "Shop 2 message"))

	msgs1, err := p.GetMessages(shop1)
	require.NoError(t, err)
	assert.Len(t, msgs1, 1)
	assert.Equal(t, "Shop 1 message", msgs1[0].Content())

	msgs2, err := p.GetMessages(shop2)
	require.NoError(t, err)
	assert.Len(t, msgs2, 1)
	assert.Equal(t, "Shop 2 message", msgs2[0].Content())
}

func TestSendMessage_FieldPersistence(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	shopId := uuid.New()
	require.NoError(t, p.SendMessage(shopId, 1000, "Test content"))

	messages, err := p.GetMessages(shopId)
	require.NoError(t, err)
	require.Len(t, messages, 1)

	msg := messages[0]
	assert.NotEqual(t, uuid.Nil, msg.Id())
	assert.Equal(t, shopId, msg.ShopId())
	assert.Equal(t, uint32(1000), msg.CharacterId())
	assert.Equal(t, "Test content", msg.Content())
	assert.False(t, msg.SentAt().IsZero())
}
