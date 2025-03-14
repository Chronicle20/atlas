package main

import (
	"atlas-pets/database"
	"atlas-pets/kafka/consumer/character"
	"atlas-pets/kafka/consumer/inventory"
	pet2 "atlas-pets/kafka/consumer/pet"
	"atlas-pets/logger"
	"atlas-pets/pet"
	"atlas-pets/service"
	"atlas-pets/tasks"
	"atlas-pets/tracing"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
	"os"
	"time"
)

const serviceName = "atlas-pets"
const consumerGroupId = "Pets Service"

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

	db := database.Connect(l, database.SetMigrations(pet.Migration))

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	character.InitConsumers(l)(cmf)(consumerGroupId)
	inventory.InitConsumers(l)(cmf)(consumerGroupId)
	pet2.InitConsumers(l)(cmf)(consumerGroupId)
	character.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)
	inventory.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)
	pet2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(pet.InitResource(GetServer())(db)).
		Run()

	go tasks.Register(l, tdm.Context())(pet.NewHungerTask(l, db, time.Minute*time.Duration(3)))

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
