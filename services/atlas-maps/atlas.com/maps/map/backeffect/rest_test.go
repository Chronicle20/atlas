package backeffect

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	beconst "github.com/Chronicle20/atlas/libs/atlas-constants/backeffect"
)

// TestTransformSlice pins TransformSlice's contract directly at the function
// level: an empty input collection must marshal as `[]`, never `null`, which
// requires the returned slice to be non-nil even when zero-length. A
// non-empty input must transform every element in order.
func TestTransformSlice(t *testing.T) {
	tests := []struct {
		name    string
		input   []BackEffectEntry
		wantLen int
	}{
		{
			name:    "empty input returns non-nil, zero-length slice",
			input:   []BackEffectEntry{},
			wantLen: 0,
		},
		{
			name: "non-empty input transforms every element in order",
			input: []BackEffectEntry{
				{Effect: beconst.EffectShow, FieldId: 100000000, PageId: 1, Duration: 1000},
				{Effect: beconst.EffectHide, FieldId: 100000000, PageId: 2, Duration: 0},
			},
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TransformSlice(tt.input)
			require.NoError(t, err)

			require.NotNil(t, got)
			assert.Len(t, got, tt.wantLen)

			for i, e := range tt.input {
				assert.Equal(t, e.Effect, got[i].Effect)
				assert.Equal(t, e.FieldId, got[i].FieldId)
				assert.EqualValues(t, e.PageId, got[i].PageId)
				assert.Equal(t, e.Duration, got[i].Duration)
			}
		})
	}
}
