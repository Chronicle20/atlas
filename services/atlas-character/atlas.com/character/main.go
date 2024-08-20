package main

import (
	"atlas-character/character"
	"atlas-character/database"
	"atlas-character/equipable"
	"atlas-character/inventory"
	"atlas-character/inventory/item"
	"atlas-character/logger"
	"atlas-character/service"
	"atlas-character/session"
	"atlas-character/tracing"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
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

	tc, err := tracing.InitTracer(l)(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	db := database.Connect(l, database.SetMigrations(character.Migration, inventory.Migration, item.Migration, equipable.Migration))

	cm := consumer.GetManager()
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(inventory.EquipItemCommandConsumer(l)(consumerGroupId))
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(inventory.UnequipItemCommandConsumer(l)(consumerGroupId))
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(inventory.MoveItemCommandConsumer(l)(consumerGroupId))
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(inventory.DropItemCommandConsumer(l)(consumerGroupId))
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(session.StatusEventConsumer(l)(consumerGroupId))
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(character.CommandConsumer(l)(consumerGroupId))
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(character.MovementEventConsumer(l)(consumerGroupId))
	_, _ = cm.RegisterHandler(inventory.EquipItemRegister(l, db))
	_, _ = cm.RegisterHandler(inventory.UnequipItemRegister(l, db))
	_, _ = cm.RegisterHandler(inventory.MoveItemRegister(l, db))
	_, _ = cm.RegisterHandler(inventory.DropItemRegister(l, db))
	_, _ = cm.RegisterHandler(session.StatusEventRegister(l, db))
	_, _ = cm.RegisterHandler(character.ChangeMapCommandRegister(l, db))
	_, _ = cm.RegisterHandler(character.MovementEventRegister(l))

	server.CreateService(l, tdm.Context(), tdm.WaitGroup(), GetServer().GetPrefix(), character.InitResource(GetServer())(db), inventory.InitResource(GetServer())(db))

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
