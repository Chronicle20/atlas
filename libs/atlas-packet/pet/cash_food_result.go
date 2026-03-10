package pet

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const PetCashFoodResultWriter = "PetCashFoodResult"

type CashFoodResult struct {
	failure bool
	index   byte
}

func NewPetCashFoodResult(index byte) CashFoodResult {
	return CashFoodResult{failure: false, index: index}
}

func NewPetCashFoodResultError() CashFoodResult {
	return CashFoodResult{failure: true}
}

func (m CashFoodResult) Failure() bool   { return m.failure }
func (m CashFoodResult) Index() byte     { return m.index }
func (m CashFoodResult) Operation() string { return PetCashFoodResultWriter }
func (m CashFoodResult) String() string {
	return fmt.Sprintf("failure [%t], index [%d]", m.failure, m.index)
}

func (m CashFoodResult) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(m.failure)
		if !m.failure {
			w.WriteByte(m.index)
		}
		return w.Bytes()
	}
}

func (m *CashFoodResult) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.failure = r.ReadBool()
		if !m.failure {
			m.index = r.ReadByte()
		}
	}
}
