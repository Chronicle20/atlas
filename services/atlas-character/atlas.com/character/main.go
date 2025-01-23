package main

import (
	"atlas-character/character"
	"atlas-character/database"
	"atlas-character/equipable"
	"atlas-character/inventory"
	"atlas-character/inventory/item"
	character2 "atlas-character/kafka/consumer/character"
	inventory2 "atlas-character/kafka/consumer/inventory"
	session2 "atlas-character/kafka/consumer/session"
	"atlas-character/logger"
	"atlas-character/service"
	"atlas-character/session"
	"atlas-character/tasks"
	"atlas-character/tracing"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
	"time"
)
import _ "net/http/pprof"

const serviceName = "atlas-character"
const consumerGroupId = "Character Service"

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
		prefix:  "/api/cos/",
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

	db := database.Connect(l, database.SetMigrations(character.Migration, inventory.Migration, item.Migration, equipable.Migration))

	if service.GetMode() == service.Mixed {
		cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
		character2.InitConsumers(l)(cmf)(consumerGroupId)
		inventory2.InitConsumers(l)(cmf)(consumerGroupId)
		session2.InitConsumers(l)(cmf)(consumerGroupId)
		character2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)
		inventory2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)
		session2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)
	}

	server.CreateService(l, tdm.Context(), tdm.WaitGroup(), GetServer().GetPrefix(), character.InitResource(GetServer())(db), inventory.InitResource(GetServer())(db))

	go tasks.Register(l, tdm.Context())(session.NewTimeout(l, db, time.Millisecond*time.Duration(5000)))

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
