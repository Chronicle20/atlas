package main

import (
	"atlas-notes/database"
	"atlas-notes/kafka/consumer/character"
	note_consumer "atlas-notes/kafka/consumer/note"
	"atlas-notes/logger"
	"atlas-notes/note"
	"atlas-notes/service"
	"atlas-notes/tracing"
	"os"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
)

const serviceName = "atlas-notes"
const consumerGroupId = "Notes Service"

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

	// Connect to the database
	db := database.Connect(l, database.SetMigrations(note.Migration))

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	character.InitConsumers(l)(cmf)(consumerGroupId)
	note_consumer.InitConsumers(l)(cmf)(consumerGroupId)
	character.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)
	note_consumer.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(note.InitializeRoutes(GetServer())(db)).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
