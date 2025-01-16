package main

import (
	"atlas-buddies/buddy"
	"atlas-buddies/database"
	invite2 "atlas-buddies/kafka/consumer/invite"
	list2 "atlas-buddies/kafka/consumer/list"
	"atlas-buddies/list"
	"atlas-buddies/logger"
	"atlas-buddies/service"
	"atlas-buddies/tracing"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
)

const serviceName = "atlas-buddies"
const consumerGroupId = "Buddy Service"

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

	db := database.Connect(l, database.SetMigrations(list.Migration, buddy.Migration))

	cm := consumer.GetManager()
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(list2.CommandConsumer(l)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
	_, _ = cm.RegisterHandler(list2.CreateCommandRegister(l)(db))
	_, _ = cm.RegisterHandler(list2.RequestAddCommandRegister(l)(db))
	_, _ = cm.RegisterHandler(list2.RequestDeleteCommandRegister(l)(db))
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(invite2.StatusEventConsumer(l)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
	_, _ = cm.RegisterHandler(invite2.AcceptedStatusEventRegister(l)(db))
	_, _ = cm.RegisterHandler(invite2.RejectedStatusEventRegister(l)(db))

	server.CreateService(l, tdm.Context(), tdm.WaitGroup(), GetServer().GetPrefix(), list.InitResource(GetServer())(db))

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
