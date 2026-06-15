package serverbound

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const LeftKnockbackHandle = "LeftKnockback"

// LeftKnockback - CField_SnowBall::Update
// Sent when the snowball crosses the knockback boundary. Empty body (header only).
type LeftKnockback struct {
}

func NewLeftKnockback() LeftKnockback {
	return LeftKnockback{}
}

func (m LeftKnockback) Operation() string {
	return LeftKnockbackHandle
}

func (m LeftKnockback) String() string {
	return "empty"
}

func (m LeftKnockback) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		return w.Bytes()
	}
}

func (m *LeftKnockback) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
	}
}
