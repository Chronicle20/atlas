package environment

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func newTestProcessor(t *testing.T) (Processor, field.Model) {
	l, _ := test.NewNullLogger()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)
	return NewProcessor(l, ctx), newTestField()
}

func TestProcessorSetRejectsBlankName(t *testing.T) {
	p, f := newTestProcessor(t)

	_, err := p.Set(f, field.ObjectKindEnvironment, "", 1)
	require.Error(t, err)
	require.Equal(t, "environment object name must not be blank", err.Error())
	require.Len(t, p.GetAll(f), 0)
}

func TestProcessorSetRejectsWhitespaceName(t *testing.T) {
	p, f := newTestProcessor(t)

	_, err := p.Set(f, field.ObjectKindEnvironment, "   ", 1)
	require.Error(t, err)
	require.Equal(t, "environment object name must not be blank", err.Error())
	require.Len(t, p.GetAll(f), 0)
}

func TestProcessorSetReturnsEntry(t *testing.T) {
	p, f := newTestProcessor(t)

	entry, err := p.Set(f, field.ObjectKindObstacle, "obs3", 2)
	require.NoError(t, err)
	require.Equal(t, ObjectEntry{Kind: field.ObjectKindObstacle, Name: "obs3", State: 2}, entry)
}

func TestProcessorSetIsIdempotent(t *testing.T) {
	p, f := newTestProcessor(t)

	_, err := p.Set(f, field.ObjectKindObstacle, "obs3", 2)
	require.NoError(t, err)
	_, err = p.Set(f, field.ObjectKindObstacle, "obs3", 2)
	require.NoError(t, err)

	require.Len(t, p.GetAll(f), 1)
}

func TestProcessorResetReturnsClearedAndEmpties(t *testing.T) {
	p, f := newTestProcessor(t)

	_, err := p.Set(f, field.ObjectKindObstacle, "a", 1)
	require.NoError(t, err)
	_, err = p.Set(f, field.ObjectKindEnvironment, "b", 2)
	require.NoError(t, err)

	cleared := p.Reset(f)
	require.Equal(t, []ObjectEntry{
		{Kind: field.ObjectKindObstacle, Name: "a", State: 1},
		{Kind: field.ObjectKindEnvironment, Name: "b", State: 2},
	}, cleared)

	require.Len(t, p.GetAll(f), 0)
}

func TestProcessorResetOnUntrackedFieldReturnsEmpty(t *testing.T) {
	p, f := newTestProcessor(t)

	cleared := p.Reset(f)
	require.Len(t, cleared, 0)
	require.NotNil(t, cleared)
}

func TestProcessorGetAllUntracked(t *testing.T) {
	p, f := newTestProcessor(t)

	got := p.GetAll(f)
	require.Len(t, got, 0)
	require.NotNil(t, got)
}
