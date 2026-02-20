package main

import (
	database "github.com/Chronicle20/atlas-database"
	assetConsumer "atlas-quest/kafka/consumer/asset"
	characterConsumer "atlas-quest/kafka/consumer/character"
	monsterConsumer "atlas-quest/kafka/consumer/monster"
	questConsumer "atlas-quest/kafka/consumer/quest"
	"atlas-quest/logger"
	"atlas-quest/quest"
	"atlas-quest/quest/progress"
	"github.com/Chronicle20/atlas-service"
	"atlas-quest/tracing"
	"os"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
)

const serviceName = "atlas-quest"
const consumerGroupId = "Quest Service"

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

	tc, err := tracing.InitTracer(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	db := database.Connect(l, database.SetMigrations(quest.Migration, progress.Migration))

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())

	// Quest command consumer
	questConsumer.InitConsumers(l)(cmf)(consumerGroupId)
	if err := questConsumer.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	// Monster kill event consumer (for quest progress tracking)
	monsterConsumer.InitConsumers(l)(cmf)(consumerGroupId)
	if err := monsterConsumer.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	// Asset/item creation event consumer (for quest progress tracking)
	assetConsumer.InitConsumers(l)(cmf)(consumerGroupId)
	if err := assetConsumer.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	// Character status event consumer (for map change quest progress)
	characterConsumer.InitConsumers(l)(cmf)(consumerGroupId)
	if err := characterConsumer.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	// Create the service with the router
	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(quest.InitResource(GetServer())(db)).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
