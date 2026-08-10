package trade

import (
	"atlas-trades/configuration"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/miniroom"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// perTenantConfig is a configProvider that answers from the tenant in context,
// standing in for the atlas-tenants-backed configuration.Registry.
type perTenantConfig struct {
	byTenant map[uuid.UUID]configuration.Model
}

func (c perTenantConfig) Get(_ logrus.FieldLogger, ctx context.Context) configuration.Model {
	t, err := tenant.FromContext(ctx)()
	if err != nil {
		return configuration.DefaultConfig()
	}
	if cfg, ok := c.byTenant[t.Id()]; ok {
		return cfg
	}
	return configuration.DefaultConfig()
}

func tickerLogger() *logrus.Logger {
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	return l
}

func tickerTenant(t *testing.T, name string) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.NewSHA1(uuid.NameSpaceOID, []byte(t.Name()+name)), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create() error = %v", err)
	}
	return tm
}

// TestReservationRefreshIntervalIsAThirdOfTheTtl pins design §5.3's pace: a hold
// must survive two consecutively missed passes before it can lapse, so a bug that
// paced the ticker at the TTL itself would expire holds under live trades.
func TestReservationRefreshIntervalIsAThirdOfTheTtl(t *testing.T) {
	if got := ReservationRefreshInterval(configuration.DefaultConfig()); got != 100*time.Second {
		t.Errorf("default interval: got %s, want 100s (a third of the 300s default TTL)", got)
	}
	cfg := configuration.DefaultConfig().WithReservationTtl(90 * time.Second)
	if got := ReservationRefreshInterval(cfg); got != 30*time.Second {
		t.Errorf("interval for a 90s TTL: got %s, want 30s", got)
	}
}

// TestReservationRefreshIntervalIsFloored pins that a zero or absurdly short
// configured TTL cannot turn the ticker into a busy loop that republishes every
// hold in the process continuously.
func TestReservationRefreshIntervalIsFloored(t *testing.T) {
	if got := ReservationRefreshInterval(configuration.DefaultConfig().WithReservationTtl(0)); got != minReservationRefreshInterval {
		t.Errorf("interval for a zero TTL: got %s, want the %s floor", got, minReservationRefreshInterval)
	}
	if got := ReservationRefreshInterval(configuration.DefaultConfig().WithReservationTtl(600 * time.Millisecond)); got != minReservationRefreshInterval {
		t.Errorf("interval for a 600ms TTL: got %s, want the %s floor", got, minReservationRefreshInterval)
	}
}

// TestTenantsListsOnlyTenantsWithLiveRooms pins what the ticker enumerates. The
// registry deletes a tenant's rooms but leaves the per-tenant map behind, so a
// tenant that traded once must stop drawing refresh passes when its last room
// ends — otherwise every tenant the process has ever seen is walked forever.
func TestTenantsListsOnlyTenantsWithLiveRooms(t *testing.T) {
	live := tickerTenant(t, "live")
	drained := tickerTenant(t, "drained")
	reg := GetRegistry()

	liveRoom := NewBuilder(miniroom.Trade, 100, "Owner", testField(t)).Build()
	if err := reg.Create(live, liveRoom); err != nil {
		t.Fatalf("create live room: %v", err)
	}
	t.Cleanup(func() { reg.Remove(live, liveRoom.Id()) })

	drainedRoom := NewBuilder(miniroom.Trade, 200, "Owner", testField(t)).Build()
	if err := reg.Create(drained, drainedRoom); err != nil {
		t.Fatalf("create drained room: %v", err)
	}
	reg.Remove(drained, drainedRoom.Id())

	var sawLive, sawDrained bool
	for _, tm := range reg.Tenants() {
		if tm.Id() == live.Id() {
			sawLive = true
		}
		if tm.Id() == drained.Id() {
			sawDrained = true
		}
	}
	if !sawLive {
		t.Error("a tenant with a live room was not listed")
	}
	if sawDrained {
		t.Error("a tenant whose last room ended is still listed")
	}
}

