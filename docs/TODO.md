# Atlas Project TODO

This document tracks planned features and improvements for the Atlas MapleStory server project.

---

## Priority Summary

### Critical (Core Gameplay)
- [ ] **Character Attack Effects** - 27 unimplemented combat mechanics in `character_attack_common.go`
- [ ] **Character Damage Effects** - 10 defensive abilities not processed

### High Priority (Feature Incomplete)
- [ ] **TokenItem Purchasing** - Returns "not implemented" error in NPC shops
- [ ] **Reactor Actions** - Boss weakening, environment manipulation, mass kill sagas
- [ ] **IP/MAC Banning** - Security feature missing in account service

---

## Services

### Account Service
- [ ] Implement IP, MAC, and temporary banning practices (`processor.go:352`)

### Buddies Service
- [ ] Trigger channel request for target when adding buddy (`list/processor.go:216`)
- [ ] Trigger channel request for target when accepting buddy (`list/processor.go:386`)

### Chalkboards Service
- [ ] Ensure character is in a valid location for chalkboard (`chalkboard/processor.go:53`)
- [ ] Ensure character is alive before setting chalkboard (`chalkboard/processor.go:54`)

### Channel Service
- [ ] Handle v83 trailing updateTime for cash item use (`character_cash_item_use.go:50`)
- [ ] Timing issue with loading pre-existing chalkboards
- [ ] Timing issue with loading pre-existing chairs
- [ ] Parties: Party Portals missing. Party member map, level, job, and name changes need to be considered
- [ ] Identify correct compartment type based on character job for cash shop (`cashshop/processor.go:105,150`)
- [ ] Select correct compartment in cash shop entry (`cash_shop_entry.go:58`)
- [ ] Block cash shop entry during: Vega scrolling, events, mini dungeons, already in shop (`cash_shop_entry.go:28-31`)
- [ ] Performance optimization for character queries (`character/processor.go:103,117`)
- [ ] Restrict skill targets to those in range based on bitmap (`skill/handler/common.go:47`)
- [ ] Pet lookup for movement processing (`movement/processor.go:79`)
- [ ] Optimize extra queries in pet consumer (`kafka/consumer/pet/consumer.go:236,274`)
- [ ] Pet skill and item writing (`socket/writer/character_info.go:32`)
- [ ] Query cash shop for whisper targets (`character_chat_whisper.go:72`)
- [ ] Remote channel lookup for whispers (`character_chat_whisper.go:83`)
- [ ] Send rejection to requester for declined invites (`kafka/consumer/invite/consumer.go:138`)
- [ ] Medal name retrieval (`kafka/consumer/message/consumer.go:213`)
- [ ] Server notice on map change failure (`socket/handler/map_change.go:42`)
- [ ] Verify alive and not in mini dungeon for channel change (`channel_change.go:23-24`)
- [ ] Send server notice on channel change failure (`channel_change.go:29`)
- [ ] Validate NPC has ability to move (`npc_action.go:24`)
- [ ] Handle quest-in-progress states in NPC conversations (`npc_continue_conversation.go:24,26,30,39`)
- [ ] Announce guild operation errors (`guild_operation.go:137`)
- [ ] Send buddy operation errors to requester (`buddy_operation.go:47`)
- [ ] NPC producer NpcId population (`npc/producer.go:30,45`)
- [ ] NPC shop commodities model incomplete (`npc/shops/commodities/model.go:69`)
- [ ] Cash shop open loop issue (`socket/writer/cash_shop_open.go:79`)
- [ ] Storage slots in cash shop operation (`socket/writer/cash_shop_operation.go:92`)
- [ ] Cash shop operation padded string (`socket/writer/cash_shop_operation.go:117,119,120`)
- [ ] Guild operation byte value (`socket/writer/guild_operation.go:93`)
- [ ] Buddy operation shop flag (`socket/writer/buddy_operation.go:118`)
- [ ] Set field usage validation (`socket/writer/set_field.go:261`)
- [ ] Multiple services have different cash shop message implementations (`kafka/message/cashshop/kafka.go:72`)

