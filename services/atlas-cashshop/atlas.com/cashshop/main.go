package main

import (
	"atlas-cashshop/database"
	"atlas-cashshop/kafka/consumer/character"
	"atlas-cashshop/logger"
	"atlas-cashshop/service"
	"atlas-cashshop/tracing"
	"atlas-cashshop/wallet"
	"atlas-cashshop/wishlist"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
	"os"
)

const serviceName = "atlas-cashshop"
const consumerGroupId = "Cash Shop Service"

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
	l := logger.CreateLogger(serviceName)
	l.Infoln("Starting main service.")

	tdm := service.GetTeardownManager()

	tc, err := tracing.InitTracer(l)(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	db := database.Connect(l, database.SetMigrations(wallet.Migration, wishlist.Migration))

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	character.InitConsumers(l)(cmf)(consumerGroupId)
	character.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(wallet.InitResource(GetServer())(db)).
		AddRouteInitializer(wishlist.InitResource(GetServer())(db)).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
