package guild

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/guild/clientbound"
	"github.com/sirupsen/logrus"
)

// bbsOptions mirrors the tenant template's GuildBBS options.operations map
// (BBS_THREAD_LIST=6 / BBS_THREAD=7 / BBS_ENTRY_NOT_FOUND=8 — version-stable
// across gms_v83/v84/v87/v95, docs/packets/dispatchers/guild_bbs.yaml).
func bbsOptions() map[string]interface{} {
	return map[string]interface{}{
		"operations": map[string]interface{}{
			GuildBBSOperationThreadList:    float64(6),
			GuildBBSOperationThread:        float64(7),
			GuildBBSOperationEntryNotFound: float64(8),
		},
	}
}

// TestGuildBBSBodyResolvesMode confirms each BBS body function config-resolves
// its mode byte from the operations table (the leading wire byte), NOT a literal
// and NOT the 99 fallback that signals a missing/misconfigured table.
func TestGuildBBSBodyResolvesMode(t *testing.T) {
	l, _ := logrustest()
	opts := bbsOptions()

	cases := []struct {
		name string
		emit func(logrus.FieldLogger) []byte
		want byte
	}{
		{"thread list", func(l logrus.FieldLogger) []byte {
			return GuildBBSThreadListBody(nil, nil, 0)(l, nil)(opts)
		}, 6},
		{"thread", func(l logrus.FieldLogger) []byte {
			return GuildBBSThreadBody(1, 100, 0, "t", "m", 0, []clientbound.BBSReply{})(l, nil)(opts)
		}, 7},
		{"entry not found", func(l logrus.FieldLogger) []byte {
			return GuildBBSEntryNotFoundBody()(l, nil)(opts)
		}, 8},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := c.emit(l)
			if len(got) == 0 {
				t.Fatalf("%s: empty output", c.name)
			}
			if got[0] != c.want {
				t.Fatalf("%s: leading mode byte = %d, want %d (99 = unresolved)", c.name, got[0], c.want)
			}
		})
	}
}

func logrustest() (logrus.FieldLogger, *logrus.Logger) {
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	return l, l
}
