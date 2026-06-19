package clientbound

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/clientbound/StatusMessageDropPickUpItemUnavailable version=gms_v83 ida=0xa20ad9
// packet-audit:verify packet=character/clientbound/StatusMessageDropPickUpItemUnavailable version=gms_v84 ida=0xa6beef
// packet-audit:verify packet=character/clientbound/StatusMessageDropPickUpItemUnavailable version=gms_v87 ida=0xab818c
// packet-audit:verify packet=character/clientbound/StatusMessageDropPickUpItemUnavailable version=gms_v95 ida=0x9fe190
// packet-audit:verify packet=character/clientbound/StatusMessageDropPickUpItemUnavailable version=jms_v185 ida=0xb07a01
func TestStatusMessageDropPickUpItemUnavailable(t *testing.T) {
	input := NewStatusMessageDropPickUpItemUnavailable(0)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// packet-audit:verify packet=character/clientbound/StatusMessageDropPickUpInventoryFull version=gms_v83 ida=0xa20ad9
// packet-audit:verify packet=character/clientbound/StatusMessageDropPickUpInventoryFull version=gms_v84 ida=0xa6beef
// packet-audit:verify packet=character/clientbound/StatusMessageDropPickUpInventoryFull version=gms_v87 ida=0xab818c
// packet-audit:verify packet=character/clientbound/StatusMessageDropPickUpInventoryFull version=gms_v95 ida=0x9fe190
// packet-audit:verify packet=character/clientbound/StatusMessageDropPickUpInventoryFull version=jms_v185 ida=0xb07a01
func TestStatusMessageDropPickUpInventoryFull(t *testing.T) {
	input := NewStatusMessageDropPickUpInventoryFull(0)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// packet-audit:verify packet=character/clientbound/StatusMessageDropPickUpGameFileDamaged version=gms_v83 ida=0xa20ad9
// packet-audit:verify packet=character/clientbound/StatusMessageDropPickUpGameFileDamaged version=gms_v84 ida=0xa6beef
// packet-audit:verify packet=character/clientbound/StatusMessageDropPickUpGameFileDamaged version=gms_v87 ida=0xab818c
// packet-audit:verify packet=character/clientbound/StatusMessageDropPickUpGameFileDamaged version=gms_v95 ida=0x9fe190
// packet-audit:verify packet=character/clientbound/StatusMessageDropPickUpGameFileDamaged version=jms_v185 ida=0xb07a01
func TestStatusMessageDropPickUpGameFileDamaged(t *testing.T) {
	input := NewStatusMessageDropPickUpGameFileDamaged(0)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// packet-audit:verify packet=character/clientbound/StatusMessageDropPickUpStackableItem version=gms_v83 ida=0xa20ad9
// packet-audit:verify packet=character/clientbound/StatusMessageDropPickUpStackableItem version=gms_v84 ida=0xa6beef
// packet-audit:verify packet=character/clientbound/StatusMessageDropPickUpStackableItem version=gms_v87 ida=0xab818c
// packet-audit:verify packet=character/clientbound/StatusMessageDropPickUpStackableItem version=gms_v95 ida=0x9fe190
// packet-audit:verify packet=character/clientbound/StatusMessageDropPickUpStackableItem version=jms_v185 ida=0xb07a01
func TestStatusMessageDropPickUpStackableItem(t *testing.T) {
	input := NewStatusMessageDropPickUpStackableItem(0, 2000000, 5)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// packet-audit:verify packet=character/clientbound/StatusMessageDropPickUpUnStackableItem version=gms_v83 ida=0xa20ad9
// packet-audit:verify packet=character/clientbound/StatusMessageDropPickUpUnStackableItem version=gms_v84 ida=0xa6beef
// packet-audit:verify packet=character/clientbound/StatusMessageDropPickUpUnStackableItem version=gms_v87 ida=0xab818c
// packet-audit:verify packet=character/clientbound/StatusMessageDropPickUpUnStackableItem version=gms_v95 ida=0x9fe190
// packet-audit:verify packet=character/clientbound/StatusMessageDropPickUpUnStackableItem version=jms_v185 ida=0xb07a01
func TestStatusMessageDropPickUpUnStackableItem(t *testing.T) {
	input := NewStatusMessageDropPickUpUnStackableItem(0, 1302000)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// packet-audit:verify packet=character/clientbound/StatusMessageDropLossStackableItem version=gms_v83 ida=0xa20ad9
// packet-audit:verify packet=character/clientbound/StatusMessageDropLossStackableItem version=gms_v84 ida=0xa6beef
// packet-audit:verify packet=character/clientbound/StatusMessageDropLossStackableItem version=gms_v87 ida=0xab818c
// packet-audit:verify packet=character/clientbound/StatusMessageDropLossStackableItem version=gms_v95 ida=0x9fe190
// packet-audit:verify packet=character/clientbound/StatusMessageDropLossStackableItem version=jms_v185 ida=0xb07a01
func TestStatusMessageDropLossStackableItem(t *testing.T) {
	input := NewStatusMessageDropLossStackableItem(0, 2000000, 5)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// packet-audit:verify packet=character/clientbound/StatusMessageDropLossUnStackableItem version=gms_v83 ida=0xa20ad9
// packet-audit:verify packet=character/clientbound/StatusMessageDropLossUnStackableItem version=gms_v84 ida=0xa6beef
// packet-audit:verify packet=character/clientbound/StatusMessageDropLossUnStackableItem version=gms_v87 ida=0xab818c
// packet-audit:verify packet=character/clientbound/StatusMessageDropLossUnStackableItem version=gms_v95 ida=0x9fe190
// packet-audit:verify packet=character/clientbound/StatusMessageDropLossUnStackableItem version=jms_v185 ida=0xb07a01
func TestStatusMessageDropLossUnStackableItem(t *testing.T) {
	input := NewStatusMessageDropLossUnStackableItem(0, 1302000)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// packet-audit:verify packet=character/clientbound/StatusMessageDropPickUpMeso version=gms_v83 ida=0xa20ad9
// packet-audit:verify packet=character/clientbound/StatusMessageDropPickUpMeso version=gms_v84 ida=0xa6beef
// packet-audit:verify packet=character/clientbound/StatusMessageDropPickUpMeso version=gms_v87 ida=0xab818c
// packet-audit:verify packet=character/clientbound/StatusMessageDropPickUpMeso version=gms_v95 ida=0x9fe190
// packet-audit:verify packet=character/clientbound/StatusMessageDropPickUpMeso version=jms_v185 ida=0xb07a01
func TestStatusMessageDropPickUpMeso(t *testing.T) {
	input := NewStatusMessageDropPickUpMeso(0, true, 1000, 0)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// packet-audit:verify packet=character/clientbound/StatusMessageForfeitQuestRecord version=gms_v83 ida=0xa20f4c
// packet-audit:verify packet=character/clientbound/StatusMessageForfeitQuestRecord version=gms_v84 ida=0xa6c362
// packet-audit:verify packet=character/clientbound/StatusMessageForfeitQuestRecord version=gms_v87 ida=0xab85d2
// packet-audit:verify packet=character/clientbound/StatusMessageForfeitQuestRecord version=gms_v95 ida=0xa03920
// packet-audit:verify packet=character/clientbound/StatusMessageForfeitQuestRecord version=jms_v185 ida=0xb07e49
func TestStatusMessageForfeitQuestRecord(t *testing.T) {
	input := NewStatusMessageForfeitQuestRecord(1, 1000)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// packet-audit:verify packet=character/clientbound/StatusMessageUpdateQuestRecord version=gms_v83 ida=0xa20f4c
// packet-audit:verify packet=character/clientbound/StatusMessageUpdateQuestRecord version=gms_v84 ida=0xa6c362
// packet-audit:verify packet=character/clientbound/StatusMessageUpdateQuestRecord version=gms_v87 ida=0xab85d2
// packet-audit:verify packet=character/clientbound/StatusMessageUpdateQuestRecord version=gms_v95 ida=0xa03920
// packet-audit:verify packet=character/clientbound/StatusMessageUpdateQuestRecord version=jms_v185 ida=0xb07e49
func TestStatusMessageUpdateQuestRecord(t *testing.T) {
	input := NewStatusMessageUpdateQuestRecord(1, 1000, "001")
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// packet-audit:verify packet=character/clientbound/StatusMessageCompleteQuestRecord version=gms_v83 ida=0xa20f4c
// packet-audit:verify packet=character/clientbound/StatusMessageCompleteQuestRecord version=gms_v84 ida=0xa6c362
// packet-audit:verify packet=character/clientbound/StatusMessageCompleteQuestRecord version=gms_v87 ida=0xab85d2
// packet-audit:verify packet=character/clientbound/StatusMessageCompleteQuestRecord version=gms_v95 ida=0xa03920
// packet-audit:verify packet=character/clientbound/StatusMessageCompleteQuestRecord version=jms_v185 ida=0xb07e49
func TestStatusMessageCompleteQuestRecord(t *testing.T) {
	input := NewStatusMessageCompleteQuestRecord(1, 1000, time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC))
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// packet-audit:verify packet=character/clientbound/StatusMessageCashItemExpire version=gms_v83 ida=0xa216fc
// packet-audit:verify packet=character/clientbound/StatusMessageCashItemExpire version=gms_v84 ida=0xa6cb31
// packet-audit:verify packet=character/clientbound/StatusMessageCashItemExpire version=gms_v87 ida=0xab8d8e
// packet-audit:verify packet=character/clientbound/StatusMessageCashItemExpire version=gms_v95 ida=0x9f8060
// packet-audit:verify packet=character/clientbound/StatusMessageCashItemExpire version=jms_v185 ida=0xb085df
func TestStatusMessageCashItemExpire(t *testing.T) {
	input := NewStatusMessageCashItemExpire(2, 5000000)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// packet-audit:verify packet=character/clientbound/StatusMessageIncreaseExperience version=gms_v83 ida=0xa21ac5
// packet-audit:verify packet=character/clientbound/StatusMessageIncreaseExperience version=gms_v84 ida=0xa6cfd7
// packet-audit:verify packet=character/clientbound/StatusMessageIncreaseExperience version=gms_v87 ida=0xab9234
// packet-audit:verify packet=character/clientbound/StatusMessageIncreaseExperience version=gms_v95 ida=0x9f86c0
// packet-audit:verify packet=character/clientbound/StatusMessageIncreaseExperience version=jms_v185 ida=0xb08a97
func TestStatusMessageIncreaseExperience(t *testing.T) {
	input := NewStatusMessageIncreaseExperience(3, true, 500, true, 10, 5, 0, 0, 2, 3, 1, 0, 0, 50, 0, 0, 100, 200)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// packet-audit:verify packet=character/clientbound/StatusMessageIncreaseSkillPoint version=gms_v84 ida=0xa6cefa
// packet-audit:verify packet=character/clientbound/StatusMessageIncreaseSkillPoint version=gms_v87 ida=0xab9157
// packet-audit:verify packet=character/clientbound/StatusMessageIncreaseSkillPoint version=gms_v95 ida=0x9f8570
// packet-audit:verify packet=character/clientbound/StatusMessageIncreaseSkillPoint version=jms_v185 ida=0xb089ab
func TestStatusMessageIncreaseSkillPoint(t *testing.T) {
	input := NewStatusMessageIncreaseSkillPoint(4, 100, 3)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// packet-audit:verify packet=character/clientbound/StatusMessageIncreaseFame version=gms_v83 ida=0xa2212d
// packet-audit:verify packet=character/clientbound/StatusMessageIncreaseFame version=gms_v84 ida=0xa6d63f
// packet-audit:verify packet=character/clientbound/StatusMessageIncreaseFame version=gms_v87 ida=0xab9975
// packet-audit:verify packet=character/clientbound/StatusMessageIncreaseFame version=gms_v95 ida=0x9f90a0
// packet-audit:verify packet=character/clientbound/StatusMessageIncreaseFame version=jms_v185 ida=0xb09180
func TestStatusMessageIncreaseFame(t *testing.T) {
	input := NewStatusMessageIncreaseFame(5, 1)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// packet-audit:verify packet=character/clientbound/StatusMessageIncreaseMeso version=gms_v83 ida=0xa221f3
// packet-audit:verify packet=character/clientbound/StatusMessageIncreaseMeso version=gms_v84 ida=0xa6d705
// packet-audit:verify packet=character/clientbound/StatusMessageIncreaseMeso version=gms_v87 ida=0xab9a3b
// packet-audit:verify packet=character/clientbound/StatusMessageIncreaseMeso version=gms_v95 ida=0x9fe910
// packet-audit:verify packet=character/clientbound/StatusMessageIncreaseMeso version=jms_v185 ida=0xb09246
func TestStatusMessageIncreaseMeso(t *testing.T) {
	input := NewStatusMessageIncreaseMeso(6, 5000)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// packet-audit:verify packet=character/clientbound/StatusMessageIncreaseGuildPoint version=gms_v83 ida=0xa222c9
// packet-audit:verify packet=character/clientbound/StatusMessageIncreaseGuildPoint version=gms_v84 ida=0xa6d7db
// packet-audit:verify packet=character/clientbound/StatusMessageIncreaseGuildPoint version=gms_v87 ida=0xab9b11
// packet-audit:verify packet=character/clientbound/StatusMessageIncreaseGuildPoint version=gms_v95 ida=0x9f91e0
// packet-audit:verify packet=character/clientbound/StatusMessageIncreaseGuildPoint version=jms_v185 ida=0xb09397
func TestStatusMessageIncreaseGuildPoint(t *testing.T) {
	input := NewStatusMessageIncreaseGuildPoint(7, 100)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// packet-audit:verify packet=character/clientbound/StatusMessageGiveBuff version=gms_v83 ida=0xa2238f
// packet-audit:verify packet=character/clientbound/StatusMessageGiveBuff version=gms_v84 ida=0xa6d8a1
// packet-audit:verify packet=character/clientbound/StatusMessageGiveBuff version=gms_v87 ida=0xab9bd7
// packet-audit:verify packet=character/clientbound/StatusMessageGiveBuff version=gms_v95 ida=0x9f2df0
// packet-audit:verify packet=character/clientbound/StatusMessageGiveBuff version=jms_v185 ida=0xb0945d
func TestStatusMessageGiveBuff(t *testing.T) {
	input := NewStatusMessageGiveBuff(8, 2022003)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// packet-audit:verify packet=character/clientbound/StatusMessageGeneralItemExpire version=gms_v83 ida=0xa217a2
// packet-audit:verify packet=character/clientbound/StatusMessageGeneralItemExpire version=gms_v84 ida=0xa6cbd7
// packet-audit:verify packet=character/clientbound/StatusMessageGeneralItemExpire version=gms_v87 ida=0xab8e34
// packet-audit:verify packet=character/clientbound/StatusMessageGeneralItemExpire version=gms_v95 ida=0x9f8180
// packet-audit:verify packet=character/clientbound/StatusMessageGeneralItemExpire version=jms_v185 ida=0xb08686
func TestStatusMessageGeneralItemExpire(t *testing.T) {
	input := NewStatusMessageGeneralItemExpire(9, []uint32{2000000, 2000001})
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// packet-audit:verify packet=character/clientbound/StatusMessageSystemMessage version=gms_v83 ida=0xa21a78
// packet-audit:verify packet=character/clientbound/StatusMessageSystemMessage version=gms_v84 ida=0xa6cead
// packet-audit:verify packet=character/clientbound/StatusMessageSystemMessage version=gms_v87 ida=0xab910a
// packet-audit:verify packet=character/clientbound/StatusMessageSystemMessage version=gms_v95 ida=0x9fe860
// packet-audit:verify packet=character/clientbound/StatusMessageSystemMessage version=jms_v185 ida=0xb0895e
func TestStatusMessageSystemMessage(t *testing.T) {
	input := NewStatusMessageSystemMessage(10, "Hello World")
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// packet-audit:verify packet=character/clientbound/StatusMessageQuestRecordEx version=gms_v83 ida=0xa2160b
// packet-audit:verify packet=character/clientbound/StatusMessageQuestRecordEx version=gms_v84 ida=0xa6ca40
// packet-audit:verify packet=character/clientbound/StatusMessageQuestRecordEx version=gms_v87 ida=0xab8c9d
// packet-audit:verify packet=character/clientbound/StatusMessageQuestRecordEx version=gms_v95 ida=0x9fe6a0
// packet-audit:verify packet=character/clientbound/StatusMessageQuestRecordEx version=jms_v185 ida=0xb084ee
func TestStatusMessageQuestRecordEx(t *testing.T) {
	input := NewStatusMessageQuestRecordEx(11, 2000, "some_info")
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// packet-audit:verify packet=character/clientbound/StatusMessageItemProtectExpire version=gms_v83 ida=0xa2187e
// packet-audit:verify packet=character/clientbound/StatusMessageItemProtectExpire version=gms_v84 ida=0xa6ccb3
// packet-audit:verify packet=character/clientbound/StatusMessageItemProtectExpire version=gms_v87 ida=0xab8f10
// packet-audit:verify packet=character/clientbound/StatusMessageItemProtectExpire version=gms_v95 ida=0x9f82e0
// packet-audit:verify packet=character/clientbound/StatusMessageItemProtectExpire version=jms_v185 ida=0xb08763
func TestStatusMessageItemProtectExpire(t *testing.T) {
	input := NewStatusMessageItemProtectExpire(12, []uint32{1302000, 1302001})
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// packet-audit:verify packet=character/clientbound/StatusMessageItemExpireReplace version=gms_v83 ida=0xa2195a
// packet-audit:verify packet=character/clientbound/StatusMessageItemExpireReplace version=gms_v84 ida=0xa6cd8f
// packet-audit:verify packet=character/clientbound/StatusMessageItemExpireReplace version=gms_v87 ida=0xab8fec
// packet-audit:verify packet=character/clientbound/StatusMessageItemExpireReplace version=gms_v95 ida=0x9fe7a0
// packet-audit:verify packet=character/clientbound/StatusMessageItemExpireReplace version=jms_v185 ida=0xb08840
func TestStatusMessageItemExpireReplace(t *testing.T) {
	input := NewStatusMessageItemExpireReplace(13, []string{"Item A expired", "Item B expired"})
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// packet-audit:verify packet=character/clientbound/StatusMessageSkillExpire version=gms_v83 ida=0xa219be
// packet-audit:verify packet=character/clientbound/StatusMessageSkillExpire version=gms_v84 ida=0xa6cdf3
// packet-audit:verify packet=character/clientbound/StatusMessageSkillExpire version=gms_v87 ida=0xab9050
// packet-audit:verify packet=character/clientbound/StatusMessageSkillExpire version=gms_v95 ida=0x9f8440
// packet-audit:verify packet=character/clientbound/StatusMessageSkillExpire version=jms_v185 ida=0xb088a4
func TestStatusMessageSkillExpire(t *testing.T) {
	input := NewStatusMessageSkillExpire(14, []uint32{1001003, 1001004})
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// packet-audit:verify packet=character/clientbound/StatusMessageJMSCounterNotice version=jms_v185 ida=0xb0931c
func TestStatusMessageJMSCounterNotice(t *testing.T) {
	input := NewStatusMessageJMSCounterNotice(15, 1)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
