package effective_stats

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestGetByCharacterId_RoundTrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.True(t, strings.HasSuffix(r.URL.Path, "/worlds/0/channels/3/characters/1234/stats"), "path: %s", r.URL.Path)
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
			"data": {
				"type": "effective-stats",
				"id": "1234",
				"attributes": {
					"strength": 50,
					"dexterity": 60,
					"luck": 120,
					"intelligence": 70,
					"maxHP": 5000,
					"maxMP": 1500,
					"weaponAttack": 80,
					"weaponDefense": 200,
					"magicAttack": 200,
					"magicDefense": 100,
					"accuracy": 250,
					"avoidability": 180,
					"speed": 130,
					"jump": 120
				}
			}
		}`))
	}))
	defer srv.Close()
	defer SetBaseURLForTest(srv.URL)()

	stats, err := NewProcessor(logrus.New(), context.Background()).GetByCharacterId(0, 3, 1234)
	require.NoError(t, err)
	require.Equal(t, uint32(120), stats.Luck)
	require.Equal(t, uint32(200), stats.MagicAttack)
}

func TestGetByCharacterId_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	defer SetBaseURLForTest(srv.URL)()

	_, err := NewProcessor(logrus.New(), context.Background()).GetByCharacterId(0, 3, 1234)
	require.Error(t, err)
}
