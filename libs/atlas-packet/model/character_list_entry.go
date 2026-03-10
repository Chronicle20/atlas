package model

import (
	"context"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type CharacterListEntry struct {
	statistics  CharacterStatistics
	avatar      Avatar
	gm          bool
	rank        uint32
	rankMove    uint32
	jobRank     uint32
	jobRankMove uint32
}

func NewCharacterListEntry(statistics CharacterStatistics, avatar Avatar, gm bool, rank uint32, rankMove uint32, jobRank uint32, jobRankMove uint32) CharacterListEntry {
	return CharacterListEntry{
		statistics:  statistics,
		avatar:      avatar,
		gm:          gm,
		rank:        rank,
		rankMove:    rankMove,
		jobRank:     jobRank,
		jobRankMove: jobRankMove,
	}
}

func (m CharacterListEntry) Statistics() CharacterStatistics { return m.statistics }
func (m CharacterListEntry) Gm() bool                       { return m.gm }
func (m CharacterListEntry) Rank() uint32                    { return m.rank }
func (m CharacterListEntry) RankMove() uint32                { return m.rankMove }
func (m CharacterListEntry) JobRank() uint32                 { return m.jobRank }
func (m CharacterListEntry) JobRankMove() uint32             { return m.jobRankMove }

func (m CharacterListEntry) Write(l logrus.FieldLogger, ctx context.Context, w *response.Writer, options map[string]interface{}, viewAll bool) {
	t := tenant.MustFromContext(ctx)
	w.WriteByteArray(m.statistics.Encode(l, ctx)(options))
	w.WriteByteArray(m.avatar.Encode(l, ctx)(options))
	if !viewAll {
		w.WriteByte(0)
	}
	if m.gm {
		w.WriteByte(0)
		return
	}

	if t.Region() == "GMS" && t.MajorVersion() <= 28 {
		w.WriteInt(1) // auto select first character
	}

	w.WriteByte(1) // world rank enabled
	w.WriteInt(m.rank)
	w.WriteInt(m.rankMove)
	w.WriteInt(m.jobRank)
	w.WriteInt(m.jobRankMove)
}

func (m *CharacterListEntry) Read(l logrus.FieldLogger, ctx context.Context, r *request.Reader, options map[string]interface{}, viewAll bool) {
	t := tenant.MustFromContext(ctx)
	m.statistics.Decode(l, ctx)(r, options)
	m.avatar.Decode(l, ctx)(r, options)
	if !viewAll {
		_ = r.ReadByte()
	}

	rankEnabled := r.ReadByte()
	if rankEnabled == 0 {
		m.gm = true
		return
	}

	if t.Region() == "GMS" && t.MajorVersion() <= 28 {
		_ = r.ReadUint32() // auto select
	}

	m.rank = r.ReadUint32()
	m.rankMove = r.ReadUint32()
	m.jobRank = r.ReadUint32()
	m.jobRankMove = r.ReadUint32()
}
