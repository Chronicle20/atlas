package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const FieldObstacleOnOffListWriter = "FieldObstacleOnOffList"

type ObstacleState struct {
	name  string
	state uint32
}

func NewObstacleState(name string, state uint32) ObstacleState {
	return ObstacleState{name: name, state: state}
}

func (o ObstacleState) Name() string  { return o.name }
func (o ObstacleState) State() uint32 { return o.state }

// packet-audit:fname CField::OnFieldObstacleOnOffStatus
type FieldObstacleOnOffList struct {
	obstacles []ObstacleState
}

func NewFieldObstacleOnOffList(obstacles []ObstacleState) FieldObstacleOnOffList {
	return FieldObstacleOnOffList{obstacles: obstacles}
}

func (m FieldObstacleOnOffList) Obstacles() []ObstacleState { return m.obstacles }

func (m FieldObstacleOnOffList) Operation() string { return FieldObstacleOnOffListWriter }
func (m FieldObstacleOnOffList) String() string {
	return fmt.Sprintf("obstacles [%d]", len(m.obstacles))
}

func (m FieldObstacleOnOffList) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(uint32(len(m.obstacles)))
		for _, o := range m.obstacles {
			w.WriteAsciiString(o.name)
			w.WriteInt(o.state)
		}
		return w.Bytes()
	}
}

func (m *FieldObstacleOnOffList) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		count := r.ReadUint32()
		m.obstacles = make([]ObstacleState, 0, count)
		for i := uint32(0); i < count; i++ {
			name := r.ReadAsciiString()
			state := r.ReadUint32()
			m.obstacles = append(m.obstacles, ObstacleState{name: name, state: state})
		}
	}
}
