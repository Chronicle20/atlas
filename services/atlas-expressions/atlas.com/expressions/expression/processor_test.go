package expression

import (
	"atlas-expressions/kafka/message"
	"context"
	"testing"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

func setupProcessorTest(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(client)
}

func setupTestContext(t *testing.T, ten tenant.Model) context.Context {
	t.Helper()
	return tenant.WithContext(context.Background(), ten)
}

func setupTestLogger(t *testing.T) logrus.FieldLogger {
	t.Helper()
	logger, _ := test.NewNullLogger()
	return logger
}

func TestNewProcessor(t *testing.T) {
	setupProcessorTest(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)

	p := NewProcessor(l, ctx)

	assert.NotNil(t, p)
}

func TestNewProcessor_ExtractsTenant(t *testing.T) {
	setupProcessorTest(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)

	p := NewProcessor(l, ctx)
	impl := p.(*ProcessorImpl)

	assert.Equal(t, ten, impl.t)
}

func TestNewProcessor_PanicsOnMissingTenant(t *testing.T) {
	setupProcessorTest(t)
	ctx := context.Background()
	l := setupTestLogger(t)

	assert.Panics(t, func() {
		NewProcessor(l, ctx)
	})
}

func TestProcessor_Change(t *testing.T) {
	setupProcessorTest(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)

	p := NewProcessor(l, ctx)
	mb := message.NewBuffer()

	transactionId := uuid.New()
	characterId := uint32(1000)
	worldId := world.Id(0)
	channelId := channel.Id(1)
	mapId := _map.Id(100000000)
	expr := uint32(5)

	f := field.NewBuilder(worldId, channelId, mapId).Build()
	model, err := p.Change(mb, transactionId, characterId, f, expr)

	assert.NoError(t, err)
	assert.Equal(t, characterId, model.CharacterId())
	assert.Equal(t, expr, model.Expression())
	assert.Equal(t, worldId, model.WorldId())
	assert.Equal(t, channelId, model.ChannelId())
	assert.Equal(t, mapId, model.MapId())
}

func TestProcessor_Change_AddsToRegistry(t *testing.T) {
	setupProcessorTest(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)

	p := NewProcessor(l, ctx)
	mb := message.NewBuffer()

	characterId := uint32(1000)

	f := field.NewBuilder(0, 1, 100000000).Build()
	_, _ = p.Change(mb, uuid.New(), characterId, f, 5)

	retrieved, found := GetRegistry().get(ctx, characterId)
	assert.True(t, found)
	assert.Equal(t, uint32(5), retrieved.Expression())
}

func TestProcessor_Change_AddsMessageToBuffer(t *testing.T) {
	setupProcessorTest(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)

	p := NewProcessor(l, ctx)
	mb := message.NewBuffer()

	f := field.NewBuilder(0, 1, 100000000).Build()
	_, err := p.Change(mb, uuid.New(), 1000, f, 5)

	assert.NoError(t, err)

	messages := mb.GetAll()
	assert.NotEmpty(t, messages)
}

func TestProcessor_Clear(t *testing.T) {
	setupProcessorTest(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)

	characterId := uint32(1000)
	f := field.NewBuilder(0, 1, 100000000).Build()
	GetRegistry().add(ctx, characterId, f, 5)

	_, found := GetRegistry().get(ctx, characterId)
	assert.True(t, found)

	p := NewProcessor(l, ctx)
	mb := message.NewBuffer()

	model, err := p.Clear(mb, uuid.New(), characterId)

	assert.NoError(t, err)
	assert.Equal(t, Model{}, model)

	_, found = GetRegistry().get(ctx, characterId)
	assert.False(t, found)
}

func TestProcessor_Clear_NonExistent(t *testing.T) {
	setupProcessorTest(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)

	p := NewProcessor(l, ctx)
	mb := message.NewBuffer()

	model, err := p.Clear(mb, uuid.New(), 9999)

	assert.NoError(t, err)
	assert.Equal(t, Model{}, model)
}

func TestProcessor_MultipleChanges(t *testing.T) {
	setupProcessorTest(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)

	p := NewProcessor(l, ctx)

	for i := uint32(0); i < 10; i++ {
		mb := message.NewBuffer()
		f := field.NewBuilder(0, 1, 100000000).Build()
		_, err := p.Change(mb, uuid.New(), 1000+i, f, i)
		assert.NoError(t, err)
	}

	for i := uint32(0); i < 10; i++ {
		retrieved, found := GetRegistry().get(ctx, 1000+i)
		assert.True(t, found)
		assert.Equal(t, i, retrieved.Expression())
	}
}

func TestProcessor_ChangeReplacesPrevious(t *testing.T) {
	setupProcessorTest(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)

	p := NewProcessor(l, ctx)
	characterId := uint32(1000)
	f := field.NewBuilder(0, 1, 100000000).Build()

	mb1 := message.NewBuffer()
	_, _ = p.Change(mb1, uuid.New(), characterId, f, 5)

	mb2 := message.NewBuffer()
	_, _ = p.Change(mb2, uuid.New(), characterId, f, 10)

	retrieved, found := GetRegistry().get(ctx, characterId)
	assert.True(t, found)
	assert.Equal(t, uint32(10), retrieved.Expression())
}
