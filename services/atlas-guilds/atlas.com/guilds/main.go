package main

import (
	"atlas-guilds/database"
	"atlas-guilds/guild"
	"atlas-guilds/guild/member"
	"atlas-guilds/guild/title"
	guild2 "atlas-guilds/kafka/consumer/guild"
	"atlas-guilds/logger"
	"atlas-guilds/service"
	"atlas-guilds/tracing"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
)

const serviceName = "atlas-guilds"
const consumerGroupId = "Guilds Service"

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

	db := database.Connect(l, database.SetMigrations(guild.Migration, title.Migration, member.Migration))

	server.CreateService(l, tdm.Context(), tdm.WaitGroup(), GetServer().GetPrefix(), guild.InitResource(GetServer())(db))

	cm := consumer.GetManager()
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(guild2.CommandConsumer(l)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
	_, _ = cm.RegisterHandler(guild2.RequestCreateRegister(db)(l))
	_, _ = cm.RegisterHandler(guild2.CreationAgreementRegister(db)(l))

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
