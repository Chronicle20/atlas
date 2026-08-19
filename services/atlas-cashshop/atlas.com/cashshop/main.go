package main

import (
	"atlas-cashshop/cashshop/inventory"
	"atlas-cashshop/cashshop/inventory/asset"
	"atlas-cashshop/cashshop/inventory/compartment"
	"atlas-cashshop/coupon"
	"atlas-cashshop/coupon/batch"
	"atlas-cashshop/coupon/redemption"
	"atlas-cashshop/kafka/consumer/account"
	"atlas-cashshop/kafka/consumer/cashshop"
	"atlas-cashshop/kafka/consumer/character"
	"atlas-cashshop/purchaserecord"
	"atlas-cashshop/surprise/opening"
	"atlas-cashshop/wallet"
	"atlas-cashshop/wishlist"
	"context"
	"os"

	compartment2 "atlas-cashshop/kafka/consumer/cashshop/compartment"
	itemConsumer "atlas-cashshop/kafka/consumer/item"
	walletConsumer "atlas-cashshop/kafka/consumer/wallet"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	outboxlib "github.com/Chronicle20/atlas/libs/atlas-outbox"
	redis "github.com/Chronicle20/atlas/libs/atlas-redis"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	service "github.com/Chronicle20/atlas/libs/atlas-service"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

const serviceName = "atlas-cashshop"

var consumerGroupId = consumergroup.Resolve("Cash Shop Service")

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
	return Server{
		baseUrl: "",
		prefix:  "/api/",
	}
}

func main() {
	rt := service.Bootstrap(serviceName, service.WithEnvironmentRegistry(serviceName))
	l := rt.Logger()

	db := database.Connect(l, database.SetMigrations(wallet.Migration, wishlist.Migration, compartment.Migration, asset.Migration, opening.Migration, purchaserecord.Migration, coupon.Migration, batch.Migration, redemption.Migration, outboxlib.Migration, database.IdempotencyMigration))

	// Seed cash_purchase_records from cash_assets history for accounts that
	// bought before purchaserecord existed. Idempotent, so it runs on every
	// boot; a failure here is a degraded answer (missing purchase history),
	// not a reason to refuse to serve the cash shop.
	if backfilled, err := purchaserecord.Backfill(l, db); err != nil {
		l.WithError(err).Warn("Failed to backfill purchase records from cash_assets history.")
	} else if backfilled > 0 {
		l.Infof("Backfilled %d purchase record(s) from existing cash_assets history.", backfilled)
	}

	// ACCEPT/RELEASE claim an idempotency key so an at-least-once redelivery
	// cannot duplicate or double-release a cash asset (task-208).
	database.StartIdempotencySweeper(l, rt.Context(), db, database.DefaultIdempotencyRetention, database.DefaultIdempotencySweep)

	// The coupon redemption rate limiter counts failed attempts per account in
	// Redis. It fails open, so a Redis outage degrades brute-force braking
	// rather than blocking redemptions.
	coupon.InitLimiter(redis.Connect(l))

	// Boot the outbox drainer: publishes the transactional outbox to Kafka.
	// Leadership is gated by a postgres advisory lock — replicas are safe.
	publisher := outboxlib.NewTopicWriterPool()
	drainer := outboxlib.NewDrainer(l, db, publisher, outboxlib.WithDSN(database.DSN()))
	routine.Go(l, rt.Context(), func(_ context.Context) {
		drainer.Run(rt.Context())
	})
	rt.TeardownFunc(func() {
		drainer.Stop()
		publisher.Close()
	})

	server.RegisterTransientErrorClassifier(func(err error) bool {
		if database.IsTransientConnectionError(err) {
			database.CountTransient(err)
			return true
		}
		return false
	})

	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
	account.InitConsumers(l)(cmf)(consumerGroupId)
	character.InitConsumers(l)(cmf)(consumerGroupId)
	compartment2.InitConsumers(l)(cmf)(consumerGroupId)
	cashshop.InitConsumers(l)(cmf)(consumerGroupId)
	itemConsumer.InitConsumers(l)(cmf)(consumerGroupId)
	walletConsumer.InitConsumers(l)(cmf)(consumerGroupId)
	if err := account.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := character.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := compartment2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := cashshop.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := itemConsumer.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := walletConsumer.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(wallet.InitResource(GetServer())(db)).
		AddRouteInitializer(wishlist.InitResource(GetServer())(db)).
		AddRouteInitializer(compartment.InitResource(GetServer())(db)).
		AddRouteInitializer(asset.InitResource(GetServer())(db)).
		AddRouteInitializer(inventory.InitResource(GetServer())(db)).
		AddRouteInitializer(coupon.InitResource(GetServer())(db)).
		AddRouteInitializer(batch.InitResource(GetServer())(db)).
		AddRouteInitializer(redemption.InitResource(GetServer())(db)).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	rt.Wait()
}
