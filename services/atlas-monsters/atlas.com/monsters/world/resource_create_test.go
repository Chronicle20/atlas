package world

import (
	"atlas-monsters/monster"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	mapconst "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestCreateMonsterInMapSuppressedSpawnReturns204 drives POST
// /worlds/{w}/channels/{c}/maps/{m}/instances/{i}/monsters through the real
// resource router with spawnIfAbsent=true against a field that already holds
// the requested template. handleCreateMonsterInMap must not translate the
// resulting zero Model into a misleading 200 with a zero-valued monster
// body; it returns 204 No Content instead (task A7, Step 1/6 decision).
func TestCreateMonsterInMapSuppressedSpawnReturns204(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "suppressed spawn returns 204"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tenantId := uuid.New()
			ten, err := tenant.Create(tenantId, "GMS", 83, 1)
			require.NoError(t, err)
			ctx := tenant.WithContext(context.Background(), ten)

			worldId := world.Id(3)
			channelId := channel.Id(3)
			mapId := mapconst.Id(300000000)
			instanceId := uuid.Nil
			f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instanceId).Build()

			reg := monster.GetMonsterRegistry()
			reg.CreateMonster(ctx, ten, f, 9100013, 82, 200, 0, 0, 0, 100, 100, "", "")

			body, err := jsonapi.Marshal(&monster.RestModel{
				MonsterId:     9100013,
				X:             82,
				Y:             200,
				SpawnIfAbsent: true,
			})
			require.NoError(t, err)

			srv := httptest.NewServer(setupWorldRouter())
			defer srv.Close()

			url := fmt.Sprintf("%s/worlds/%d/channels/%d/maps/%d/instances/%s/monsters", srv.URL, worldId, channelId, mapId, instanceId)
			req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("TENANT_ID", tenantId.String())
			req.Header.Set("REGION", "GMS")
			req.Header.Set("MAJOR_VERSION", "83")
			req.Header.Set("MINOR_VERSION", "1")

			resp, err := (&http.Client{}).Do(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			require.Equal(t, http.StatusNoContent, resp.StatusCode)

			inField := reg.GetMonstersInMap(ten, f)
			require.Len(t, inField, 1, "the suppressed spawn must not have added a second monster")
		})
	}
}
