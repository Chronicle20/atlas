package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const MtsOperation2Writer = "MtsOperation2"

// packet-audit:fname CITC::OnQueryCashResult
type MtsOperation2 struct {
	cash        uint32
	maplePoints uint32
}

func NewMtsOperation2(cash uint32, maplePoints uint32) MtsOperation2 {
	return MtsOperation2{cash: cash, maplePoints: maplePoints}
}

func (m MtsOperation2) Cash() uint32        { return m.cash }
func (m MtsOperation2) MaplePoints() uint32 { return m.maplePoints }

func (m MtsOperation2) Operation() string { return MtsOperation2Writer }
func (m MtsOperation2) String() string {
	return fmt.Sprintf("cash [%d] maplePoints [%d]", m.cash, m.maplePoints)
}

func (m MtsOperation2) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.cash)
		w.WriteInt(m.maplePoints)
		return w.Bytes()
	}
}

func (m *MtsOperation2) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.cash = r.ReadUint32()
		m.maplePoints = r.ReadUint32()
	}
}
