package main

import (
	"atlas-rankings/ranking"
	"atlas-rankings/tasks"
	"context"
	"os"
	"time"

	"github.com/sirupsen/logrus"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	env "github.com/Chronicle20/atlas/libs/atlas-env"
	lock "github.com/Chronicle20/atlas/libs/atlas-lock"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	service "github.com/Chronicle20/atlas/libs/atlas-service"
)

const serviceName = "atlas-rankings"

const baseTick = time.Minute

type Server struct {
	baseUrl string
	prefix  string
}

func (s Server) GetBaseURL() string {
	return s.baseUrl
}

func (s Server) GetPrefix() string {
	return s.prefix
}

func GetServer() Server {
	return Server{baseUrl: "", prefix: "/api/"}
}

func main() {
	rt := service.Bootstrap(serviceName, service.WithEnvironmentRegistry(serviceName))
	l := rt.Logger()

	db := database.Connect(l, database.SetMigrations(ranking.Migration))
	rc := atlas.Connect(l)

	server.RegisterTransientErrorClassifier(func(err error) bool {
		if database.IsTransientConnectionError(err) {
			database.CountTransient(err)
			return true
		}
		return false
	})

	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(ranking.InitResource(GetServer())(db)).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	// tasks/recompute.go sits outside env-domain-guard's permitted atlas-env
	// import list (main.go, kafka/, rest/, socket/), so this pod's
	// environment identity is threaded in as a plain function value rather
	// than the package importing atlas-env itself. Without it,
	// RecomputeTask's per-tenant configuration lookup and character read
	// would resolve through RootUrlFor's legacy-baseline fallback and
	// silently hit main's environment instead of this pod's.
	registerRecompute := func(l logrus.FieldLogger, ctx context.Context) {
		tasks.Register(l, ctx)(tasks.NewRecomputeTask(l, ctx, db, baseTick, func(ctx context.Context) context.Context {
			return env.WithContext(ctx, env.Self())
		}))
	}

	if leaderEnabled(l) {
		ttl := leaderTTL(l)
		le, err := lock.New(rc, "rankings-recompute",
			lock.WithTTL(ttl),
			lock.WithRefreshInterval(leaderRefresh(l, ttl)),
			lock.WithBackoff(leaderBackoff(l)),
			lock.WithLogger(l),
		)
		if err != nil {
			l.WithError(err).Fatal("Unable to construct LeaderElection.")
		}
		routine.Go(l, rt.Context(), func(_ context.Context) {
			err := le.Run(rt.Context(), func(leaderCtx context.Context) {
				registerRecompute(l, leaderCtx)
				<-leaderCtx.Done()
			})
			if err != nil {
				l.WithError(err).Errorf("LeaderElection.Run exited with error.")
			}
		})
	} else {
		l.Warnf("RANKINGS_LEADER_ELECTION_ENABLED=false — recompute runs unconditionally on this pod.")
		registerRecompute(l, rt.Context())
	}

	rt.Wait()
}
