package main

import (
	"atlas-parties/character"
	"atlas-parties/kafka/consumer/invite"
	party2 "atlas-parties/kafka/consumer/party"
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
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(party2.CommandConsumer(l)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
	_, _ = cm.RegisterHandler(party2.CreateCommandRegister(l))
	_, _ = cm.RegisterHandler(party2.JoinCommandRegister(l))
	_, _ = cm.RegisterHandler(party2.LeaveCommandRegister(l))
	_, _ = cm.RegisterHandler(party2.ChangeLeaderCommandRegister(l))
	_, _ = cm.RegisterHandler(party2.RequestInviteCommandRegister(l))
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(character.StatusEventConsumer(l)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
	_, _ = cm.RegisterHandler(character.LoginStatusRegister(l))
	_, _ = cm.RegisterHandler(character.LogoutStatusRegister(l))
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(invite.StatusEventConsumer(l)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
	_, _ = cm.RegisterHandler(invite.AcceptedStatusEventRegister(l))

	server.CreateService(l, tdm.Context(), tdm.WaitGroup(), GetServer().GetPrefix(), party.InitResource(GetServer()))

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
