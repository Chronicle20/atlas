package bosshp

import (
	monsterdata "atlas-channel/data/monster"
	"atlas-channel/data/monster/mock"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	fieldpkt "github.com/Chronicle20/atlas/libs/atlas-packet/field"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name               string
		boss               bool
		tagColor           byte
		tagBackgroundColor byte
		lookupErr          error
		expectOk           bool
		expectErr          bool
		expectGauge        Gauge
	}{
		{
			name:               "qualifying",
			boss:               true,
			tagColor:           6,
			tagBackgroundColor: 1,
			expectOk:           true,
			expectGauge: Gauge{
				monsterId:          8800002,
				currentHp:          50000,
				maxHp:              100000,
				tagColor:           6,
				tagBackgroundColor: 1,
			},
		},
		{
			name:               "boss, zero tag colour (FR-2)",
			boss:               true,
			tagColor:           0,
			tagBackgroundColor: 1,
			expectOk:           false,
			expectGauge:        Gauge{},
		},
		{
			name:        "non-boss, non-zero tag colour (FR-2)",
			boss:        false,
			tagColor:    6,
			expectOk:    false,
			expectGauge: Gauge{},
		},
		{
			name:               "zero background colour still qualifies (FR-3)",
			boss:               true,
			tagColor:           6,
			tagBackgroundColor: 0,
			expectOk:           true,
			expectGauge: Gauge{
				monsterId:          8800002,
				currentHp:          50000,
				maxHp:              100000,
				tagColor:           6,
				tagBackgroundColor: 0,
			},
		},
		{
			name:        "lookup failure (FR-17)",
			lookupErr:   errors.New("boom"),
			expectOk:    false,
			expectErr:   true,
			expectGauge: Gauge{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &mock.ProcessorMock{
				GetByIdFunc: func(monsterId uint32) (monsterdata.Model, error) {
					if tt.lookupErr != nil {
						return monsterdata.Model{}, tt.lookupErr
					}
					return monsterdata.Extract(monsterdata.RestModel{
						Id:                 monsterId,
						Boss:               tt.boss,
						TagColor:           tt.tagColor,
						TagBackgroundColor: tt.tagBackgroundColor,
					})
				},
			}

			r := NewResolverFrom(p)
			g, ok, err := r.Resolve(8800002, 50000, 100000)

			if ok != tt.expectOk {
				t.Errorf("ok = %v, want %v", ok, tt.expectOk)
			}
			if tt.expectErr {
				if err == nil {
					t.Fatalf("err = nil, want non-nil")
				}
				if !strings.Contains(err.Error(), "boom") {
					t.Errorf("err = %q, want to contain %q", err.Error(), "boom")
				}
			} else if err != nil {
				t.Errorf("err = %v, want nil", err)
			}

			if g.MonsterId() != tt.expectGauge.monsterId {
				t.Errorf("MonsterId() = %d, want %d", g.MonsterId(), tt.expectGauge.monsterId)
			}
			if g.CurrentHp() != tt.expectGauge.currentHp {
				t.Errorf("CurrentHp() = %d, want %d", g.CurrentHp(), tt.expectGauge.currentHp)
			}
			if g.MaxHp() != tt.expectGauge.maxHp {
				t.Errorf("MaxHp() = %d, want %d", g.MaxHp(), tt.expectGauge.maxHp)
			}
			if g.TagColor() != tt.expectGauge.tagColor {
				t.Errorf("TagColor() = %d, want %d", g.TagColor(), tt.expectGauge.tagColor)
			}
			if g.TagBackgroundColor() != tt.expectGauge.tagBackgroundColor {
				t.Errorf("TagBackgroundColor() = %d, want %d", g.TagBackgroundColor(), tt.expectGauge.tagBackgroundColor)
			}
		})
	}
}

func TestBossHpBodyBytes(t *testing.T) {
	t.Run("mode resolved from operations table", func(t *testing.T) {
		options := map[string]interface{}{
			"operations": map[string]interface{}{"BOSS_HP": float64(5)},
		}

		b := fieldpkt.FieldEffectBossHpBody(8800002, 50000, 100000, 6, 1)(logrus.New(), context.Background())(options)

		expected := []byte{0x05, 0x02, 0x47, 0x86, 0x00, 0x50, 0xc3, 0x00, 0x00, 0xa0, 0x86, 0x01, 0x00, 0x06, 0x01}
		if len(b) != len(expected) {
			t.Fatalf("len(b) = %d, want %d (b = % x)", len(b), len(expected), b)
		}
		for i := range expected {
			if b[i] != expected[i] {
				t.Errorf("b[%d] = 0x%02x, want 0x%02x", i, b[i], expected[i])
			}
		}
	})

	t.Run("unmapped mode falls back to sentinel", func(t *testing.T) {
		options := map[string]interface{}{
			"operations": map[string]interface{}{},
		}

		b := fieldpkt.FieldEffectBossHpBody(8800002, 50000, 100000, 6, 1)(logrus.New(), context.Background())(options)

		if len(b) == 0 {
			t.Fatalf("len(b) = 0, want at least 1 byte")
		}
		if b[0] != 99 {
			t.Errorf("b[0] = %d, want 99 (atlas_packet.ResolveCode sentinel)", b[0])
		}
	})
}
