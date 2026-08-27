package writer

import (
	"atlas-login/character"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestToCharacterListEntry_SpawnPoint pins both halves of the task-272 fix on
// the CHARACTER_LIST path: the model value reaches the packet statistics
// struct, and the uint32 -> byte narrowing at the wire boundary truncates
// rather than erroring. Truncation above 255 is a pre-existing property of the
// wire format (one byte), asserted here so it is documented rather than latent.
//
// toCharacterListEntry calls location.GetField, which fails fast with no MAPS
// base URL configured; the entry then renders mapId = 0. That is expected and
// irrelevant to this assertion.
func TestToCharacterListEntry_SpawnPoint(t *testing.T) {
	tests := []struct {
		name string
		set  uint32
		want byte
	}{
		{name: "in range", set: 7, want: 7},
		{name: "truncates above 255", set: 256, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := pt.CreateContext("GMS", 83, 1)
			l, _ := testlog.NewNullLogger()

			c := character.NewBuilder().
				SetId(99).
				SetSp("0").
				SetSpawnPoint(tt.set).
				Build()

			entry := toCharacterListEntry(l, ctx, c, false)

			if got := entry.Statistics().SpawnPoint(); got != tt.want {
				t.Errorf("SpawnPoint() = %d, want %d", got, tt.want)
			}
		})
	}
}
