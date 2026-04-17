package handler

import (
	"atlas-login/session"
	model2 "atlas-login/socket/model"
	"atlas-login/socket/writer"
	"atlas-login/world"
	"context"
	"sort"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	loginCB "github.com/Chronicle20/atlas/libs/atlas-packet/login/clientbound"
	loginSB "github.com/Chronicle20/atlas/libs/atlas-packet/login/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func ServerListRequestHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := loginSB.ServerListRequest{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		_ = announceServerInformation(l)(ctx)(wp)(s)
	}
}

func announceServerInformation(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) model.Operator[session.Model] {
		ws, err := world.NewProcessor(l, ctx).GetAll()
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve worlds to display to session.")
		}
		sort.Slice(ws, func(i, j int) bool {
			return ws[i].Id() < ws[j].Id()
		})

		if len(ws) == 0 {
			l.Warnf("Responding with no valid worlds.")
		}

		return func(wp writer.Producer) model.Operator[session.Model] {
			return model.ThenOperator(announceServerList(l)(ctx)(wp)(ws), model.Operators[session.Model](announceLastWorld(l)(ctx)(wp), announceRecommendedWorlds(l)(ctx)(wp)(ws)))
		}
	}
}

func announceRecommendedWorlds(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(ws []world.Model) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(ws []world.Model) model.Operator[session.Model] {
		return func(wp writer.Producer) func(ws []world.Model) model.Operator[session.Model] {
			return func(ws []world.Model) model.Operator[session.Model] {
				return func(s session.Model) error {
					var rs = make([]model2.Recommendation, 0)
					for _, x := range ws {
						if x.Recommended() {
							rs = append(rs, model2.NewWorldRecommendation(x.Id(), x.RecommendedMessage()))
						}
					}
					err := session.Announce(l)(ctx)(wp)(loginCB.ServerListRecommendationsWriter)(writer.ServerListRecommendationsBody(rs))(s)
					if err != nil {
						l.WithError(err).Errorf("Unable to issue recommended worlds")
						return err
					}
					return nil
				}
			}
		}
	}
}

func announceLastWorld(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) model.Operator[session.Model] {
		return func(wp writer.Producer) model.Operator[session.Model] {
			return func(s session.Model) error {
				err := session.Announce(l)(ctx)(wp)(loginCB.SelectWorldWriter)(loginCB.NewSelectWorld(uint32(0)).Encode)(s)
				if err != nil {
					l.WithError(err).Errorf("Unable to identify the last world a account was logged into")
					return err
				}
				return nil
			}
		}
	}
}

func announceServerList(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(ws []world.Model) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(ws []world.Model) model.Operator[session.Model] {
		return func(wp writer.Producer) func(ws []world.Model) model.Operator[session.Model] {
			return func(ws []world.Model) model.Operator[session.Model] {
				return func(s session.Model) error {
					for _, x := range ws {
						var cls []model2.Load
						for _, c := range x.Channels() {
							cls = append(cls, model2.NewChannelLoad(c.ChannelId(), c.CurrentCapacity()))
						}

						err := session.Announce(l)(ctx)(wp)(loginCB.ServerListEntryWriter)(writer.ServerListEntryBody(x.Id(), x.Name(), x.State(), x.EventMessage(), cls))(s)
						if err != nil {
							l.WithError(err).Errorf("Unable to write server list entry for [%d]", x.Id())
						}
					}
					err := session.Announce(l)(ctx)(wp)(loginCB.ServerListEndWriter)(writer.ServerListEndBody())(s)
					if err != nil {
						l.WithError(err).Errorf("Unable to complete writing the server list")
						return err
					}
					return nil
				}
			}
		}
	}
}
