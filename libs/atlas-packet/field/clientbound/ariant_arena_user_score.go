package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const AriantArenaUserScoreWriter = "AriantArenaUserScore"

type AriantArenaUserScore struct {
	count byte
	name  string
	score uint32
}

func NewAriantArenaUserScore(count byte, name string, score uint32) AriantArenaUserScore {
	return AriantArenaUserScore{count: count, name: name, score: score}
}

func (m AriantArenaUserScore) Count() byte   { return m.count }
func (m AriantArenaUserScore) Name() string  { return m.name }
func (m AriantArenaUserScore) Score() uint32 { return m.score }

func (m AriantArenaUserScore) Operation() string { return AriantArenaUserScoreWriter }
func (m AriantArenaUserScore) String() string {
	return fmt.Sprintf("count [%d] name [%s] score [%d]", m.count, m.name, m.score)
}

func (m AriantArenaUserScore) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.count)
		w.WriteAsciiString(m.name)
		w.WriteInt(m.score)
		return w.Bytes()
	}
}

func (m *AriantArenaUserScore) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.count = r.ReadByte()
		m.name = r.ReadAsciiString()
		m.score = r.ReadUint32()
	}
}
