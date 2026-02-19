package character

import (
	"testing"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func setupProcessorTest(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(client)
}

func testLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	return l
}

func TestProcessor_Enter(t *testing.T) {
	setupProcessorTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	l := testLogger()
	p := NewProcessor(l, ctx)

	characterId := uint32(20001)
	f := field.NewBuilder(1, 2, 100000000).Build()

	p.Enter(f, characterId)

	result, ok := GetRegistry().GetMap(ctx, characterId)
	assert.True(t, ok)
	assert.Equal(t, f.WorldId(), result.WorldId())
	assert.Equal(t, f.ChannelId(), result.ChannelId())
	assert.Equal(t, f.MapId(), result.MapId())
}

func TestProcessor_Exit(t *testing.T) {
	setupProcessorTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	l := testLogger()
	p := NewProcessor(l, ctx)

	characterId := uint32(20002)
	f := field.NewBuilder(1, 2, 100000000).Build()

	p.Enter(f, characterId)

	_, ok := GetRegistry().GetMap(ctx, characterId)
	assert.True(t, ok)

	p.Exit(f, characterId)

	_, ok = GetRegistry().GetMap(ctx, characterId)
	assert.False(t, ok)
}

func TestProcessor_GetMap(t *testing.T) {
	setupProcessorTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	l := testLogger()
	p := NewProcessor(l, ctx)

	characterId := uint32(20003)
	f := field.NewBuilder(1, 2, 100000000).Build()

	p.Enter(f, characterId)

	m, err := p.GetMap(characterId)
	assert.NoError(t, err)
	assert.Equal(t, f.WorldId(), m.WorldId())
	assert.Equal(t, f.ChannelId(), m.ChannelId())
	assert.Equal(t, f.MapId(), m.MapId())
}

func TestProcessor_GetMap_NotFound(t *testing.T) {
	setupProcessorTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	l := testLogger()
	p := NewProcessor(l, ctx)

	_, err := p.GetMap(99999998)
	assert.Error(t, err)
}

func TestProcessor_TransitionMap(t *testing.T) {
	setupProcessorTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	l := testLogger()
	p := NewProcessor(l, ctx)

	characterId := uint32(20004)
	f := field.NewBuilder(1, 2, 100000000).Build()

	p.Enter(f, characterId)

	newF := f.Clone().SetMapId(200000000).Build()
	p.TransitionMap(newF, characterId)

	result, ok := GetRegistry().GetMap(ctx, characterId)
	assert.True(t, ok)
	assert.Equal(t, newF.MapId(), result.MapId())
}

func TestProcessor_TransitionChannel(t *testing.T) {
	setupProcessorTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	l := testLogger()
	p := NewProcessor(l, ctx)

	characterId := uint32(20005)
	f := field.NewBuilder(1, 1, 100000000).Build()

	p.Enter(f, characterId)

	newF := f.Clone().SetChannelId(2).Build()
	p.TransitionChannel(newF, characterId)

	result, ok := GetRegistry().GetMap(ctx, characterId)
	assert.True(t, ok)
	assert.Equal(t, newF.ChannelId(), result.ChannelId())
}
