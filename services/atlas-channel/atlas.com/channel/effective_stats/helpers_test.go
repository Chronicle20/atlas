package effective_stats

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestMaxHpOrBase(t *testing.T) {
	tests := []struct {
		name      string
		effective uint32
		base      uint16
		want      uint16
	}{
		{"effective zero falls back to base", 0, 1500, 1500},
		{"effective in range used", 2000, 1500, 2000},
		{"effective above uint16 max clamped", math.MaxUint32, 1500, math.MaxUint16},
		{"effective exactly uint16 max preserved", math.MaxUint16, 1500, math.MaxUint16},
		{"both zero stays zero", 0, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaxHpOrBase(tt.effective, tt.base)
			if got != tt.want {
				t.Errorf("MaxHpOrBase(%d, %d) = %d, want %d", tt.effective, tt.base, got, tt.want)
			}
		})
	}
}

func TestResolveCharacterMaxes_PrefersEffective(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
			"data": {
				"type": "effective-stats",
				"id": "1234",
				"attributes": {"maxHP": 2500, "maxMP": 290}
			}
		}`))
	}))
	defer srv.Close()
	defer SetBaseURLForTest(srv.URL)()

	maxHp, maxMp := ResolveCharacterMaxes(logrus.New(), context.Background(), 0, 3, 1234, 2000, 240)
	require.Equal(t, uint16(2500), maxHp)
	require.Equal(t, uint16(290), maxMp)
}

func TestResolveCharacterMaxes_FallsBackOnFetchError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	defer SetBaseURLForTest(srv.URL)()

	maxHp, maxMp := ResolveCharacterMaxes(logrus.New(), context.Background(), 0, 3, 1234, 2000, 240)
	require.Equal(t, uint16(2000), maxHp)
	require.Equal(t, uint16(240), maxMp)
}

func TestResolveCharacterMaxes_FallsBackOnZeroResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
			"data": {
				"type": "effective-stats",
				"id": "1234",
				"attributes": {"maxHP": 0, "maxMP": 0}
			}
		}`))
	}))
	defer srv.Close()
	defer SetBaseURLForTest(srv.URL)()

	maxHp, maxMp := ResolveCharacterMaxes(logrus.New(), context.Background(), 0, 3, 1234, 2000, 240)
	require.Equal(t, uint16(2000), maxHp)
	require.Equal(t, uint16(240), maxMp)
}

func TestMaxMpOrBase(t *testing.T) {
	tests := []struct {
		name      string
		effective uint32
		base      uint16
		want      uint16
	}{
		{"effective zero falls back to base", 0, 240, 240},
		{"effective in range used", 290, 240, 290},
		{"effective above uint16 max clamped", math.MaxUint32, 240, math.MaxUint16},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaxMpOrBase(tt.effective, tt.base)
			if got != tt.want {
				t.Errorf("MaxMpOrBase(%d, %d) = %d, want %d", tt.effective, tt.base, got, tt.want)
			}
		})
	}
}
