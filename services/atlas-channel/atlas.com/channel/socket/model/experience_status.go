package model

type IncreaseExperienceConfig struct {
	White                   bool  // White or Yellow text for displaying Amount.
	Amount                  int32 // Right side. You have gained experience.
	InChat                  bool  // If true. Amount is written. You have gained experience (%d) in chat.
	MonsterBookBonus        int32 // Right side. Yellow. Bonus Event EXP
	MobEventBonusPercentage byte  // In chat. Pink. A bonus EXP %d% is awarded for every 3rd monster defeated.
	PlayTimeHour            byte  // Right side. Yellow. Bonus EXP for hunting over (%d) hrs. (+MobEventBonusPercentage)
	PartyBonusPercentage    byte  // Not used in v83
	WeddingBonusEXP         int32 // Right side. Yellow. Bonus Wedding EXP
	QuestBonusRate          byte  // Earned 'Spirit Week Event' bonus EXP (%d)
	QuestBonusRemainCount   byte  // The next %d completed quests will include additional Event Bonus EXP
	PartyBonusEventRate     byte  // Dictates if right side of PartyBonusExp displays a (x0)
	PartyBonusExp           int32 // Right side. Yellow. Bonus Event Party EXP (x0)
	ItemBonusEXP            int32 // Right side. Yellow. Equip Item Bonus EXP
	PremiumIPExp            int32 // Right side. Yellow. Internet Cafe EXP Bonus
	RainbowWeekEventEXP     int32 // Right side. Yellow. Rainbow Week Bonus Event EXP
	PartyEXPRingEXP         int32 // Available v95+
	CakePieEventBonus       int32 // Available v95+
}
