package portal

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testCtx(t *testing.T) (context.Context, uuid.UUID) {
	t.Helper()
	tenantId := uuid.New()
	ten, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	return tenant.WithContext(context.Background(), ten), tenantId
}

// staticRequest builds a requests.Request that returns ms without any I/O.
func staticRequest(ms []RestModel) requests.Request[[]RestModel] {
	return func(l logrus.FieldLogger, ctx context.Context) ([]RestModel, error) {
		return ms, nil
	}
}

func TestGetInMapByName_CachesWholeList(t *testing.T) {
	ctx, _ := testCtx(t)
	l, _ := test.NewNullLogger()

	mapId := _map.Id(100000000)
	calls := 0
	orig := requestInMapFn
	requestInMapFn = func(ctx context.Context, id _map.Id) requests.Request[[]RestModel] {
		calls++
		return staticRequest([]RestModel{
			{Id: "1", Name: "sp", Target: "tp", Type: 2, X: 100, Y: 200, TargetMapId: 104040000},
			{Id: "2", Name: "other", Target: "tp2", Type: 2, X: 10, Y: 20, TargetMapId: 104040001},
		})
	}
	t.Cleanup(func() { requestInMapFn = orig })

	p := NewProcessor(l, ctx)

	m1, err := p.GetInMapByName(mapId, "sp")
	require.NoError(t, err)
	require.Equal(t, "sp", m1.Name())

	m2, err := p.GetInMapByName(mapId, "other")
	require.NoError(t, err)
	require.Equal(t, "other", m2.Name())

	require.Equal(t, 1, calls, "expected exactly 1 REST fetch for the whole list, cached across calls")
}

func TestGetInMapByName_TenantScoped(t *testing.T) {
	l, _ := test.NewNullLogger()
	mapId := _map.Id(100000000)
	calls := 0

	orig := requestInMapFn
	requestInMapFn = func(ctx context.Context, id _map.Id) requests.Request[[]RestModel] {
		calls++
		return staticRequest([]RestModel{
			{Id: "1", Name: "sp", Target: "tp", Type: 2, X: 100, Y: 200, TargetMapId: 104040000},
		})
	}
	t.Cleanup(func() { requestInMapFn = orig })

	ctx1, _ := testCtx(t)
	ctx2, _ := testCtx(t)

	p1 := NewProcessor(l, ctx1)
	p2 := NewProcessor(l, ctx2)

	_, err := p1.GetInMapByName(mapId, "sp")
	require.NoError(t, err)
	_, err = p2.GetInMapByName(mapId, "sp")
	require.NoError(t, err)

	require.Equal(t, 2, calls, "each tenant must trigger its own REST fetch")
}

func TestGetInMapByName_NotFound(t *testing.T) {
	ctx, _ := testCtx(t)
	l, _ := test.NewNullLogger()
	mapId := _map.Id(100000000)
	calls := 0

	orig := requestInMapFn
	requestInMapFn = func(ctx context.Context, id _map.Id) requests.Request[[]RestModel] {
		calls++
		return staticRequest([]RestModel{
			{Id: "1", Name: "sp", Target: "tp", Type: 2, X: 100, Y: 200, TargetMapId: 104040000},
		})
	}
	t.Cleanup(func() { requestInMapFn = orig })

	p := NewProcessor(l, ctx)

	_, err := p.GetInMapByName(mapId, "sp")
	require.NoError(t, err)
	require.Equal(t, 1, calls)

	_, err = p.GetInMapByName(mapId, "does-not-exist")
	require.Error(t, err)
	require.Equal(t, 1, calls, "a cache hit that misses on filter must not perform a second fetch")
}

func TestModelAccessors(t *testing.T) {
	rm := RestModel{
		Id:          "7",
		Name:        "sp",
		Target:      "tp",
		Type:        2,
		X:           100,
		Y:           200,
		TargetMapId: 104040000,
		ScriptName:  "",
	}
	m, err := Extract(rm)
	require.NoError(t, err)
	require.Equal(t, uint32(7), m.Id())
	require.Equal(t, "sp", m.Name())
	require.Equal(t, "tp", m.Target())
	require.Equal(t, uint8(2), m.Type())
	require.Equal(t, int16(100), m.X())
	require.Equal(t, int16(200), m.Y())
	require.Equal(t, _map.Id(104040000), m.TargetMapId())
	require.Equal(t, "", m.ScriptName())
}
