package character

import (
	monster2 "atlas-maps/map/monster"
	"os"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

// TestMain also initializes the shared spawn point registry against
// miniredis. map.ProcessorImpl.Exit calls monster2.GetRegistry().RearmOneTime
// on every field-empties path, and this package's handlers reach Exit
// through ExitAndEmit/TransitionChannelAndEmit, so the registry singleton
// must be live before those tests run.
func TestMain(m *testing.M) {
	producertest.InstallNoop()

	mr, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer mr.Close()

	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	monster2.InitRegistry(rc)

	os.Exit(m.Run())
}
