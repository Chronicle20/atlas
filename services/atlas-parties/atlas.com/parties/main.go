package main

import (
	"atlas-parties/character"
	"atlas-parties/logger"
	"atlas-parties/party"
	"atlas-parties/service"
	"atlas-parties/tracing"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
)

const serviceName = "atlas-parties"
const consumerGroupId = "Party Service"

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

	cm := consumer.GetManager()
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(party.CommandConsumer(l)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
	_, _ = cm.RegisterHandler(party.CreateCommandRegister(l))
	_, _ = cm.RegisterHandler(party.JoinCommandRegister(l))
	_, _ = cm.RegisterHandler(party.LeaveCommandRegister(l))
	_, _ = cm.RegisterHandler(party.ChangeLeaderCommandRegister(l))
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(character.StatusEventConsumer(l)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
	_, _ = cm.RegisterHandler(character.LoginStatusRegister(l))
	_, _ = cm.RegisterHandler(character.LogoutStatusRegister(l))

	server.CreateService(l, tdm.Context(), tdm.WaitGroup(), GetServer().GetPrefix(), party.InitResource(GetServer()))

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
