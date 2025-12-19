package _map

import (
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
)

type Model struct {
	worldId   world.Id
	channelId channel.Id
	mapId     Id
}

func NewModel(worldId world.Id) func(channelId channel.Id) func(mapId Id) Model {
	return func(channelId channel.Id) func(mapId Id) Model {
		return func(mapId Id) Model {
			return Model{
				worldId:   worldId,
				channelId: channelId,
				mapId:     mapId,
			}
		}
	}
}

func (m Model) WorldId() world.Id {
	return m.worldId
}

func (m Model) ChannelId() channel.Id {
	return m.channelId
}

func (m Model) MapId() Id {
	return m.mapId
}

func GenerateSequentialDungeonIds(baseId Id, size int) []Id {
	ids := make([]Id, 0)
	for i := range size {
		ids = append(ids, Id(int(baseId)+i))
	}
	return ids
}

func GenerateEveryTwoDungeonIds(baseId Id, size int) []Id {
	ids := make([]Id, 0)
	for i := range size {
		ids = append(ids, Id(int(baseId)+i*2))
	}
	return ids
}

func MiniDungeonHenesysPigFarmIds() []Id {
	return GenerateSequentialDungeonIds(MiniDungeonHenesysPigFarmBaseId, 30)
}

func MiniDungeonGolemsCastleRuinsIds() []Id {
	return GenerateSequentialDungeonIds(MiniDungeonGolemsCastleRuinsBaseId, 40)
}

func MiniDungeonCaveOfMushroomsIds() []Id {
	return GenerateSequentialDungeonIds(MiniDungeonCaveOfMushroomsBaseId, 30)
}

func AransPastHeadBlacksmithsShopIds() []Id {
	return GenerateSequentialDungeonIds(AransPastHeadBlacksmithsShopBaseId, 10)
}

func AransPastOutsideBlacksmithsShopIds() []Id {
	return GenerateSequentialDungeonIds(AransPastOutsideBlacksmithsShopBaseId, 10)
}

func HiddenStreetTheHillOfChristmasIds() []Id {
	return GenerateSequentialDungeonIds(HiddenStreetTheHillOfChristmasBaseId, 15)
}

func MiniDungeonDrummerBunnysLairIds() []Id {
	return GenerateSequentialDungeonIds(MiniDungeonDrummerBunnysLairBaseId, 25)
}

func MiniDungeonTheRoundTableOfKentaurusIds() []Id {
	return GenerateSequentialDungeonIds(MiniDungeonTheRoundTableOfKentaurusBaseId, 30)
}

func MiniDungeonTheRestoringMemoryIds() []Id {
	return GenerateSequentialDungeonIds(MiniDungeonTheRestoringMemoryBaseId, 30)
}

func MiniDungeonNewtSecuredZoneIds() []Id {
	return GenerateSequentialDungeonIds(MiniDungeonNewtSecuredZoneBaseId, 30)
}

func MiniDungeonHillOfSandstormsIds() []Id {
	return GenerateSequentialDungeonIds(MiniDungeonHillOfSandstormsBaseId, 35)
}

func HiddenStreetCriticalErrorIds() []Id {
	return GenerateSequentialDungeonIds(HiddenStreetCriticalErrorBaseId, 40)
}

func MalaysiaLongestRideOnByebyeStationIds() []Id {
	return GenerateSequentialDungeonIds(MalaysiaLongestRideOnByebyeStationBaseId, 20)
}

func MiniDungeonSweetCakeHill1Ids() []Id {
	return GenerateSequentialDungeonIds(MiniDungeonSweetCakeHill1BaseId, 10)
}

func MiniDungeonSweetCakeHill2Ids() []Id {
	return GenerateSequentialDungeonIds(MiniDungeonSweetCakeHill2BaseId, 10)
}

func MiniDungeonSweetCakeHill3Ids() []Id {
	return GenerateSequentialDungeonIds(MiniDungeonSweetCakeHill3BaseId, 10)
}

func MuLungDojoSoGongsRoomIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoSoGongsRoomBaseId, 5)
}

func MuLungDojoMuLungDojo1stFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo1stFloorBaseId, 15)
}

func MuLungDojoMuLungDojo2ndFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo2ndFloorBaseId, 15)
}

func MuLungDojoMuLungDojo3rdFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo3rdFloorBaseId, 15)
}