// TestRefreshAllReservationsPassesEveryTenantUnderItsOwnTenantContext pins that
// each pass is tenant-scoped. A pass run under the wrong tenant — or under none —
// reads an empty registry partition and refreshes nothing, so every hold in the
// process would silently lapse.
func TestRefreshAllReservationsPassesEveryTenantUnderItsOwnTenantContext(t *testing.T) {
	a := tickerTenant(t, "a")
	b := tickerTenant(t, "b")

	seen := make([]uuid.UUID, 0, 2)
	_, err := refreshAllReservations(tickerLogger(), context.Background(), []tenant.Model{a, b}, perTenantConfig{}, func(tctx context.Context) error {
		tm, terr := tenant.FromContext(tctx)()
		if terr != nil {
			t.Errorf("pass ran with no tenant in context: %v", terr)
			return nil
		}
		seen = append(seen, tm.Id())
		return nil
	})
	if err != nil {
		t.Fatalf("refresh all: %v", err)
	}
	if len(seen) != 2 || seen[0] != a.Id() || seen[1] != b.Id() {
		t.Errorf("passes: got %v, want one per tenant in order [%s %s]", seen, a.Id(), b.Id())
	}
}

// TestRefreshAllReservationsPacesToTheShortestLiveTtl pins the pacing rule. A
// tenant with a short TTL paced by another tenant's long one would lose its holds
// between passes; the reverse merely refreshes more often than needed, which is
// free.
func TestRefreshAllReservationsPacesToTheShortestLiveTtl(t *testing.T) {
	slow := tickerTenant(t, "slow")
	fast := tickerTenant(t, "fast")
	cfgs := perTenantConfig{byTenant: map[uuid.UUID]configuration.Model{
		slow.Id(): configuration.DefaultConfig(),
		fast.Id(): configuration.DefaultConfig().WithReservationTtl(60 * time.Second),
	}}

	next, err := refreshAllReservations(tickerLogger(), context.Background(), []tenant.Model{slow, fast}, cfgs, func(context.Context) error { return nil })
	if err != nil {
		t.Fatalf("refresh all: %v", err)
	}
	if next != 20*time.Second {
		t.Errorf("next pass: got %s, want 20s — a third of the shortest live TTL (60s)", next)
	}
}

// TestRefreshAllReservationsWithNoLiveTenantsPacesToTheDefault pins the idle
// pace: with nothing to refresh the ticker must still come back on the shipped
// cadence, because the next room can be created a millisecond later.
func TestRefreshAllReservationsWithNoLiveTenantsPacesToTheDefault(t *testing.T) {
	next, err := refreshAllReservations(tickerLogger(), context.Background(), nil, perTenantConfig{}, func(context.Context) error {
		t.Error("a pass ran with no live tenants")
		return nil
	})
	if err != nil {
		t.Fatalf("refresh all: %v", err)
	}
	if want := ReservationRefreshInterval(configuration.DefaultConfig()); next != want {
		t.Errorf("idle pace: got %s, want the default %s", next, want)
	}
}

// TestRefreshAllReservationsKeepsGoingAfterATenantFails pins that one tenant's
// failure does not cost the others their holds, and that the failure is still
// reported so the ticker can log it.
func TestRefreshAllReservationsKeepsGoingAfterATenantFails(t *testing.T) {
	bad := tickerTenant(t, "bad")
	good := tickerTenant(t, "good")

	var passes int
	next, err := refreshAllReservations(tickerLogger(), context.Background(), []tenant.Model{bad, good}, perTenantConfig{}, func(tctx context.Context) error {
		passes++
		tm, _ := tenant.FromContext(tctx)()
		if tm.Id() == bad.Id() {
			return errors.New("outbox unavailable")
		}
		return nil
	})
	if passes != 2 {
		t.Errorf("passes: got %d, want 2 — a failing tenant must not abort the pass", passes)
	}
	if err == nil {
		t.Error("refresh all reported success despite a failing tenant")
	}
	if want := ReservationRefreshInterval(configuration.DefaultConfig()); next != want {
		t.Errorf("next pass after a failure: got %s, want the default %s — the ticker must still be paced", next, want)
	}
}

