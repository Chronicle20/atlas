package session

import (
	"atlas-account/account"
	consumer2 "atlas-account/kafka/consumer"
	"atlas-account/kafka/producer"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"strings"
)

const (
	consumerNameLogout = "logout_account_command"
)

func CreateAccountSessionCommandConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerNameLogout)(EnvCommandTopicAccountLogout)(groupId)
	}
}

func handleLogoutAccountCommand(l logrus.FieldLogger, span opentracing.Span, command logoutCommand) {
	pi := producer.ProviderImpl(l)(span)
	l.Debugf("Received logout account command account [%d] from [%s].", command.AccountId, command.Issuer)

	ok := account.Get().Logout(account.AccountKey{TenantId: command.Tenant.Id, AccountId: command.AccountId}, account.ServiceKey{SessionId: command.SessionId, Service: account.Service(strings.ToUpper(command.Issuer))})
	if ok {
		_ = pi(EnvEventTopicAccountStatus)(loggedOutEventProvider()(command.Tenant, command.AccountId, ""))
		return
	}

	l.Debugf("Ignoring logout command from [%s] as account [%d] is not in correct state.", command.Issuer, command.AccountId)
}

func CreateAccountSessionRegister(l *logrus.Logger) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvCommandTopicAccountLogout)()
	return t, message.AdaptHandler(message.PersistentConfig(handleLogoutAccountCommand))
}
