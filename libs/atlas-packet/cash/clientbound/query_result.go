package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CashQueryResultWriter = "CashShopCashQueryResult"

type QueryResult struct {
	credit  uint32
	points  uint32
	prepaid uint32
}

func NewCashQueryResult(credit uint32, points uint32, prepaid uint32) QueryResult {
	return QueryResult{credit: credit, points: points, prepaid: prepaid}
}

func (m QueryResult) Credit() uint32  { return m.credit }
func (m QueryResult) Points() uint32  { return m.points }
func (m QueryResult) Prepaid() uint32 { return m.prepaid }

func (m QueryResult) Operation() string { return CashQueryResultWriter }

func (m QueryResult) String() string {
	return fmt.Sprintf("credit [%d] points [%d] prepaid [%d]", m.credit, m.points, m.prepaid)
}

func (m QueryResult) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.credit)
		w.WriteInt(m.points)
		if t.Region() == "GMS" && t.MajorVersion() > 12 {
			w.WriteInt(m.prepaid)
		}
		return w.Bytes()
	}
}

func (m *QueryResult) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		t := tenant.MustFromContext(ctx)
		m.credit = r.ReadUint32()
		m.points = r.ReadUint32()
		if t.Region() == "GMS" && t.MajorVersion() > 12 {
			m.prepaid = r.ReadUint32()
		}
	}
}