func MuLungDojoMuLungDojo4thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo4thFloorBaseId, 15)
}

func MuLungDojoMuLungDojo5thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo5thFloorBaseId, 15)
}

func MuLungDojoMuLungDojo6thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo6thFloorBaseId, 15)
}

func MuLungDojoMuLungDojo7thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo7thFloorBaseId, 15)
}

func MuLungDojoMuLungDojo8thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo8thFloorBaseId, 15)
}

func MuLungDojoMuLungDojo9thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo9thFloorBaseId, 15)
}

func MuLungDojoMuLungDojo10thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo10thFloorBaseId, 15)
}

func MuLungDojoMuLungDojo11thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo11thFloorBaseId, 15)
}

func MuLungDojoMuLungDojo12thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo12thFloorBaseId, 15)
}

func MuLungDojoMuLungDojo13thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo13thFloorBaseId, 15)
}

func MuLungDojoMuLungDojo14thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo14thFloorBaseId, 15)
}

func MuLungDojoMuLungDojo15thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo15thFloorBaseId, 15)
}

func MuLungDojoMuLungDojo16thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo16thFloorBaseId, 15)
}

func MuLungDojoMuLungDojo17thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo17thFloorBaseId, 15)
}

func MuLungDojoMuLungDojo18thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo18thFloorBaseId, 15)
}

func MuLungDojoMuLungDojo19thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo19thFloorBaseId, 15)
}

func MuLungDojoMuLungDojo20thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo20thFloorBaseId, 15)
}

func MuLungDojoMuLungDojo21stFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo21stFloorBaseId, 15)
}

func MuLungDojoMuLungDojo22ndFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo22ndFloorBaseId, 15)
}

func MuLungDojoMuLungDojo23rdFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo23rdFloorBaseId, 15)
}

func MuLungDojoMuLungDojo24thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo24thFloorBaseId, 15)
}

func MuLungDojoMuLungDojo25thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo25thFloorBaseId, 15)
}

func MuLungDojoMuLungDojo26thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo26thFloorBaseId, 15)
}

func MuLungDojoMuLungDojo27thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo27thFloorBaseId, 15)
}

func MuLungDojoMuLungDojo28thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo28thFloorBaseId, 15)
}

func MuLungDojoMuLungDojo29thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo29thFloorBaseId, 15)
}

func MuLungDojoMuLungDojo30thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo30thFloorBaseId, 15)
}

func MuLungDojoMuLungDojo31stFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo31stFloorBaseId, 15)
}

func MuLungDojoMuLungDojo32ndFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo32ndFloorBaseId, 15)
}

func MuLungDojoMuLungDojo33rdFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo33rdFloorBaseId, 15)
}

func MuLungDojoMuLungDojo34thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo34thFloorBaseId, 15)
}

func MuLungDojoMuLungDojo35thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo35thFloorBaseId, 15)
}

func MuLungDojoMuLungDojo36thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo36thFloorBaseId, 15)
}

func MuLungDojoMuLungDojo37thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo37thFloorBaseId, 15)
}

func MuLungDojoMuLungDojo38thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungDojo38thFloorBaseId, 15)
}

func MuLungDojoMuLungMiniDojo1stFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo1stFloorBaseId, 5)
}

func MuLungDojoMuLungMiniDojo2ndFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo2ndFloorBaseId, 5)
}

func MuLungDojoMuLungMiniDojo3rdFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo3rdFloorBaseId, 5)
}

func MuLungDojoMuLungMiniDojo4thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo4thFloorBaseId, 5)
}

func MuLungDojoMuLungMiniDojo5thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo5thFloorBaseId, 5)
}

func MuLungDojoMuLungMiniDojo6thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo6thFloorBaseId, 5)
}

func MuLungDojoMuLungMiniDojo7thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo7thFloorBaseId, 5)
}

func MuLungDojoMuLungMiniDojo8thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo8thFloorBaseId, 5)
}

func MuLungDojoMuLungMiniDojo9thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo9thFloorBaseId, 5)
}

func MuLungDojoMuLungMiniDojo10thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo10thFloorBaseId, 5)
}

func MuLungDojoMuLungMiniDojo11thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo11thFloorBaseId, 5)
}

func MuLungDojoMuLungMiniDojo12thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo12thFloorBaseId, 5)
}

func MuLungDojoMuLungMiniDojo13thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo13thFloorBaseId, 5)
}