#### Character Attack System (27 unimplemented effects)
Location: `socket/handler/character_attack_common.go:93-119`
- [ ] Apply cooldown
- [ ] Cancel dark sight / wind walk
- [ ] Apply combo orbs (add or consume)
- [ ] Decrease HP from DragonKnight Sacrifice
- [ ] Apply attack effects (heal, MP consumption, dispel, cure all, combo reset)
- [ ] Destroy Chief Bandit exploded mesos
- [ ] Apply Pick Pocket
- [ ] Increase HP from Energy Drain, Vampire, or Drain
- [ ] Apply Bandit Steal
- [ ] Fire Demon ice weaken
- [ ] Ice Demon fire weaken
- [ ] Homing Beacon / Bullseye
- [ ] Flame Thrower
- [ ] Snow Charge
- [ ] Hamstring
- [ ] Slow
- [ ] Blind
- [ ] Paladin / White Knight charges
- [ ] Combo Drain
- [ ] Mortal Blow
- [ ] Three Snails consumption
- [ ] Heavens Hammer
- [ ] ComboTempest
- [ ] BodyPressure
- [ ] Monster Weapon Atk Reflect
- [ ] Monster Magic Atk Reflect
- [ ] Apply MPEater

#### Character Damage System (10 unimplemented effects)
Location: `socket/handler/character_damage.go:23-32`
- [ ] Process Mana Reflection
- [ ] Process Achilles
- [ ] Process Combo Barrier
- [ ] Process Body Pressure
- [ ] Process PowerGuard
- [ ] Process Paladin Divine Shield
- [ ] Process Aran High Defense
- [ ] Process MagicGuard
- [ ] Process MesoGuard
- [ ] Decrease battleship HP

#### Protocol/Version Compatibility
- [ ] Test buddy model with JMS before moving to library (`socket/model/buddy.go:28`)
- [ ] Proper temp stat encoding for GMS v12 (`socket/model/monster.go:205`)
- [ ] Complete skill ID list for skill_usage_info (`socket/model/skill_usage_info.go:65,123,166`)
- [ ] Battle Mage attack info handling (`socket/model/attack_info.go:96,139`)
- [ ] Character model TODOs (`socket/model/character.go:212,221`)
- [ ] Look up actual buff values if riding mount (`socket/model/character.go:481`)
- [ ] Document GMS v83/v95 constants (`socket/writer/character_attack_common.go:41,50,58`)
- [ ] Wild Hunter swallow (`socket/writer/character_attack_common.go:117`)
- [ ] BlazeWizardSpellMastery handling (`socket/writer/character_attack_common.go:157,170`)
- [ ] Clean up character spawn code (`socket/writer/character_spawn.go:75`)
- [ ] Handle GMS-JMS ring encoding differences (`socket/writer/character_spawn.go:100`)
- [ ] Fix crash issues in character effects (`socket/writer/character_effect.go:265,276`)
- [ ] Quest complete communication (`socket/writer/character_effect.go:119`)
- [ ] Write doors for party (`socket/writer/party_operation.go:31,190`)
- [ ] Party operation auto-reject flag (`socket/writer/party_operation.go:130`)
- [ ] Test party operations with JMS (`socket/writer/party_operation.go:199`)
- [ ] JMS map codes for cash shop (`socket/writer/cash_shop_operation.go:128`)
- [ ] Load gifts in cash shop (`socket/writer/cash_shop_operation.go:131`)

#### Set Field Writer
- [ ] Retrieve owner name from id (`socket/writer/set_field.go:418,630`)
- [ ] Create flags bitmask (`socket/writer/set_field.go:419`)
- [ ] Multiple incomplete string writes (`socket/writer/set_field.go:446,512,533,551,631`)

### Character Service
- [ ] Blocked name checking disabled (`processor.go:204`)
- [ ] Determine appropriate drop type and mod (`processor.go:739`)
- [ ] Define AP auto-assign range for Beginner/Noblesse/Legend (`processor.go:1248`)
- [ ] Pre-compute HP and MP to avoid loop cost (`processor.go:1264`)
- [ ] Award job change AP (Cygnus only?) (`processor.go:1461`)

### Consumables Service
- [ ] Consume Vega scroll (`consumable/processor.go:512`)
- [ ] Handle spikes/cursed property (`consumable/processor.go:515`)
- [ ] Improve HP/MP structure (`consumable/processor.go:631`)
- [ ] Equipable producer owner name (`equipable/producer.go:35`)

