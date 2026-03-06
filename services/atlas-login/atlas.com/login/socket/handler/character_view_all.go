package handler

import (
	"atlas-login/character"
	"atlas-login/session"
	"atlas-login/socket/writer"
	"atlas-login/world"
	"context"

	world2 "github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterViewAllHandle = "CharacterViewAllHandle"

func CharacterViewAllHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		var gameStartMode byte
		var nexonPassport string
		var machineId string
		var gameRoomClient uint32
		var gameStartMode2 byte

		if t.Region() == "GMS" && t.MajorVersion() > 83 {
			gameStartMode = r.ReadByte()
			nexonPassport = r.ReadAsciiString()
			machineId = r.ReadAsciiString()
			gameRoomClient = r.ReadUint32()
			gameStartMode2 = r.ReadByte()
		}
		l.Debugf("Processing request to view all characters. GameStartMode [%d], NexonPassport [%s], MachineId [%s], GameRoomClient [%d], GameStartMode2 [%d]", gameStartMode, nexonPassport, machineId, gameRoomClient, gameStartMode2)

		ws, err := world.NewProcessor(l, ctx).GetAll()
		if err != nil {
			l.Debugf("Unable to retrieve available worlds.")
			err = session.Announce(l)(ctx)(wp)(writer.CharacterViewAll)(writer.CharacterViewAllErrorBody())(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write view error.")
			}
			return
		}

		var wcs = make(map[world2.Id][]character.Model)
		var count int
		for _, w := range ws {
			var cs []character.Model
			cp := character.NewProcessor(l, ctx)
			cs, err = cp.GetForWorld(cp.InventoryDecorator())(s.AccountId(), w.Id())
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve characters for account [%d] in world [%d].", s.AccountId(), w.Id())
			}
			count += len(cs)
			wcs[w.Id()] = cs
		}

		l.Debugf("Located [%d] characters for account [%d].", count, s.AccountId())
		if count == 0 {
			err = session.Announce(l)(ctx)(wp)(writer.CharacterViewAll)(writer.CharacterViewAllSearchFailedBody())(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write search failed.")
			}
			return
		}

		err = session.Announce(l)(ctx)(wp)(writer.CharacterViewAll)(writer.CharacterViewAllCountBody(uint32(len(ws)), uint32(count)))(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to write count.")
			return
		}

		for w, cs := range wcs {
			err = session.Announce(l)(ctx)(wp)(writer.CharacterViewAll)(writer.CharacterViewAllCharacterBody(w, cs))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write search failed.")
			}
		}

		return
	}
}