// TestRefreshAllReservationsPassContextCarriesCancellation pins that the process
// context reaches each pass, so a shutdown cancels an in-flight refresh instead
// of detaching it.
func TestRefreshAllReservationsPassContextCarriesCancellation(t *testing.T) {
	tm := tickerTenant(t, "cancel")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := refreshAllReservations(tickerLogger(), ctx, []tenant.Model{tm}, perTenantConfig{}, func(tctx context.Context) error {
		if tctx.Err() == nil {
			t.Error("the pass context did not inherit the cancelled process context")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("refresh all: %v", err)
	}
}

// TestReservationRefreshRunsItsFirstPassImmediately pins the boot window. The
// pace is reported BY a pass, so before one has run there is no honest interval
// — and the shipped default (100s) is the wrong guess for any tenant whose TTL
// is under 300s: one configured at 60s that staged an item at boot would lose
// its hold at t≈65s while the first pass was still waiting for t=100s.
//
// The assertion is that the first pass runs on a timescale nowhere near the
// default interval, which is what makes the boot window closed.
func TestReservationRefreshRunsItsFirstPassImmediately(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ran := make(chan time.Duration, 1)
	start := time.Now()
	routine.Go(tickerLogger(), ctx, func(rctx context.Context) {
		runReservationRefresh(tickerLogger(), rctx, reservationRefreshInitialDelay, func(context.Context) (time.Duration, error) {
			select {
			case ran <- time.Since(start):
			default:
			}
			<-rctx.Done()
			return time.Hour, nil
		})
	})

	select {
	case elapsed := <-ran:
		if elapsed > time.Second {
			t.Errorf("first pass ran after %s, want promptly — a tenant with a sub-default TTL loses its holds inside that window", elapsed)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the first pass never ran")
	}
}

// TestReservationRefreshPacesLaterPassesFromTheReportedInterval pins that the
// loop honours what each pass reports rather than a fixed period — the whole
// reason it is a timer and not a ticker.
func TestReservationRefreshPacesLaterPassesFromTheReportedInterval(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fired := make(chan struct{}, 8)
	routine.Go(tickerLogger(), ctx, func(rctx context.Context) {
		runReservationRefresh(tickerLogger(), rctx, reservationRefreshInitialDelay, func(context.Context) (time.Duration, error) {
			select {
			case fired <- struct{}{}:
			default:
			}
			return time.Millisecond, nil
		})
	})

	// Three fires inside a second is only possible if the loop re-armed at the
	// 1ms the pass reported; the default pace would deliver one.
	for i := 0; i < 3; i++ {
		select {
		case <-fired:
		case <-time.After(5 * time.Second):
			t.Fatalf("pass %d never ran — the loop did not re-arm at the reported interval", i+1)
		}
	}
}

// TestReservationRefreshStopsOnContextCancellation pins the shutdown path
// main.go's teardown waits on: the loop must return when the process context is
// cancelled, or `<-refreshStopped` blocks shutdown forever.
func TestReservationRefreshStopsOnContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})
	routine.Go(tickerLogger(), ctx, func(rctx context.Context) {
		defer close(stopped)
		runReservationRefresh(tickerLogger(), rctx, time.Hour, func(context.Context) (time.Duration, error) {
			return time.Hour, nil
		})
	})

	cancel()
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("the refresh loop did not return after its context was cancelled")
	}
}

// compile-time assurance the ticker's config stand-in satisfies the seam it
// replaces.
var _ configProvider = perTenantConfig{}