func MuLungDojoMuLungMiniDojo14thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo14thFloorBaseId, 5)
}

func MuLungDojoMuLungMiniDojo15thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo15thFloorBaseId, 5)
}

func MuLungDojoMuLungMiniDojo16thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo16thFloorBaseId, 5)
}

func MuLungDojoMuLungMiniDojo17thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo17thFloorBaseId, 5)
}

func MuLungDojoMuLungMiniDojo18thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo18thFloorBaseId, 5)
}

func MuLungDojoMuLungMiniDojo19thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo19thFloorBaseId, 5)
}

func MuLungDojoMuLungMiniDojo20thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo20thFloorBaseId, 5)
}

func MuLungDojoMuLungMiniDojo21stFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo21stFloorBaseId, 5)
}

func MuLungDojoMuLungMiniDojo22ndFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo22ndFloorBaseId, 5)
}

func MuLungDojoMuLungMiniDojo23rdFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo23rdFloorBaseId, 5)
}

func MuLungDojoMuLungMiniDojo24thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo24thFloorBaseId, 5)
}

func MuLungDojoMuLungMiniDojo25thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo25thFloorBaseId, 5)
}

func MuLungDojoMuLungMiniDojo26thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo26thFloorBaseId, 5)
}

func MuLungDojoMuLungMiniDojo27thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo27thFloorBaseId, 5)
}

func MuLungDojoMuLungMiniDojo28thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo28thFloorBaseId, 5)
}

func MuLungDojoMuLungMiniDojo29thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo29thFloorBaseId, 5)
}

func MuLungDojoMuLungMiniDojo30thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo30thFloorBaseId, 5)
}

func MuLungDojoMuLungMiniDojo31stFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo31stFloorBaseId, 5)
}

func MuLungDojoMuLungMiniDojo32ndFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo32ndFloorBaseId, 5)
}

func MuLungDojoMuLungMiniDojo33rdFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo33rdFloorBaseId, 5)
}

func MuLungDojoMuLungMiniDojo34thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo34thFloorBaseId, 5)
}

func MuLungDojoMuLungMiniDojo35thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo35thFloorBaseId, 5)
}

func MuLungDojoMuLungMiniDojo36thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo36thFloorBaseId, 5)
}

func MuLungDojoMuLungMiniDojo37thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo37thFloorBaseId, 5)
}

func MuLungDojoMuLungMiniDojo38thFloorIds() []Id {
	return GenerateSequentialDungeonIds(MuLungDojoMuLungMiniDojo38thFloorBaseId, 5)
}

func HiddenStreetNettsPyramidIds() []Id {
	return GenerateSequentialDungeonIds(HiddenStreetNettsPyramidBaseId, 4)
}

func HiddenStreetTombOfPharaohYetiIds() []Id {
	return GenerateSequentialDungeonIds(HiddenStreetTombOfPharaohYetiBaseId, 90)
}

// TODO 926010100 - 926023500 HiddenStreetNettsPyramid battle room

func HiddenStreetStage1ManoIds() []Id {
	first := GenerateSequentialDungeonIds(HiddenStreetStage1ManoBase1Id, 12)
	second := GenerateSequentialDungeonIds(HiddenStreetStage1ManoBase2Id, 18)
	return append(first, second...)
}

func HiddenStreetStage2StumpyIds() []Id {
	first := GenerateSequentialDungeonIds(HiddenStreetStage2StumpyBase1Id, 12)
	second := GenerateSequentialDungeonIds(HiddenStreetStage2StumpyBase2Id, 18)
	return append(first, second...)
}

func HiddenStreetStage3DeoIds() []Id {
	first := GenerateSequentialDungeonIds(HiddenStreetStage3DeoBase1Id, 12)
	second := GenerateSequentialDungeonIds(HiddenStreetStage3DeoBase2Id, 18)
	return append(first, second...)
}

func HiddenStreetStage4KingSlimeIds() []Id {
	first := GenerateSequentialDungeonIds(HiddenStreetStage4KingSlimeBase1Id, 12)
	second := GenerateSequentialDungeonIds(HiddenStreetStage4KingSlimeBase2Id, 18)
	return append(first, second...)
}

