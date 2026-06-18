package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CoconutScoreWriter = "CoconutScore"

// packet-audit:fname CField_Coconut::OnCoconutScore
type CoconutScore struct {
	mapleScore uint16
	storyScore uint16
}

func NewCoconutScore(mapleScore uint16, storyScore uint16) CoconutScore {
	return CoconutScore{mapleScore: mapleScore, storyScore: storyScore}
}

func (m CoconutScore) MapleScore() uint16 { return m.mapleScore }
func (m CoconutScore) StoryScore() uint16 { return m.storyScore }

func (m CoconutScore) Operation() string { return CoconutScoreWriter }
func (m CoconutScore) String() string {
	return fmt.Sprintf("mapleScore [%d] storyScore [%d]", m.mapleScore, m.storyScore)
}

func (m CoconutScore) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteShort(m.mapleScore)
		w.WriteShort(m.storyScore)
		return w.Bytes()
	}
}

func (m *CoconutScore) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mapleScore = r.ReadUint16()
		m.storyScore = r.ReadUint16()
	}
}
