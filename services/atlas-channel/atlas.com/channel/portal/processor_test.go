package portal

import (
	portalData "atlas-channel/data/portal"
	portalDataMock "atlas-channel/data/portal/mock"
	"atlas-channel/movement"
	movementMock "atlas-channel/movement/mock"
	"atlas-channel/position"
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const (
	enterInnerTestCharacterId = uint32(42)
	enterInnerTestMapId       = _map.Id(104040000)
)

func enterInnerTestField() field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(1), enterInnerTestMapId).Build()
}

func enterInnerTestCtx(t *testing.T) context.Context {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 95, 1)
	require.NoError(t, err)
	return tenant.WithContext(context.Background(), ten)
}

func enterInnerTestPortal(t *testing.T, id uint32, name string, target string, targetMapId _map.Id, x int16, y int16) portalData.Model {
	t.Helper()
	m, err := portalData.Extract(portalData.RestModel{
		Id:          fmt.Sprintf("%d", id),
		Name:        name,
		Target:      target,
		Type:        2,
		X:           x,
		Y:           y,
		TargetMapId: targetMapId,
	})
	require.NoError(t, err)
	return m
}

// enterInnerFixture bundles the shared fixture values every TestEnterInner
// row starts from.
type enterInnerFixture struct {
	sourcePortal   portalData.Model
	destPortal     portalData.Model
	claimedX       int16
	claimedY       int16
	claimedTargetX int16
	claimedTargetY int16
}

func newEnterInnerFixture() enterInnerFixture {
	return enterInnerFixture{
		claimedX:       100,
		claimedY:       200,
		claimedTargetX: 300,
		claimedTargetY: -50,
	}
}

