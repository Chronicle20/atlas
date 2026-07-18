package clientbound

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// Balloon is the CEmployee::SetBalloon block (v83 @0x50d897) shared by the
// hired-merchant SPAWN and UPDATE packets. The client reads MiniRoomType first;
// if it is 0 the balloon is torn down and nothing further is read, otherwise it
// reads the shop's mini-room serial, the balloon title, and three status bytes.
//
// The three status bytes (a/b/c at @0x50d897) drive the balloon's visitor
// display; their exact roles are not individually confirmed from IDA, so they are
// modelled as curVisitors/maxVisitors/spec — non-crashing, since none gate a read.
type Balloon struct {
	miniRoomType byte
	miniRoomSN   uint32
	title        string
	curVisitors  byte
	maxVisitors  byte
	spec         byte
}

func NewBalloon(miniRoomType byte, miniRoomSN uint32, title string, curVisitors byte, maxVisitors byte, spec byte) Balloon {
	return Balloon{
		miniRoomType: miniRoomType,
		miniRoomSN:   miniRoomSN,
		title:        title,
		curVisitors:  curVisitors,
		maxVisitors:  maxVisitors,
		spec:         spec,
	}
}

// NewEmptyBalloon signals the client to remove any balloon (MiniRoomType 0).
func NewEmptyBalloon() Balloon { return Balloon{} }

func (b Balloon) MiniRoomType() byte { return b.miniRoomType }
func (b Balloon) MiniRoomSN() uint32 { return b.miniRoomSN }
func (b Balloon) Title() string      { return b.title }

func (b Balloon) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(b.miniRoomType)
		if b.miniRoomType != 0 {
			w.WriteInt(b.miniRoomSN)
			w.WriteAsciiString(b.title)
			w.WriteByte(b.curVisitors)
			w.WriteByte(b.maxVisitors)
			w.WriteByte(b.spec)
		}
		return w.Bytes()
	}
}

func (b *Balloon) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		b.miniRoomType = r.ReadByte()
		if b.miniRoomType == 0 {
			return
		}
		b.miniRoomSN = r.ReadUint32()
		b.title = r.ReadAsciiString()
		b.curVisitors = r.ReadByte()
		b.maxVisitors = r.ReadByte()
		b.spec = r.ReadByte()
	}
}
