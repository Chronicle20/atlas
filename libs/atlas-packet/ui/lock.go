package ui

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const UiLockWriter = "UiLock"

type Lock struct {
	enable                      bool
	tAfterLeaveDirectionMode    int32
}

func NewUiLock(enable bool, tAfterLeaveDirectionMode int32) Lock {
	return Lock{enable: enable, tAfterLeaveDirectionMode: tAfterLeaveDirectionMode}
}

func (m Lock) Enable() bool     { return m.enable }
func (m Lock) Operation() string { return UiLockWriter }
func (m Lock) String() string    { return fmt.Sprintf("enable [%t]", m.enable) }

func (m Lock) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(m.enable)
		if t.Region() == "GMS" && t.MajorVersion() >= 90 {
			w.WriteInt32(m.tAfterLeaveDirectionMode)
		}
		return w.Bytes()
	}
}

func (m *Lock) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.enable = r.ReadBool()
		if t.Region() == "GMS" && t.MajorVersion() >= 90 {
			m.tAfterLeaveDirectionMode = r.ReadInt32()
		}
	}
}
