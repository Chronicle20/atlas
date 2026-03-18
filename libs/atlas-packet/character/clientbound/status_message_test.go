package clientbound

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas-packet/test"
)

func TestStatusMessageDropPickUpItemUnavailable(t *testing.T) {
	input := NewStatusMessageDropPickUpItemUnavailable(0)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestStatusMessageDropPickUpInventoryFull(t *testing.T) {
	input := NewStatusMessageDropPickUpInventoryFull(0)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestStatusMessageDropPickUpStackableItem(t *testing.T) {
	input := NewStatusMessageDropPickUpStackableItem(0, 2000000, 5)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestStatusMessageDropPickUpUnStackableItem(t *testing.T) {
	input := NewStatusMessageDropPickUpUnStackableItem(0, 1302000)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestStatusMessageDropPickUpMeso(t *testing.T) {
	input := NewStatusMessageDropPickUpMeso(0, true, 1000, 0)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestStatusMessageForfeitQuestRecord(t *testing.T) {
	input := NewStatusMessageForfeitQuestRecord(1, 1000)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestStatusMessageUpdateQuestRecord(t *testing.T) {
	input := NewStatusMessageUpdateQuestRecord(1, 1000, "001")
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestStatusMessageCompleteQuestRecord(t *testing.T) {
	input := NewStatusMessageCompleteQuestRecord(1, 1000, time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC))
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestStatusMessageCashItemExpire(t *testing.T) {
	input := NewStatusMessageCashItemExpire(2, 5000000)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestStatusMessageIncreaseExperience(t *testing.T) {
	input := NewStatusMessageIncreaseExperience(3, true, 500, true, 10, 5, 0, 0, 2, 3, 1, 0, 0, 50, 0, 0, 100, 200)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestStatusMessageIncreaseSkillPoint(t *testing.T) {
	input := NewStatusMessageIncreaseSkillPoint(4, 100, 3)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestStatusMessageIncreaseFame(t *testing.T) {
	input := NewStatusMessageIncreaseFame(5, 1)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestStatusMessageIncreaseMeso(t *testing.T) {
	input := NewStatusMessageIncreaseMeso(6, 5000)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestStatusMessageIncreaseGuildPoint(t *testing.T) {
	input := NewStatusMessageIncreaseGuildPoint(7, 100)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestStatusMessageGiveBuff(t *testing.T) {
	input := NewStatusMessageGiveBuff(8, 2022003)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestStatusMessageGeneralItemExpire(t *testing.T) {
	input := NewStatusMessageGeneralItemExpire(9, []uint32{2000000, 2000001})
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestStatusMessageSystemMessage(t *testing.T) {
	input := NewStatusMessageSystemMessage(10, "Hello World")
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestStatusMessageQuestRecordEx(t *testing.T) {
	input := NewStatusMessageQuestRecordEx(11, 2000, "some_info")
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestStatusMessageItemProtectExpire(t *testing.T) {
	input := NewStatusMessageItemProtectExpire(12, []uint32{1302000, 1302001})
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestStatusMessageItemExpireReplace(t *testing.T) {
	input := NewStatusMessageItemExpireReplace(13, []string{"Item A expired", "Item B expired"})
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestStatusMessageSkillExpire(t *testing.T) {
	input := NewStatusMessageSkillExpire(14, []uint32{1001003, 1001004})
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