### Data Service
- [ ] Player NPCs and CPQ support (`map/reader.go:114`)
- [ ] Validate skill reader logic (`skill/reader.go:173`)
- [ ] Handle map chairs (`skill/reader.go:177`)
- [ ] Handle LT in skills (`skill/reader.go:188`)
- [ ] Support mount types: SpaceShip, YetiMount1/2, Broomstick, BalrogMount (`skill/reader.go:209`)
- [ ] WindBreakerFinal statup validation (`skill/reader.go:230`)
- [ ] Weird logic check (`skill/reader.go:250`)
- [ ] Space dash handling (`skill/reader.go:279`)
- [ ] Power explosion handling (`skill/reader.go:292`)
- [ ] Better naming for skill properties (`skill/reader.go:424`)

### Guilds Service
- [ ] Improve guild creation logic (`guild/processor.go:196`)
- [ ] Validate guild name (`guild/processor.go:236`)
- [ ] Respond with failure on guild errors (`guild/processor.go:319`)
- [ ] Proper error handling (`guild/processor.go:482,486`)
- [ ] Second query for party information (`party/rest.go:85`)

### Inventory Service
- [ ] Cash processor incomplete (`cash/processor.go:30`)
- [ ] Equipable model incomplete (`equipable/model.go:111`)
- [ ] Asset owner ID handling (`asset/processor.go:309`)
- [ ] Cash Shop integration for assets (`asset/processor.go:386,392,431,437`)
- [ ] Determine if creating Equip or Cash Equip (`asset/processor.go:595`)
- [ ] Asset owner validation for stacking (`compartment/processor.go:583`)
- [ ] Migrate TransactionId usage (5 locations in `kafka/consumer/compartment/consumer.go:117,132,147,213,228`)
- [ ] TransactionId removal from producers (`compartment/producer.go:62,123,137,152`)
- [ ] Add required fields to drop event (`kafka/consumer/drop/consumer.go:47,52`)

### Invite Service
- [ ] Character deletion should remove pending invites
- [ ] Invites should be able to be queued

### Login Service

#### Error Response Handling
- [ ] Character view all selected PIC errors (`character_view_all_selected_pic.go:33,52,58,65,71`)
- [ ] Register PIC errors (`register_pic.go:36,41`)
- [ ] Accept TOS error (`accept_tos.go:30`)
- [ ] Character view all selected PIC register errors (`character_view_all_selected_pic_register.go:34,53,60,66`)
- [ ] Character view all selected errors (`character_view_all_selected.go:31,50,56`)

#### Other Login TODOs
- [ ] Blocked name checking disabled (`character/processor.go:54`)
- [ ] Terminate on too many PIN attempts (`after_login.go:98`)
- [ ] Clarify gender defaulting logic (`create_character.go:55`)
- [ ] Verify character is not engaged before deletion (`delete_character.go:85`)
- [ ] Verify character is not part of a family before deletion (`delete_character.go:86`)

### Monster Death Service
- [ ] Determine drop type (`monster/processor.go:21`)
- [ ] Party drop distribution (`monster/processor.go:148`)
- [ ] Account for healing (`monster/processor.go:159`)

### NPC Conversations Service
- [ ] Transmit stats in NPC conversations (`kafka/message/npc/kafka.go:78`)
- [ ] Conversation processor incomplete (`conversation/processor.go:560`)

### NPC Shops Service
- [ ] Better transaction handling (`shops/processor.go:396,450`)
- [ ] **Implement TokenItem purchasing** (`shops/processor.go:427`)

### Pets Service
- [ ] Generate cashId if cashId == 0 (`pet/processor.go:199`)

### Portals Service
- [ ] Transmit stats in portal transitions (`character/kafka.go:18`)

### Reactor Actions Service
- [ ] Create saga action for boss weakening (`script/executor.go:230,244`)
- [ ] Create saga action for environment object manipulation (`script/executor.go:251,261`)
- [ ] Create saga action for mass monster killing (`script/executor.go:268,273`)

---

## Libraries

### atlas-constants
- [ ] BladeRecruit job ID handling (`job/model.go:91`)
- [ ] Translated name for FairytaleLandBeanstalkClimb2 (`map/constants.go:1641`)
- [ ] Define HiddenStreet Nett's Pyramid battle room maps (926010100-926023500) (`map/model.go:434`)

---

## Notes

### Summary Statistics
- **Total inline TODOs found**: ~207 across the codebase
- **Most concentrated areas**:
  - Channel Service: ~97 TODOs (socket handlers, writers, models)
  - Inventory Service: ~20 TODOs (asset processing, compartments, Kafka)
  - Login Service: ~20 TODOs (error handling, character operations)
  - Data Service: ~10 TODOs (skill reader, map reader)
  - Character Service: ~5 TODOs (stat calculations, job changes)
