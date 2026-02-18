package messenger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMemberFilterMatch(t *testing.T) {
	m, err := NewBuilder().
		SetId(1).
		AddMember(100, 0).
		AddMember(200, 1).
		Build()
	assert.NoError(t, err)

	assert.True(t, MemberFilter(100)(m))
	assert.True(t, MemberFilter(200)(m))
}

func TestMemberFilterNoMatch(t *testing.T) {
	m, err := NewBuilder().
		SetId(1).
		AddMember(100, 0).
		AddMember(200, 1).
		Build()
	assert.NoError(t, err)

	assert.False(t, MemberFilter(999)(m))
}

func TestMemberFilterEmptyMembers(t *testing.T) {
	m, err := NewBuilder().
		SetId(1).
		Build()
	assert.NoError(t, err)

	assert.False(t, MemberFilter(100)(m))
}

func TestGetByIdSuccess(t *testing.T) {
	setupTestRegistry(t)
	ctx := createTestCtx(t)

	created := GetRegistry().Create(ctx, 100)

	result, err := GetById(ctx)(created.Id())

	assert.NoError(t, err)
	assert.Equal(t, created.Id(), result.Id())
	assert.Len(t, result.Members(), 1)
	assert.Equal(t, uint32(100), result.Members()[0].Id())
}

func TestGetByIdNotFound(t *testing.T) {
	setupTestRegistry(t)
	ctx := createTestCtx(t)

	_, err := GetById(ctx)(999999999)

	assert.Error(t, err)
}

func TestGetSliceEmpty(t *testing.T) {
	setupTestRegistry(t)
	ctx := createTestCtx(t)

	result, err := GetSlice(ctx)()

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestGetSliceWithData(t *testing.T) {
	setupTestRegistry(t)
	ctx := createTestCtx(t)

	m1 := GetRegistry().Create(ctx, 100)
	m2 := GetRegistry().Create(ctx, 200)

	result, err := GetSlice(ctx)()

	assert.NoError(t, err)
	assert.Len(t, result, 2)

	ids := make(map[uint32]bool)
	for _, m := range result {
		ids[m.Id()] = true
	}
	assert.True(t, ids[m1.Id()])
	assert.True(t, ids[m2.Id()])
}

func TestGetSliceWithFilter(t *testing.T) {
	setupTestRegistry(t)
	ctx := createTestCtx(t)

	m1 := GetRegistry().Create(ctx, 100)
	_ = GetRegistry().Create(ctx, 200)

	result, err := GetSlice(ctx)(MemberFilter(100))

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, m1.Id(), result[0].Id())
}

func TestGetSliceWithFilterNoMatch(t *testing.T) {
	setupTestRegistry(t)
	ctx := createTestCtx(t)

	_ = GetRegistry().Create(ctx, 100)
	_ = GetRegistry().Create(ctx, 200)

	result, err := GetSlice(ctx)(MemberFilter(999))

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestProcessorImplGetById(t *testing.T) {
	setupTestRegistry(t)
	ctx := createTestCtx(t)

	created := GetRegistry().Create(ctx, 100)

	proc := NewProcessor(nil, ctx)
	result, err := proc.GetById(created.Id())

	assert.NoError(t, err)
	assert.Equal(t, created.Id(), result.Id())
}

func TestProcessorImplGetSlice(t *testing.T) {
	setupTestRegistry(t)
	ctx := createTestCtx(t)

	_ = GetRegistry().Create(ctx, 100)
	_ = GetRegistry().Create(ctx, 200)

	proc := NewProcessor(nil, ctx)
	result, err := proc.GetSlice()

	assert.NoError(t, err)
	assert.Len(t, result, 2)
}
