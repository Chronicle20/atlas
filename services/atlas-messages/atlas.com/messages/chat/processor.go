package chat

import (
	"context"
	"sort"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Processor interface {
	// Capture appends one player-authored chat line to the sender's bounded
	// buffer. Failures are the caller's to log; capture must never block or
	// fail the chat flow.
	Capture(f field.Model, senderId uint32, senderName string, chatType string, text string) error
	// RecentInvolving returns lines authored by any of the listed characters,
	// merged and sorted ascending by timestamp.
	RecentInvolving(characterIds []uint32) ([]Line, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx, t: tenant.MustFromContext(ctx)}
}

func (p *ProcessorImpl) Capture(f field.Model, senderId uint32, senderName string, chatType string, text string) error {
	r := GetRegistry()
	if r == nil {
		return nil
	}
	return r.Append(p.ctx, p.t, Line{
		Timestamp:  time.Now().UnixMilli(),
		SenderId:   senderId,
		SenderName: senderName,
		ChatType:   chatType,
		Text:       text,
		WorldId:    byte(f.WorldId()),
		ChannelId:  byte(f.ChannelId()),
		MapId:      uint32(f.MapId()),
	})
}

func (p *ProcessorImpl) RecentInvolving(characterIds []uint32) ([]Line, error) {
	r := GetRegistry()
	if r == nil {
		return nil, nil
	}
	all := make([]Line, 0)
	for _, id := range characterIds {
		lines, err := r.RecentBySender(p.ctx, p.t, id)
		if err != nil {
			return nil, err
		}
		all = append(all, lines...)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Timestamp < all[j].Timestamp })
	return all, nil
}