func HiddenStreetStage5FaustIds() []Id {
	first := GenerateSequentialDungeonIds(HiddenStreetStage5FaustBase1Id, 12)
	second := GenerateSequentialDungeonIds(HiddenStreetStage5FaustBase2Id, 18)
	return append(first, second...)
}

func HiddenStreetStage6KingClangIds() []Id {
	first := GenerateSequentialDungeonIds(HiddenStreetStage6KingClangBase1Id, 12)
	second := GenerateSequentialDungeonIds(HiddenStreetStage6KingClangBase2Id, 18)
	return append(first, second...)
}

func HiddenStreetStage7AlisharIds() []Id {
	first := GenerateSequentialDungeonIds(HiddenStreetStage7AlisharBase1Id, 12)
	second := GenerateSequentialDungeonIds(HiddenStreetStage7AlisharBase2Id, 18)
	return append(first, second...)
}

func HiddenStreetStage8TimerIds() []Id {
	first := GenerateSequentialDungeonIds(HiddenStreetStage8TimerBase1Id, 12)
	second := GenerateSequentialDungeonIds(HiddenStreetStage8TimerBase2Id, 18)
	return append(first, second...)
}

func HiddenStreetStage9MushmomIds() []Id {
	first := GenerateSequentialDungeonIds(HiddenStreetStage9MushmomBase1Id, 12)
	second := GenerateSequentialDungeonIds(HiddenStreetStage9MushmomBase2Id, 18)
	return append(first, second...)
}

func HiddenStreetStage10DyleIds() []Id {
	first := GenerateSequentialDungeonIds(HiddenStreetStage10DyleBase1Id, 12)
	second := GenerateSequentialDungeonIds(HiddenStreetStage10DyleBase2Id, 18)
	return append(first, second...)
}

func HiddenStreetStage11ZenoIds() []Id {
	first := GenerateSequentialDungeonIds(HiddenStreetStage11ZenoBase1Id, 12)
	second := GenerateSequentialDungeonIds(HiddenStreetStage11ZenoBase2Id, 18)
	return append(first, second...)
}

func HiddenStreetStage12NonaTailedFoxIds() []Id {
	first := GenerateSequentialDungeonIds(HiddenStreetStage12NonaTailedFoxBase1Id, 12)
	second := GenerateSequentialDungeonIds(HiddenStreetStage12NonaTailedFoxBase2Id, 18)
	return append(first, second...)
}

func HiddenStreetStage13LordPirateIds() []Id {
	first := GenerateSequentialDungeonIds(HiddenStreetStage13LordPirateBase1Id, 12)
	second := GenerateSequentialDungeonIds(HiddenStreetStage13LordPirateBase2Id, 18)
	return append(first, second...)
}

func HiddenStreetStage14TaeRoonIds() []Id {
	first := GenerateSequentialDungeonIds(HiddenStreetStage14TaeRoonBase1Id, 12)
	second := GenerateSequentialDungeonIds(HiddenStreetStage14TaeRoonBase2Id, 18)
	return append(first, second...)
}

func HiddenStreetStage15PapaPixieIds() []Id {
	first := GenerateSequentialDungeonIds(HiddenStreetStage15PapaPixieBase1Id, 12)
	second := GenerateSequentialDungeonIds(HiddenStreetStage15PapaPixieBase2Id, 18)
	return append(first, second...)
}

func HiddenStreetStage16PriestCatIds() []Id {
	first := GenerateSequentialDungeonIds(HiddenStreetStage16PriestCatBase1Id, 12)
	second := GenerateSequentialDungeonIds(HiddenStreetStage16PriestCatBase2Id, 18)
	return append(first, second...)
}

func HiddenStreetStage17JrBalrogIds() []Id {
	first := GenerateSequentialDungeonIds(HiddenStreetStage17JrBalrogBase1Id, 12)
	second := GenerateSequentialDungeonIds(HiddenStreetStage17JrBalrogBase2Id, 18)
	return append(first, second...)
}

func HiddenStreetStage18FrankenroidIds() []Id {
	first := GenerateSequentialDungeonIds(HiddenStreetStage18FrankenroidBase1Id, 12)
	second := GenerateSequentialDungeonIds(HiddenStreetStage18FrankenroidBase2Id, 18)
	return append(first, second...)
}

func HiddenStreetStage19ElizaIds() []Id {
	first := GenerateSequentialDungeonIds(HiddenStreetStage19ElizaBase1Id, 12)
	second := GenerateSequentialDungeonIds(HiddenStreetStage19ElizaBase2Id, 18)
	return append(first, second...)
}

