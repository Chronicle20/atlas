package handler

import (
	"atlas-login/session"
	"atlas-login/socket/writer"
	"atlas-login/world"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
	"sort"
)

const ServerListRequestHandle = "ServerListRequestHandle"

func ServerListRequestHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader) {
	return func(s session.Model, r *request.Reader) {
		issueServerInformation(l, ctx, wp)(s)
	}
}

func issueServerInformation(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model) {
	return func(s session.Model) {
		ws, err := world.GetAll(l, ctx, world.ChannelLoadDecorator(l, ctx))
		if err != nil {
			l.WithError(err).Errorf("Retrieving worlds")
			return
		}
		sort.Slice(ws, func(i, j int) bool {
			return ws[i].Id() < ws[j].Id()
		})

		if len(ws) == 0 {
			l.Warnf("Responding with no valid worlds.")
		}

		respondToSession(l, wp)(s, ws)
	}
}

func respondToSession(l logrus.FieldLogger, wp writer.Producer) func(ms session.Model, ws []world.Model) {
	return func(ms session.Model, ws []world.Model) {
		announceServerList(l, wp)(ws, ms)
		announceLastWorld(l, wp)(ms)
		announceRecommendedWorlds(l, wp)(ws, ms)
	}
}

func announceRecommendedWorlds(l logrus.FieldLogger, wp writer.Producer) func(ws []world.Model, ms session.Model) {
	serverListRecommendationFunc := session.Announce(l)(wp)(writer.ServerListRecommendations)

	return func(ws []world.Model, ms session.Model) {
		var rs = make([]world.Recommendation, 0)
		for _, x := range ws {
			if x.Recommended() {
				rs = append(rs, x.Recommendation())
			}
		}
		err := serverListRecommendationFunc(ms, writer.ServerListRecommendationsBody(rs))
		if err != nil {
			l.WithError(err).Errorf("Unable to issue recommended worlds")
		}
	}
}

func announceLastWorld(l logrus.FieldLogger, wp writer.Producer) func(ms session.Model) {
	selectWorldFunc := session.Announce(l)(wp)(writer.SelectWorld)
	return func(ms session.Model) {
		err := selectWorldFunc(ms, writer.SelectWorldBody(0))
		if err != nil {
			l.WithError(err).Errorf("Unable to identify the last world a account was logged into")
		}
	}
}

func announceServerList(l logrus.FieldLogger, wp writer.Producer) func(ws []world.Model, ms session.Model) {
	serverListEntryFunc := session.Announce(l)(wp)(writer.ServerListEntry)
	serverListEndFunc := session.Announce(l)(wp)(writer.ServerListEnd)
	return func(ws []world.Model, ms session.Model) {
		for _, x := range ws {
			err := serverListEntryFunc(ms, writer.ServerListEntryBody(ms.Tenant())(x.Id(), x.Name(), x.State(), x.EventMessage(), x.ChannelLoad()))
			if err != nil {
				l.WithError(err).Errorf("Unable to write server list entry for %d", x.Id())
			}
		}
		err := serverListEndFunc(ms, writer.ServerListEndBody)
		if err != nil {
			l.WithError(err).Errorf("Unable to complete writing the server list")
		}
	}
}
