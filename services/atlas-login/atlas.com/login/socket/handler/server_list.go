package handler

import (
	"atlas-login/channel"
	"atlas-login/session"
	"atlas-login/socket/writer"
	"atlas-login/world"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const ServerListRequestHandle = "ServerListRequestHandle"

func ServerListRequestHandleFunc(l logrus.FieldLogger, span opentracing.Span, wp writer.Producer) func(s session.Model, r *request.Reader) {
	return func(s session.Model, _ *request.Reader) {
		issueServerInformation(l, span, wp)(s)
	}
}

func issueServerInformation(l logrus.FieldLogger, span opentracing.Span, wp writer.Producer) func(s session.Model) {
	return func(s session.Model) {
		ws, err := world.GetAll(l, span)
		if err != nil {
			l.WithError(err).Errorf("Retrieving worlds")
			return
		}

		cls, err := channel.GetChannelLoadByWorld(l, span)
		if err != nil {
			l.WithError(err).Errorf("Retrieving channel load")
			return
		}

		nws := combine(l, ws, cls)
		respondToSession(l, wp)(s, nws)
	}
}

func combine(l logrus.FieldLogger, ws []world.Model, cls map[int][]channel.Load) []world.Model {
	var nws = make([]world.Model, 0)

	for _, x := range ws {
		if val, ok := cls[int(x.Id())]; ok {
			nws = append(nws, x.SetChannelLoad(val))
		} else {
			l.Errorf("Processing world without a channel load")
			nws = append(nws, x)
		}
	}
	return nws
}

func respondToSession(l logrus.FieldLogger, wp writer.Producer) func(ms session.Model, ws []world.Model) {
	return func(ms session.Model, ws []world.Model) {
		announceServerList(l, wp)(ws, ms)
		announceLastWorld(l, wp)(ms)
		announceRecommendedWorlds(l, wp)(ws, ms)
	}
}

func announceRecommendedWorlds(l logrus.FieldLogger, wp writer.Producer) func(ws []world.Model, ms session.Model) {
	serverListRecommendationFunc := session.Announce(wp)(writer.ServerListRecommendations)

	return func(ws []world.Model, ms session.Model) {
		var rs = make([]world.Recommendation, 0)
		for _, x := range ws {
			if x.Recommended() {
				rs = append(rs, x.Recommendation())
			}
		}
		err := serverListRecommendationFunc(ms, writer.ServerListRecommendationsBody(l)(rs))
		if err != nil {
			l.WithError(err).Errorf("Unable to issue recommended worlds")
		}
	}
}

func announceLastWorld(l logrus.FieldLogger, wp writer.Producer) func(ms session.Model) {
	selectWorldFunc := session.Announce(wp)(writer.SelectWorld)
	return func(ms session.Model) {
		err := selectWorldFunc(ms, writer.SelectWorldBody(l)(0))
		if err != nil {
			l.WithError(err).Errorf("Unable to identify the last world a account was logged into")
		}
	}
}

func announceServerList(l logrus.FieldLogger, wp writer.Producer) func(ws []world.Model, ms session.Model) {
	serverListEntryFunc := session.Announce(wp)(writer.ServerListEntry)
	serverListEndFunc := session.Announce(wp)(writer.ServerListEnd)
	return func(ws []world.Model, ms session.Model) {
		for _, x := range ws {
			err := serverListEntryFunc(ms, writer.ServerListEntryBody(l, ms.Tenant())(x.Id(), x.Name(), x.State(), x.EventMessage(), x.ChannelLoad()))
			if err != nil {
				l.WithError(err).Errorf("Unable to write server list entry for %d", x.Id())
			}
		}
		err := serverListEndFunc(ms, writer.ServerListEndBody(l))
		if err != nil {
			l.WithError(err).Errorf("Unable to complete writing the server list")
		}
	}
}
