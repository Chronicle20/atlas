package main

import (
	"atlas-guilds/database"
	"atlas-guilds/guild"
	"atlas-guilds/guild/character"
	"atlas-guilds/guild/member"
	"atlas-guilds/guild/title"
	character2 "atlas-guilds/kafka/consumer/character"
	guild2 "atlas-guilds/kafka/consumer/guild"
	"atlas-guilds/kafka/consumer/invite"
	"atlas-guilds/logger"
	"atlas-guilds/service"
	"atlas-guilds/tasks"
	"atlas-guilds/tracing"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
	"time"
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

	db := database.Connect(l, database.SetMigrations(guild.Migration, title.Migration, member.Migration, character.Migration))

	server.CreateService(l, tdm.Context(), tdm.WaitGroup(), GetServer().GetPrefix(), guild.InitResource(GetServer())(db))

	cm := consumer.GetManager()
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(guild2.CommandConsumer(l)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
	_, _ = cm.RegisterHandler(guild2.RequestCreateRegister(db)(l))
	_, _ = cm.RegisterHandler(guild2.CreationAgreementRegister(db)(l))
	_, _ = cm.RegisterHandler(guild2.ChangeEmblemRegister(db)(l))
	_, _ = cm.RegisterHandler(guild2.ChangeNoticeRegister(db)(l))
	_, _ = cm.RegisterHandler(guild2.LeaveRegister(db)(l))
	_, _ = cm.RegisterHandler(guild2.RequestInviteRegister(db)(l))
	_, _ = cm.RegisterHandler(guild2.ChangeTitlesRegister(db)(l))
	_, _ = cm.RegisterHandler(guild2.ChangeMemberTitleRegister(db)(l))
	_, _ = cm.RegisterHandler(guild2.RequestDisbandRegister(db)(l))
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(character2.StatusEventConsumer(l)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
	_, _ = cm.RegisterHandler(character2.LoginStatusRegister(l)(db))
	_, _ = cm.RegisterHandler(character2.LogoutStatusRegister(l)(db))
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(invite.StatusEventConsumer(l)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
	_, _ = cm.RegisterHandler(invite.AcceptedStatusEventRegister(l)(db))

	go tasks.Register(l, tdm.Context())(guild.NewTransitionTimeout(l, db, time.Second*time.Duration(35)))

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