func HiddenStreetStage20ChimeraIds() []Id {
	first := GenerateSequentialDungeonIds(HiddenStreetStage20ChimeraBase1Id, 12)
	second := GenerateSequentialDungeonIds(HiddenStreetStage20ChimeraBase2Id, 18)
	return append(first, second...)
}

func HiddenStreetStage21SnowmanIds() []Id {
	first := GenerateSequentialDungeonIds(HiddenStreetStage21SnowmanBase1Id, 12)
	second := GenerateSequentialDungeonIds(HiddenStreetStage21SnowmanBase2Id, 18)
	return append(first, second...)
}

func HiddenStreetStage22CrimsonBalrogIds() []Id {
	first := GenerateSequentialDungeonIds(HiddenStreetStage22CrimsonBalrogBase1Id, 12)
	second := GenerateSequentialDungeonIds(HiddenStreetStage22CrimsonBalrogBase2Id, 18)
	return append(first, second...)
}

func HiddenStreetStage23ManonIds() []Id {
	first := GenerateSequentialDungeonIds(HiddenStreetStage23ManonBase1Id, 12)
	second := GenerateSequentialDungeonIds(HiddenStreetStage23ManonBase2Id, 18)
	return append(first, second...)
}

func HiddenStreetStage24GriffeyIds() []Id {
	first := GenerateSequentialDungeonIds(HiddenStreetStage24GriffeyBase1Id, 12)
	second := GenerateSequentialDungeonIds(HiddenStreetStage24GriffeyBase2Id, 18)
	return append(first, second...)
}

func HiddenStreetStage25LeviathanIds() []Id {
	first := GenerateSequentialDungeonIds(HiddenStreetStage25LeviathanBase1Id, 12)
	second := GenerateSequentialDungeonIds(HiddenStreetStage25LeviathanBase2Id, 18)
	return append(first, second...)
}

func HiddenStreetStage26PapulatusIds() []Id {
	first := GenerateSequentialDungeonIds(HiddenStreetStage26PapulatusBase1Id, 12)
	second := GenerateSequentialDungeonIds(HiddenStreetStage26PapulatusBase2Id, 18)
	return append(first, second...)
}

func HiddenStreetStage27PianusIds() []Id {
	first := GenerateSequentialDungeonIds(HiddenStreetStage27PianusBase1Id, 12)
	second := GenerateSequentialDungeonIds(HiddenStreetStage27PianusBase2Id, 18)
	return append(first, second...)
}

func HiddenStreetCheeseStorageIds() []Id {
	return GenerateSequentialDungeonIds(HiddenStreetCheeseStorageBaseId, 10)
}

func SnowIslandToRienIds() []Id {
	return GenerateSequentialDungeonIds(SnowIslandToRienBaseId, 10)
}

func SnowIslandToLithHarborIds() []Id {
	return GenerateSequentialDungeonIds(SnowIslandToLithHarborBaseId, 10)
}

func EmpressRoadToEreveFromOrbisIds() []Id {
	first := GenerateEveryTwoDungeonIds(EmpressRoadToEreveFromOrbisBase1Id, 6)
	second := GenerateEveryTwoDungeonIds(EmpressRoadToEreveFromOrbisBase2Id, 6)
	return append(first, second...)
}

func EmpressRoadToOrbisFromEreveIds() []Id {
	first := GenerateEveryTwoDungeonIds(EmpressRoadToOrbisFromEreveBase1Id, 5)
	second := GenerateEveryTwoDungeonIds(EmpressRoadToOrbisFromEreveBase2Id, 5)
	return append(first, second...)
}

func EmpressRoadToElliniaFromEreveIds() []Id {
	first := GenerateEveryTwoDungeonIds(EmpressRoadToElliniaFromEreveBase1Id, 5)
	second := GenerateEveryTwoDungeonIds(EmpressRoadToElliniaFromEreveBase2Id, 5)
	return append(first, second...)
}

func EmpressRoadToEreveFromElliniaIds() []Id {
	first := GenerateEveryTwoDungeonIds(EmpressRoadToEreveFromElliniaBase1Id, 4)
	second := GenerateEveryTwoDungeonIds(EmpressRoadToEreveFromElliniaBase2Id, 4)
	return append(first, second...)
}
