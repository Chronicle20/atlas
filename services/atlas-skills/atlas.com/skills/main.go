package main

import (
	"atlas-skills/database"
	"atlas-skills/kafka/consumer/character"
	macro2 "atlas-skills/kafka/consumer/macro"
	skill2 "atlas-skills/kafka/consumer/skill"
	"atlas-skills/logger"
	"atlas-skills/macro"
	"atlas-skills/service"
	"atlas-skills/skill"
	"atlas-skills/tasks"
	"atlas-skills/tracing"
	"os"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
)

const serviceName = "atlas-skills"
const consumerGroupId = "Skills Service"

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

	db := database.Connect(l, database.SetMigrations(skill.Migration, macro.Migration))

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	skill2.InitConsumers(l)(cmf)(consumerGroupId)
	character.InitConsumers(l)(cmf)(consumerGroupId)
	macro2.InitConsumers(l)(cmf)(consumerGroupId)
	skill2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)
	character.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)
	macro2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)

	go tasks.Register(tasks.NewExpirationTask(l, db, 1000))

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(skill.InitResource(GetServer())(db)).
		AddRouteInitializer(macro.InitResource(GetServer())(db)).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
