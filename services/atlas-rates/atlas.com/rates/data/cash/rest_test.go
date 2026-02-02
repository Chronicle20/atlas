package cash

import (
	"testing"
	"time"
)

func TestHasRateProperties(t *testing.T) {
	tests := []struct {
		name     string
		model    RestModel
		expected bool
	}{
		{
			name:     "nil spec",
			model:    RestModel{Spec: nil},
			expected: false,
		},
		{
			name:     "empty spec",
			model:    RestModel{Spec: make(map[SpecType]int32)},
			expected: false,
		},
		{
			name: "has rate",
			model: RestModel{Spec: map[SpecType]int32{
				SpecTypeRate: 2,
			}},
			expected: true,
		},
		{
			name: "has expR but not rate",
			model: RestModel{Spec: map[SpecType]int32{
				SpecTypeExpR: 2,
			}},
			expected: false,
		},
		{
			name: "has drpR but not rate",
			model: RestModel{Spec: map[SpecType]int32{
				SpecTypeDrpR: 2,
			}},
			expected: false,
		},
		{
			name: "has only time",
			model: RestModel{Spec: map[SpecType]int32{
				SpecTypeTime: 2147483647,
			}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.model.HasRateProperties(); got != tt.expected {
				t.Errorf("HasRateProperties() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetRate(t *testing.T) {
	tests := []struct {
		name     string
		model    RestModel
		expected int32
	}{
		{
			name:     "nil spec",
			model:    RestModel{Spec: nil},
			expected: 0,
		},
		{
			name:     "empty spec",
			model:    RestModel{Spec: make(map[SpecType]int32)},
			expected: 0,
		},
		{
			name: "has rate",
			model: RestModel{Spec: map[SpecType]int32{
				SpecTypeRate: 2,
			}},
			expected: 2,
		},
		{
			name: "has expR but not rate",
			model: RestModel{Spec: map[SpecType]int32{
				SpecTypeExpR: 3,
			}},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.model.GetRate(); got != tt.expected {
				t.Errorf("GetRate() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetExpR(t *testing.T) {
	tests := []struct {
		name     string
		model    RestModel
		expected int32
	}{
		{
			name:     "nil spec",
			model:    RestModel{Spec: nil},
			expected: 0,
		},
		{
			name:     "empty spec",
			model:    RestModel{Spec: make(map[SpecType]int32)},
			expected: 0,
		},
		{
			name: "has expR",
			model: RestModel{Spec: map[SpecType]int32{
				SpecTypeExpR: 3,
			}},
			expected: 3,
		},
		{
			name: "has drpR but not expR",
			model: RestModel{Spec: map[SpecType]int32{
				SpecTypeDrpR: 2,
			}},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.model.GetExpR(); got != tt.expected {
				t.Errorf("GetExpR() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetDrpR(t *testing.T) {
	tests := []struct {
		name     string
		model    RestModel
		expected int32
	}{
		{
			name:     "nil spec",
			model:    RestModel{Spec: nil},
			expected: 0,
		},
		{
			name:     "empty spec",
			model:    RestModel{Spec: make(map[SpecType]int32)},
			expected: 0,
		},
		{
			name: "has drpR",
			model: RestModel{Spec: map[SpecType]int32{
				SpecTypeDrpR: 2,
			}},
			expected: 2,
		},
		{
			name: "has expR but not drpR",
			model: RestModel{Spec: map[SpecType]int32{
				SpecTypeExpR: 3,
			}},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.model.GetDrpR(); got != tt.expected {
				t.Errorf("GetDrpR() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetTime(t *testing.T) {
	tests := []struct {
		name     string
		model    RestModel
		expected int32
	}{
		{
			name:     "nil spec",
			model:    RestModel{Spec: nil},
			expected: 0,
		},
		{
			name:     "empty spec",
			model:    RestModel{Spec: make(map[SpecType]int32)},
			expected: 0,
		},
		{
			name: "has time",
			model: RestModel{Spec: map[SpecType]int32{
				SpecTypeTime: 2147483647,
			}},
			expected: 2147483647,
		},
		{
			name: "has time 90 minutes",
			model: RestModel{Spec: map[SpecType]int32{
				SpecTypeTime: 90,
			}},
			expected: 90,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.model.GetTime(); got != tt.expected {
				t.Errorf("GetTime() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestHasTimeWindows(t *testing.T) {
	tests := []struct {
		name     string
		model    RestModel
		expected bool
	}{
		{
			name:     "nil time windows",
			model:    RestModel{TimeWindows: nil},
			expected: false,
		},
		{
			name:     "empty time windows",
			model:    RestModel{TimeWindows: []TimeWindow{}},
			expected: false,
		},
		{
			name: "has time windows",
			model: RestModel{TimeWindows: []TimeWindow{
				{Day: "MON", StartHour: 18, EndHour: 20},
			}},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.model.HasTimeWindows(); got != tt.expected {
				t.Errorf("HasTimeWindows() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsActiveAt(t *testing.T) {
	// Create a Monday at 19:00 (7 PM)
	mondayEvening := time.Date(2024, 1, 1, 19, 0, 0, 0, time.UTC) // Jan 1, 2024 was a Monday

	// Create a Monday at 10:00 (10 AM)
	mondayMorning := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	// Create a Tuesday at 19:00
	tuesdayEvening := time.Date(2024, 1, 2, 19, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		model     RestModel
		checkTime time.Time
		isHoliday bool
		expected  bool
	}{
		{
			name:      "no time windows - always active",
			model:     RestModel{TimeWindows: nil},
			checkTime: mondayEvening,
			isHoliday: false,
			expected:  true,
		},
		{
			name: "within time window",
			model: RestModel{TimeWindows: []TimeWindow{
				{Day: "MON", StartHour: 18, EndHour: 20},
			}},
			checkTime: mondayEvening,
			isHoliday: false,
			expected:  true,
		},
		{
			name: "outside time window - wrong hour",
			model: RestModel{TimeWindows: []TimeWindow{
				{Day: "MON", StartHour: 18, EndHour: 20},
			}},
			checkTime: mondayMorning,
			isHoliday: false,
			expected:  false,
		},
		{
			name: "outside time window - wrong day",
			model: RestModel{TimeWindows: []TimeWindow{
				{Day: "MON", StartHour: 18, EndHour: 20},
			}},
			checkTime: tuesdayEvening,
			isHoliday: false,
			expected:  false,
		},
		{
			name: "all day window (00-24)",
			model: RestModel{TimeWindows: []TimeWindow{
				{Day: "MON", StartHour: 0, EndHour: 24},
			}},
			checkTime: mondayMorning,
			isHoliday: false,
			expected:  true,
		},
		{
			name: "all day window - evening",
			model: RestModel{TimeWindows: []TimeWindow{
				{Day: "MON", StartHour: 0, EndHour: 24},
			}},
			checkTime: mondayEvening,
			isHoliday: false,
			expected:  true,
		},
		{
			name: "holiday window active on holiday",
			model: RestModel{TimeWindows: []TimeWindow{
				{Day: "HOL", StartHour: 0, EndHour: 24},
			}},
			checkTime: mondayMorning,
			isHoliday: true,
			expected:  true,
		},
		{
			name: "holiday window inactive on regular day",
			model: RestModel{TimeWindows: []TimeWindow{
				{Day: "HOL", StartHour: 0, EndHour: 24},
			}},
			checkTime: mondayMorning,
			isHoliday: false,
			expected:  false,
		},
		{
			name: "multiple windows - matches one",
			model: RestModel{TimeWindows: []TimeWindow{
				{Day: "MON", StartHour: 18, EndHour: 20},
				{Day: "TUE", StartHour: 18, EndHour: 20},
				{Day: "WED", StartHour: 18, EndHour: 20},
			}},
			checkTime: tuesdayEvening,
			isHoliday: false,
			expected:  true,
		},
		{
			name: "full week coverage",
			model: RestModel{TimeWindows: []TimeWindow{
				{Day: "MON", StartHour: 0, EndHour: 24},
				{Day: "TUE", StartHour: 0, EndHour: 24},
				{Day: "WED", StartHour: 0, EndHour: 24},
				{Day: "THU", StartHour: 0, EndHour: 24},
				{Day: "FRI", StartHour: 0, EndHour: 24},
				{Day: "SAT", StartHour: 0, EndHour: 24},
				{Day: "SUN", StartHour: 0, EndHour: 24},
			}},
			checkTime: tuesdayEvening,
			isHoliday: false,
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.model.IsActiveAt(tt.checkTime, tt.isHoliday); got != tt.expected {
				t.Errorf("IsActiveAt() = %v, want %v", got, tt.expected)
			}
		})
	}
}