func TestEnterInner(t *testing.T) {
	tests := []struct {
		name             string
		mutate           func(t *testing.T, f *enterInnerFixture, pd *portalDataMock.ProcessorMock)
		seedRegistry     func(t *testing.T, ctx context.Context)
		expectTeleport   bool
		expectTeleportTo [2]int16
	}{
		{
			name: "source portal unresolvable",
			mutate: func(t *testing.T, f *enterInnerFixture, pd *portalDataMock.ProcessorMock) {
				pd.GetInMapByNameFunc = func(mapId _map.Id, name string) (portalData.Model, error) {
					if name == "sp" {
						return portalData.Model{}, fmt.Errorf("no portal named [sp]")
					}
					return f.destPortal, nil
				}
			},
			expectTeleport: false,
		},
		{
			name: "source targetMapId is the sentinel",
			mutate: func(t *testing.T, f *enterInnerFixture, pd *portalDataMock.ProcessorMock) {
				f.sourcePortal = enterInnerTestPortal(t, 1, "sp", "tp", _map.EmptyMapId, 100, 200)
			},
			expectTeleport: false,
		},
		{
			name: "source targetMapId is a different map",
			mutate: func(t *testing.T, f *enterInnerFixture, pd *portalDataMock.ProcessorMock) {
				f.sourcePortal = enterInnerTestPortal(t, 1, "sp", "tp", enterInnerTestMapId+1, 100, 200)
			},
			expectTeleport: false,
		},
		{
			name: "source target empty",
			mutate: func(t *testing.T, f *enterInnerFixture, pd *portalDataMock.ProcessorMock) {
				f.sourcePortal = enterInnerTestPortal(t, 1, "sp", "", enterInnerTestMapId, 100, 200)
			},
			expectTeleport: false,
		},
		{
			name: "destination portal unresolvable",
			mutate: func(t *testing.T, f *enterInnerFixture, pd *portalDataMock.ProcessorMock) {
				pd.GetInMapByNameFunc = func(mapId _map.Id, name string) (portalData.Model, error) {
					if name == "tp" {
						return portalData.Model{}, fmt.Errorf("no portal named [tp]")
					}
					return f.sourcePortal, nil
				}
			},
			expectTeleport: false,
		},
		{
			name: "last known position beyond threshold",
			seedRegistry: func(t *testing.T, ctx context.Context) {
				position.GetRegistry().Put(tenant.MustFromContext(ctx), enterInnerTestCharacterId, position.Position{X: 100 + maxPortalEntryDistance + 1, Y: 200})
			},
			expectTeleport: false,
		},
		{
			name: "last known position at the threshold",
			seedRegistry: func(t *testing.T, ctx context.Context) {
				position.GetRegistry().Put(tenant.MustFromContext(ctx), enterInnerTestCharacterId, position.Position{X: 100 + maxPortalEntryDistance, Y: 200})
			},
			expectTeleport:   true,
			expectTeleportTo: [2]int16{300, -50},
		},
		{
			name: "claimed target disagrees with destination portal",
			mutate: func(t *testing.T, f *enterInnerFixture, pd *portalDataMock.ProcessorMock) {
				f.claimedTargetX = 999
			},
			seedRegistry: func(t *testing.T, ctx context.Context) {
				position.GetRegistry().Put(tenant.MustFromContext(ctx), enterInnerTestCharacterId, position.Position{X: 100, Y: 200})
			},
			expectTeleport: false,
		},
		{
			name:             "last-position registry miss",
			expectTeleport:   true,
			expectTeleportTo: [2]int16{300, -50},
		},
		{
			name: "happy path adopts server coordinates",
			mutate: func(t *testing.T, f *enterInnerFixture, pd *portalDataMock.ProcessorMock) {
				f.claimedX = 9999
				f.claimedY = 9999
			},
			expectTeleport:   true,
			expectTeleportTo: [2]int16{300, -50},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := enterInnerTestCtx(t)
			fx := newEnterInnerFixture()
			fx.sourcePortal = enterInnerTestPortal(t, 1, "sp", "tp", enterInnerTestMapId, 100, 200)
			fx.destPortal = enterInnerTestPortal(t, 2, "tp", "sp", enterInnerTestMapId, 300, -50)

			pd := &portalDataMock.ProcessorMock{
				GetInMapByNameFunc: func(mapId _map.Id, name string) (portalData.Model, error) {
					if name == "sp" {
						return fx.sourcePortal, nil
					}
					if name == "tp" {
						return fx.destPortal, nil
					}
					return portalData.Model{}, fmt.Errorf("no portal named [%s]", name)
				},
			}

			if tc.mutate != nil {
				tc.mutate(t, &fx, pd)
			}

			if tc.seedRegistry != nil {
				tc.seedRegistry(t, ctx)
			} else {
				position.GetRegistry().Put(tenant.MustFromContext(ctx), enterInnerTestCharacterId, position.Position{X: 100, Y: 200})
			}
			t.Cleanup(func() {
				position.GetRegistry().Clear(tenant.MustFromContext(ctx), enterInnerTestCharacterId)
			})

			l, hook := test.NewNullLogger()

			teleportCalled := false
			var teleportedX, teleportedY int16
			mv := &movementMock.ProcessorMock{
				TeleportCharacterFunc: func(f field.Model, characterId uint32, x int16, y int16) error {
					teleportCalled = true
					teleportedX = x
					teleportedY = y
					return nil
				},
			}
			origMovementProcessor := newMovementProcessor
			newMovementProcessor = func(l logrus.FieldLogger, ctx context.Context) movement.Processor {
				return mv
			}
			t.Cleanup(func() { newMovementProcessor = origMovementProcessor })

			p := &ProcessorImpl{l: l, ctx: ctx, pd: pd}

			err := p.EnterInner(enterInnerTestField(), enterInnerTestCharacterId, "sp", fx.claimedX, fx.claimedY, fx.claimedTargetX, fx.claimedTargetY)
			require.NoError(t, err)

			require.Equal(t, tc.expectTeleport, teleportCalled)
			if tc.expectTeleport {
				require.Equal(t, tc.expectTeleportTo[0], teleportedX)
				require.Equal(t, tc.expectTeleportTo[1], teleportedY)
			} else {
				require.NotNil(t, hook.LastEntry(), "expected a refusal to log")
				require.Equal(t, logrus.WarnLevel, hook.LastEntry().Level)
			}
		})
	}
}
